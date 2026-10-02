package term

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"

	"srcspan/srcspan/diagnostic"
	"srcspan/srcspan/files"
)

// A writer that can apply styling for different parts of a diagnostic renderer.
//
// # Implementations
//
//   - [StylesWriter] - custom styles
//   - plain text output without styling is used by [EmitToString],
//     [EmitToIoWrite] and [EmitIntoString] (`PlainWriter` in Rust, which is
//     not public either)
type WriteStyle interface {
	io.Writer

	SetHeader(severity diagnostic.Severity) error
	SetHeaderMessage() error
	SetLineNumber() error
	SetNoteBullet() error
	SetSourceBorder() error
	SetLabel(severity diagnostic.Severity, label_style diagnostic.LabelStyle) error
	Reset() error
}

// A [`WriteStyle`] implementation that ignores all styling
// calls. useful for non color output.
//
// Available on all targets
type plainWriter struct {
	w io.Writer
}

func init() {
	runewidth.DefaultCondition.EastAsianWidth = false
}

func newPlainWriter(w io.Writer) plainWriter {
	return plainWriter{
		w: w,
	}
}

func (w plainWriter) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

func (w plainWriter) Flush() error {
	if f, ok := w.w.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

func (plainWriter) SetHeader(_ diagnostic.Severity) error {
	return nil
}

func (plainWriter) SetHeaderMessage() error {
	return nil
}

func (plainWriter) SetLineNumber() error {
	return nil
}

func (plainWriter) SetNoteBullet() error {
	return nil
}

func (plainWriter) SetSourceBorder() error {
	return nil
}

func (plainWriter) SetLabel(_ diagnostic.Severity, _ diagnostic.LabelStyle) error {
	return nil
}

func (plainWriter) Reset() error {
	return nil
}

func (r *Renderer) Write(p []byte) (n int, err error) {
	return r.writer.Write(p)
}

// The 'location focus' of a source code snippet.
type locus struct {
	// The user-facing name of the file.
	Name string

	// The location.
	Location files.Location
}

// Single-line label, with an optional message.
//
// ```text
// ^^^^^^^^^ blah blah
// ```
type singleLabel struct {
	LabelStyle diagnostic.LabelStyle
	Range      files.Range
	Message    string
}

// A multi-line label to render.
//
// Locations are relative to the start of where the source code is rendered.
type multiLabelTag int

const (
	multiLabelTop multiLabelTag = iota
	multiLabelLeft
	multiLabelBottom
)

type multiLabel struct {
	tag multiLabelTag

	// Multi-line label top.
	// The contained value indicates where the label starts.
	//
	// ```text
	// ╭────────────^
	// ```
	//
	// Can also be rendered at the beginning of the line
	// if there is only whitespace before the label starts.
	//
	// ```text
	// ╭
	// ```
	labelStart int

	// Left vertical labels for multi-line labels.
	//
	// ```text
	// │
	// ```

	// none

	// Multi-line label bottom, with an optional message.
	// The first value indicates where the label ends.
	//
	// ```text
	// ╰────────────^ blah blah
	// ```
	//	Bottom(usize, &'diagnostic str)
	labelEnds  int
	diagnostic string
}

func newMultiLabelTop(labelStart int) multiLabel {
	return multiLabel{
		tag:        multiLabelTop,
		labelStart: labelStart,
	}
}

func newMultiLabelLeft() multiLabel {
	return multiLabel{
		tag: multiLabelLeft,
	}
}

func newMultiLabelBottom(labelEnds int, diagnostic string) multiLabel {
	return multiLabel{
		tag:        multiLabelBottom,
		labelEnds:  labelEnds,
		diagnostic: diagnostic,
	}
}

func (ml multiLabel) isTop() bool {
	return ml.tag == multiLabelTop
}

func (ml multiLabel) asTop() (labelStart int, ok bool) {
	ok = ml.tag == multiLabelTop
	if ok {
		labelStart = ml.labelStart
	}
	return
}

func (ml multiLabel) isLeft() bool {
	return ml.tag == multiLabelLeft
}

func (ml multiLabel) isBottom() bool {
	return ml.tag == multiLabelBottom
}

func (ml multiLabel) asBottom() (labelEnds int, diagnostic string, ok bool) {
	ok = ml.tag == multiLabelBottom
	if ok {
		labelEnds = ml.labelEnds
		diagnostic = ml.diagnostic
	}
	return
}

func (ml multiLabel) String() string {
	switch ml.tag {
	case multiLabelTop:
		return fmt.Sprintf("Top(start=%d)", ml.labelStart)
	case multiLabelLeft:
		return "Left()"
	case multiLabelBottom:
		return fmt.Sprintf("Bottom(labenEnds=%+v, diagnostic=%+v)", ml.labelEnds, ml.diagnostic)
	default:
		panic("")
	}
}

type verticalBound int

const (
	verticalBoundTop verticalBound = iota
	verticalBoundBottom
)

type underlineStyle struct {
	LabelStyle    diagnostic.LabelStyle
	verticalBound verticalBound
}

// Returns the label style of the underline, or nil if there is no underline.
// (`underline.map(|(s, _)| s)` in Rust)
func (u *underlineStyle) labelStyle() *diagnostic.LabelStyle {
	if u == nil {
		return nil
	}
	return &u.LabelStyle
}

/// A renderer of display list entries.
///
/// The following diagram gives an overview of each of the parts of the renderer's output:
///
/// ```text
///                     ┌ outer gutter
///                     │ ┌ left border
///                     │ │ ┌ inner gutter
///                     │ │ │   ┌─────────────────────────── source ─────────────────────────────┐
///                     │ │ │   │                                                                │
///                  ┌────────────────────────────────────────────────────────────────────────────
///        header ── │ error[0001]: oh noes, a cupcake has occurred!
/// snippet start ── │    ┌─ test:9:0
/// snippet empty ── │    │
///  snippet line ── │  9 │   ╭ Cupcake ipsum dolor. Sit amet marshmallow topping cheesecake
///  snippet line ── │ 10 │   │ muffin. Halvah croissant candy canes bonbon candy. Apple pie jelly
///                  │    │ ╭─│─────────^
/// snippet break ── │    · │ │
///  snippet line ── │ 33 │ │ │ Muffin danish chocolate soufflé pastry icing bonbon oat cake.
///  snippet line ── │ 34 │ │ │ Powder cake jujubes oat cake. Lemon drops tootsie roll marshmallow
///                  │    │ │ ╰─────────────────────────────^ blah blah
/// snippet break ── │    · │
///  snippet line ── │ 38 │ │   Brownie lemon drops chocolate jelly-o candy canes. Danish marzipan
///  snippet line ── │ 39 │ │   jujubes soufflé carrot cake marshmallow tiramisu caramels candy canes.
///                  │    │ │           ^^^^^^^^^^^^^^^^^^^ -------------------- blah blah
///                  │    │ │           │
///                  │    │ │           blah blah
///                  │    │ │           note: this is a note
///  snippet line ── │ 40 │ │   Fruitcake jelly-o danish toffee. Tootsie roll pastry cheesecake
///  snippet line ── │ 41 │ │   soufflé marzipan. Chocolate bar oat cake jujubes lollipop pastry
///  snippet line ── │ 42 │ │   cupcake. Candy canes cupcake toffee gingerbread candy canes muffin
///                  │    │ │                                ^^^^^^^^^^^^^^^^^^ blah blah
///                  │    │ ╰──────────^ blah blah
/// snippet break ── │    ·
///  snippet line ── │ 82 │     gingerbread toffee chupa chups chupa chups jelly-o cotton candy.
///                  │    │                 ^^^^^^                         ------- blah blah
/// snippet empty ── │    │
///  snippet note ── │    = blah blah
///  snippet note ── │    = blah blah blah
///                  │      blah blah
///  snippet note ── │    = blah blah blah
///                  │      blah blah
///         empty ── │
/// ```
///
/// > Filler text from <http://www.cupcakeipsum.com>

type Renderer struct {
	writer WriteStyle //io.Writer
	config Config
}

// Construct a renderer from the given writer and config.
func NewRenderer(writer WriteStyle /*io.Writer*/, config Config) Renderer {
	return Renderer{
		writer: writer,
		config: config,
	}
}

func (r *Renderer) chars() Chars {
	return r.config.Chars
}

// Diagnostic header, with severity, code, and message.
//
// ```text
// error[E0001]: unexpected type in `+` application
// ```
func (r *Renderer) RenderHeader(
	locus *locus,
	severity diagnostic.Severity,
	code *string,
	message string,
) (err error) {
	// Write locus
	//
	// ```text
	// test:2:9:
	// ```
	if locus != nil {
		err = r.snippetLocus(locus)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(r, ": ")
		if err != nil {
			return err
		}
	}

	// Write severity name
	//
	// ```text
	// error
	// ```
	//revive:disable:indent-error-flow
	if err = r.SetHeader(severity); err != nil {
		return err
	} else {
		switch severity {
		case diagnostic.Bug:
			_, err = fmt.Fprintf(r, "bug")
			if err != nil {
				return err
			}
		case diagnostic.Error:
			_, err = fmt.Fprintf(r, "error")
			if err != nil {
				return err
			}
		case diagnostic.Warning:
			_, err = fmt.Fprintf(r, "warning")
			if err != nil {
				return err
			}
		case diagnostic.Help:
			_, err = fmt.Fprintf(r, "help")
			if err != nil {
				return err
			}
		case diagnostic.Note:
			_, err = fmt.Fprintf(r, "note")
			if err != nil {
				return err
			}
		default:
			panic(fmt.Sprintf("Unknown diagnostic.Severity: %v", severity))
		}
	}

	// Write error code
	//
	// ```text
	// [E0001]
	// ```
	if code != nil && *code != "" {
		_, err = fmt.Fprintf(r, "[%s]", *code)
		if err != nil {
			return err
		}
	}

	// Write diagnostic message
	//
	// ```text
	// : unexpected type in `+` application
	// ```
	err = r.SetHeaderMessage()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(r, ": %s", message)
	if err != nil {
		return err
	}
	err = r.Reset()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(r, "")
	if err != nil {
		return err
	}

	return nil
}

// Empty line.
func (r *Renderer) RenderEmpty() error {
	_, err := fmt.Fprintf(r, "\n")
	return err
}

// Top left border and locus.
//
// ```text
// ┌─ test:2:9
// ```
func (r *Renderer) RenderSnippetStart(
	outer_padding int,
	locus *locus,
) (err error) {
	err = r.outerGutter(outer_padding)
	if err != nil {
		return err
	}

	err = r.SetSourceBorder()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(r, "%s", r.chars().SnippetStart)
	if err != nil {
		return err
	}

	err = r.Reset()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(r, " ")
	if err != nil {
		return err
	}

	err = r.snippetLocus(locus)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(r, "\n")
	if err != nil {
		return err
	}

	return nil
}

// A line of source code.
//
// ```text
// 10 │   │ muffin. Halvah croissant candy canes bonbon candy. Apple pie jelly
//
//	│ ╭─│─────────^
//
// ```
func (r *Renderer) RenderSnippetSource(
	outer_padding int,
	line_number int,
	source string,
	severity diagnostic.Severity,
	single_labels []singleLabel,
	num_multi_labels int,
	multi_labels []struct {
		LabelIndex int
		LabelStyle diagnostic.LabelStyle
		multiLabel multiLabel
	},
) (err error) {
	// Trim trailing newlines, linefeeds, and null chars from source, if they exist.
	// FIXME: Use the number of trimmed placeholders when rendering single line carets

	//         let source = source.trim_end_matches(['\n', '\r', '\0'].as_ref());
	source = strings.TrimRight(source, "\n\r\x00")

	// Write source line
	//
	// ```text
	// 10 │   │ muffin. Halvah croissant candy canes bonbon candy. Apple pie jelly
	// ```

	{
		// Write outer gutter (with line number) and border
		err = r.outerGutterNumber(line_number, outer_padding)
		if err != nil {
			return err
		}
		err = r.borderLeft()
		if err != nil {
			return err
		}
		// Write inner gutter (with multi-line continuations on the left if necessary)
		//		fmt.Printf(">>%d<<", num_multi_labels)
		multi_labels_idx := 0
		for label_column := 0; label_column < num_multi_labels; label_column++ { //nolint:intrange // emulate rust code
			if multi_labels_idx < len(multi_labels) {
				ml := multi_labels[multi_labels_idx]
				if ml.LabelIndex == label_column {
					if start, ok := ml.multiLabel.asTop(); ok {
						if start <= leadingWhitespaceLen(source) {
							err = r.labelMultiTopLeft(severity, ml.LabelStyle)
							if err != nil {
								return err
							}
						} else {
							err = r.innerGutterSpace()
							if err != nil {
								return err
							}
						}
					} else if ml.multiLabel.isLeft() || ml.multiLabel.isBottom() {
						err = r.labelMultiLeft(severity, ml.LabelStyle, nil)
						if err != nil {
							return err
						}
					}
					multi_labels_idx++
				} else {
					err = r.innerGutterSpace()
					if err != nil {
						return err
					}
				}
			} else {
				err = r.innerGutterSpace()
				if err != nil {
					return err
				}
			}
		}
		// Write source text
		_, err = fmt.Fprintf(r, " ")
		if err != nil {
			return err
		}

		in_primary := false

		for _, cm := range r.charMetrics(charIndices(source)) {
			metrics := cm.charMetric
			ch := cm.Char

			column_range := files.NewRange(metrics.byte_index, metrics.byte_index+utf8.RuneLen(ch))

			// Check if we are overlapping a primary label
			var is_primary bool
			for _, sl := range single_labels {
				if sl.LabelStyle == diagnostic.Primary && isOverlapping(sl.Range, column_range) {
					is_primary = true
				}
			}
			for _, ml := range multi_labels {
				if ml.LabelStyle == diagnostic.Primary {
					if start, ok := ml.multiLabel.asTop(); ok {
						is_primary = is_primary || column_range.Start >= start
					} else if ml.multiLabel.isLeft() {
						is_primary = true
					} else if start, _, ok := ml.multiLabel.asBottom(); ok {
						is_primary = is_primary || column_range.End <= start
					}
				}
			}

			// Set the source color if we are in a primary label
			if is_primary && !in_primary {
				err = r.SetLabel(severity, diagnostic.Primary)
				if err != nil {
					return err
				}
				in_primary = true
			} else if !is_primary && in_primary {
				err = r.Reset()
				if err != nil {
					return err
				}
				in_primary = false
			}

			switch ch {
			case '\t':
				for i := 0; i < metrics.unicode_width; i++ { //nolint:intrange // emulate rust code
					_, err = fmt.Fprintf(r, " ")
					if err != nil {
						return err
					}
				}
			default:
				_, err = fmt.Fprintf(r, "%c", ch)
				if err != nil {
					return err
				}
			}
		}

		if in_primary {
			err = r.Reset()
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(r, "\n")
		if err != nil {
			return err
		}
	}

	// Write single labels underneath source
	//
	// ```text
	//   │     - ---- ^^^ second mutable borrow occurs here
	//   │     │ │
	//   │     │ first mutable borrow occurs here
	//   │     first borrow later used by call
	//   │     help: some help here
	// ```
	if len(single_labels) != 0 {
		// Our plan is as follows:
		//
		// 1. Do an initial scan to find:
		//    - The number of non-empty messages.
		//    - The right-most start and end positions of labels.
		//    - A candidate for a trailing label (where the label's message
		//      is printed to the left of the caret).
		// 2. Check if the trailing label candidate overlaps another label -
		//    if so we print it underneath the carets with the other labels.
		// 3. Print a line of carets, and (possibly) the trailing message
		//    to the left.
		// 4. Print vertical lines pointing to the carets, and the messages
		//    for those carets.
		//
		// We try our best avoid introducing new dynamic allocations,
		// instead preferring to iterate over the labels multiple times. It
		// is unclear what the performance tradeoffs are however, so further
		// investigation may be required.

		// The number of non-empty messages to print.
		num_messages := 0

		// The right-most start position, eg:
		//
		// ```text
		// -^^^^---- ^^^^^^^
		//           │
		//           right-most start position
		// ```
		max_label_start := 0

		// The right-most end position, eg:
		//
		// ```text
		// -^^^^---- ^^^^^^^
		//                 │
		//                 right-most end position
		// ```
		max_label_end := 0

		// A trailing message, eg:
		//
		// ```text
		// ^^^ second mutable borrow occurs here
		// ```
		var trailing_label *struct {
			Index       int
			singleLabel singleLabel
		}

		for label_index, label := range single_labels {
			if len(label.Message) != 0 {
				num_messages += 1
			}

			max_label_start = max(max_label_start, label.Range.Start)
			max_label_end = max(max_label_end, label.Range.End)

			// This is a candidate for the trailing label, so let's record it.
			if label.Range.End == max_label_end {
				if label.Message == "" {
					trailing_label = nil
				} else {
					label := &struct {
						Index       int
						singleLabel singleLabel
					}{
						Index:       label_index,
						singleLabel: label,
					}

					trailing_label = label
				}
			}
		}

		if trailing_label != nil {
			// Check to see if the trailing label candidate overlaps any of
			// the other labels on the current line.
			for label_index, sl := range single_labels {
				if label_index != trailing_label.Index {
					if isOverlapping(trailing_label.singleLabel.Range, sl.Range) {
						// If it does, we'll instead want to render it below the
						// carets along with the other hanging labels.
						trailing_label = nil
						break
					}
				}
			}
		}

		// Write a line of carets
		//
		// ```text
		//   │ ^^^^^^  -------^^^^^^^^^-------^^^^^----- ^^^^ trailing label message
		// ```

		err = r.outerGutter(outer_padding)
		if err != nil {
			return err
		}
		err = r.borderLeft()
		if err != nil {
			return err
		}
		err = r.innerGutter(severity, num_multi_labels, multi_labels)
		if err != nil {
			return err
		}
		//             write!(self, " ")?;
		_, err = fmt.Fprintf(r, " ")
		if err != nil {
			return err
		}

		var previous_label_style *diagnostic.LabelStyle
		placeholder_metrics := charMetric{
			byte_index:    len(source),
			unicode_width: 1,
		}

		for _, cm := range append(r.charMetrics(charIndices(source)),
			// Add a placeholder source column at the end to allow for
			// printing carets at the end of lines, eg:
			//
			// ```text
			// 1 │ Hello world!
			//   │             ^
			// ```
			struct {
				charMetric charMetric
				Char       rune
			}{
				charMetric: placeholder_metrics,
				Char:       '\000',
			}) {
			metrics := cm.charMetric
			ch := cm.Char

			// Find the current label style at this column
			column_range := files.NewRange(metrics.byte_index, metrics.byte_index+utf8.RuneLen(ch))

			var current_label_style *diagnostic.LabelStyle
			var max_priority int
			for _, sl := range single_labels {
				if isOverlapping(sl.Range, column_range) {
					if max_priority <= labelPriorityKey(&sl.LabelStyle) {
						max_priority = labelPriorityKey(&sl.LabelStyle)
						current_label_style = &sl.LabelStyle
					}
				}
			}

			// Update writer style if necessary
			if !sameLabelStyle(previous_label_style, current_label_style) {
				if current_label_style == nil {
					err = r.Reset()
					if err != nil {
						return err
					}
				} else {
					err = r.SetLabel(severity, *current_label_style)
					if err != nil {
						return err
					}
				}
			}

			var caret_ch *rune
			if current_label_style != nil {
				switch *current_label_style {
				case diagnostic.Primary:
					ch := r.chars().SinglePrimaryCaret
					caret_ch = &ch
				case diagnostic.Secondary:
					ch := r.chars().SingleSecondaryCaret
					caret_ch = &ch
				default:
					panic(fmt.Sprintf("Unknown diagnostic.LabelStyle: %v", *current_label_style))
				}
			} else {
				// Only print padding if we are before the end of the last single line caret
				if metrics.byte_index < max_label_end {
					ch := ' '
					caret_ch = &ch
				} else {
					caret_ch = nil
				}
			}

			if caret_ch != nil {
				// FIXME: improve rendering of carets between character boundaries
				for i := 0; i < metrics.unicode_width; i++ { //nolint:intrange // emulate rust code
					_, err = fmt.Fprintf(r, "%c", *caret_ch)
					if err != nil {
						return err
					}
				}
			}

			previous_label_style = current_label_style
		}

		// Reset style if it was previously set
		if previous_label_style != nil {
			err = r.Reset()
			if err != nil {
				return err
			}
		}

		// Write first trailing label message
		if trailing_label != nil {
			_, err = fmt.Fprintf(r, " ")
			if err != nil {
				return err
			}

			err = r.SetLabel(severity, trailing_label.singleLabel.LabelStyle)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(r, "%s", trailing_label.singleLabel.Message)
			if err != nil {
				return err
			}

			err = r.Reset()
			if err != nil {
				return err
			}
		}

		_, err = fmt.Fprintln(r, "")
		if err != nil {
			return err
		}

		// Write hanging labels pointing to carets
		//
		// ```text
		//   │     │ │
		//   │     │ first mutable borrow occurs here
		//   │     first borrow later used by call
		//   │     help: some help here
		// ```
		num_trailing_label := 0
		if trailing_label != nil {
			num_trailing_label = 1
		}

		if num_messages > num_trailing_label {
			// Write first set of vertical lines before hanging labels
			//
			// ```text
			//   │     │ │
			// ```
			err = r.outerGutter(outer_padding)
			if err != nil {
				return err
			}
			err = r.borderLeft()
			if err != nil {
				return err
			}
			err = r.innerGutter(severity, num_multi_labels, multi_labels)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(r, " ")
			if err != nil {
				return err
			}

			err = r.caretPointers(
				severity,
				max_label_start,
				single_labels,
				trailing_label,
				charIndices(source),
			)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(r, "")
			if err != nil {
				return err
			}

			// Write hanging labels pointing to carets
			//
			// ```text
			//   │     │ first mutable borrow occurs here
			//   │     first borrow later used by call
			//   │     help: some help here
			// ```
			hanging_labels := hangingLabels(single_labels, trailing_label)
			slices.Reverse(hanging_labels)

			for _, sl := range hanging_labels {
				label_style := sl.LabelStyle
				range_ := sl.Range
				message := sl.Message

				err = r.outerGutter(outer_padding)
				if err != nil {
					return err
				}

				err = r.borderLeft()
				if err != nil {
					return err
				}

				err = r.innerGutter(severity, num_multi_labels, multi_labels)
				if err != nil {
					return err
				}

				_, err = fmt.Fprintf(r, " ")
				if err != nil {
					return err
				}

				//.take_while(|(byte_index, _)| *byte_index < range.start),
				char_indices := charIndices(source)
				last_i := 0
				for i, ci := range char_indices {
					if ci.Index < range_.Start {
						continue
					} else {
						last_i = i
						break
					}
				}
				char_indices = char_indices[0:last_i]

				err = r.caretPointers(
					severity,
					max_label_start,
					single_labels,
					trailing_label,
					char_indices,
				)
				if err != nil {
					return err
				}

				err = r.SetLabel(severity, label_style)
				if err != nil {
					return err
				}

				_, err = fmt.Fprintf(r, "%s", message)
				if err != nil {
					return err
				}

				err = r.Reset()
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(r, "")
				if err != nil {
					return err
				}
			}
		}
	}

	// Write top or bottom label carets underneath source
	//
	// ```text
	//     │ ╰───│──────────────────^ woops
	//     │   ╭─│─────────^
	// ```
	var label_style diagnostic.LabelStyle
	var range_ int
	var bottom_message *string

	for multi_label_index, ml := range multi_labels {
		//_label_index := ml.LabelIndex
		_label_style := ml.LabelStyle
		_label := ml.multiLabel

		if _label.isLeft() {
			// no label caret needed
			continue
		}
		if start, ok := _label.asTop(); ok {
			// no label caret needed if this can be started in front of the line
			if start <= leadingWhitespaceLen(source) {
				continue
			} else {
				label_style = _label_style
				range_ = start
				bottom_message = nil
			}
		}
		if _range_, message, ok := _label.asBottom(); ok {
			label_style = _label_style
			range_ = _range_
			bottom_message = &message
		}

		err = r.outerGutter(outer_padding)
		if err != nil {
			return err
		}
		err = r.borderLeft()
		if err != nil {
			return err
		}

		// Write inner gutter.
		//
		// ```text
		//  │ ╭─│───│
		// ```
		var underline *underlineStyle
		multi_labels_idx := 0
		for label_column := 0; label_column < num_multi_labels; label_column++ { //nolint:intrange // emulate rust code
			if multi_labels_idx < len(multi_labels) {
				ml := multi_labels[multi_labels_idx]

				i := multi_labels_idx
				label_index := ml.LabelIndex
				ls := ml.LabelStyle
				label := ml.multiLabel

				if label_index == label_column {
					if label.isLeft() {
						err = r.labelMultiLeft(severity, ls, underline.labelStyle())
						if err != nil {
							return err
						}
					} else if _, ok := label.asTop(); ok && multi_label_index > i {
						err = r.labelMultiLeft(severity, ls, underline.labelStyle())
						if err != nil {
							return err
						}
					} else if _, _, ok := label.asBottom(); ok && multi_label_index < i {
						err = r.labelMultiLeft(severity, ls, underline.labelStyle())
						if err != nil {
							return err
						}
					} else if _, ok := label.asTop(); ok && multi_label_index == i {
						underline = &underlineStyle{
							LabelStyle:    ls,
							verticalBound: verticalBoundTop,
						}
						err = r.labelMultiTopLeft(severity, label_style)
						if err != nil {
							return err
						}
					} else if _, _, ok := label.asBottom(); ok && multi_label_index == i {
						underline = &underlineStyle{
							LabelStyle:    ls,
							verticalBound: verticalBoundBottom,
						}
						err = r.labelMultiBottomLeft(severity, label_style)
						if err != nil {
							return err
						}
					} else if label.isTop() || label.isBottom() {
						err = r.innerGutterColumn(severity, underline)
						if err != nil {
							return err
						}
					}
					multi_labels_idx++
				} else {
					err = r.innerGutterColumn(severity, underline)
					if err != nil {
						return err
					}
				}
			} else {
				err = r.innerGutterColumn(severity, underline)
				if err != nil {
					return err
				}
			}
		}

		// Finish the top or bottom caret
		if bottom_message == nil {
			err = r.labelMultiTopCaret(severity, label_style, source, range_)
			if err != nil {
				return err
			}
		} else {
			err = r.labelMultiBottomCaret(severity, label_style, source, range_, *bottom_message)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// An empty source line, for providing additional whitespace to source snippets.
//
// ```text
// │ │ │
// ```
func (r *Renderer) RenderSnippetEmpty(
	outer_padding int,
	severity diagnostic.Severity,
	num_multi_labels int,
	multi_labels []struct {
		LabelIndex int
		LabelStyle diagnostic.LabelStyle
		multiLabel multiLabel
	},
) (err error) {
	err = r.outerGutter(outer_padding)
	if err != nil {
		return
	}
	err = r.borderLeft()
	if err != nil {
		return
	}
	err = r.innerGutter(severity, num_multi_labels, multi_labels)
	if err != nil {
		return
	}

	_, err = fmt.Fprintln(r, "")
	return
}

// A broken source line, for labeling skipped sections of source.
//
// ```text
// · │ │
// ```
func (r *Renderer) RenderSnippetBreak(
	outer_padding int,
	severity diagnostic.Severity,
	num_multi_labels int,
	multi_labels []struct {
		LabelIndex int
		LabelStyle diagnostic.LabelStyle
		multiLabel multiLabel
	},
) (err error) {
	err = r.outerGutter(outer_padding)
	if err != nil {
		return err
	}
	err = r.borderLeftBreak()
	if err != nil {
		return err
	}
	err = r.innerGutter(severity, num_multi_labels, multi_labels)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(r)
	return err
}

// Additional notes.
//
// ```text
// = expected type `Int`
//
//	found type `String`
//
// ```
func (r *Renderer) RenderSnippetNote(
	outer_padding int,
	message string,
) (err error) {
	for note_line_index, line := range lines(message) {
		err = r.outerGutter(outer_padding)
		if err != nil {
			return err
		}

		switch note_line_index {
		case 0:
			err = r.SetNoteBullet()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(r, "%c", r.chars().NoteBullet)
			if err != nil {
				return err
			}

			err = r.Reset()
			if err != nil {
				return err
			}
		default:
			_, err = fmt.Fprintf(r, " ")
			if err != nil {
				return err
			}
		}

		// Write line of message
		_, err = fmt.Fprintf(r, " %s\n", line)
		if err != nil {
			return err
		}
	}

	return err
}

// Adds tab-stop aware unicode-width computations to an iterator over
// character indices. Assumes that the character indices begin at the start
// of the line.
func (r *Renderer) charMetrics(
	char_indices []struct {
		Index int
		Char  rune
	},
) []struct {
	charMetric charMetric
	Char       rune
} {
	tab_width := r.config.TabWidth
	unicode_column := 0

	var char_metrics []struct {
		charMetric charMetric
		Char       rune
	}

	for _, ci := range char_indices {
		var uw int

		if ci.Char == '\t' {
			if tab_width == 0 {
				// Guard divide-by-zero
				uw = 0
			} else {
				uw = tab_width - (unicode_column % tab_width)
			}
		} else {
			uw = runewidth.RuneWidth(ci.Char)
		}

		metrics := charMetric{
			byte_index:    ci.Index,
			unicode_width: uw,
		}
		unicode_column += metrics.unicode_width

		char_metrics = append(char_metrics,
			struct {
				charMetric charMetric
				Char       rune
			}{
				charMetric: metrics,
				Char:       ci.Char,
			},
		)
	}

	return char_metrics
}

// Location focus.
func (r *Renderer) snippetLocus(locus *locus) (err error) {
	// Location.{Line,Column} is Index or 0-based.
	_, err = fmt.Fprintf(r, "%s:%d:%d", locus.Name, locus.Location.LineNumber, locus.Location.ColumnNumber)
	return
}

// The outer gutter of a source line.
func (r *Renderer) outerGutter(
	outer_padding int,
) (err error) {
	_, err = fmt.Fprintf(r, "%*s ", outer_padding, "")
	return
}

// The outer gutter of a source line, with line number.
func (r *Renderer) outerGutterNumber(
	line_number int,
	outer_padding int,
) (err error) {
	err = r.SetLineNumber()
	if err != nil {
		return
	}

	_, err = fmt.Fprintf(r, "%*d", outer_padding, line_number)
	if err != nil {
		return
	}

	err = r.Reset()
	if err != nil {
		return
	}

	_, err = fmt.Fprintf(r, " ")
	return
}

// The left-hand border of a source line.
func (r *Renderer) borderLeft() (err error) {
	err = r.SetSourceBorder()
	if err != nil {
		return
	}

	_, err = fmt.Fprintf(r, "%c", r.chars().SourceBorderLeft)
	if err != nil {
		return
	}

	err = r.Reset()
	if err != nil {
		return
	}
	return
}

// The broken left-hand border of a source line.
func (r *Renderer) borderLeftBreak() (err error) {
	err = r.SetSourceBorder()
	if err != nil {
		return
	}

	_, err = fmt.Fprintf(r, "%c", r.chars().SourceBorderLeftBreak)
	if err != nil {
		return
	}

	err = r.Reset()
	if err != nil {
		return
	}

	return
}

// Write vertical lines pointing to carets.
func (r *Renderer) caretPointers(
	severity diagnostic.Severity,
	max_label_start int,
	single_labels []singleLabel,
	trailing_label *struct {
		Index       int
		singleLabel singleLabel
	},
	char_indices []struct {
		Index int
		Char  rune
	},
) (err error) {
	for _, cm := range r.charMetrics(char_indices) {
		metrics := cm.charMetric
		ch := cm.Char

		column_range_start := metrics.byte_index
		column_range_end := metrics.byte_index + utf8.RuneLen(ch)

		var label_style *diagnostic.LabelStyle
		var max_priority int
		for _, sl := range hangingLabels(single_labels, trailing_label) {
			if column_range_start <= sl.Range.Start && sl.Range.Start < column_range_end {
				if max_priority <= labelPriorityKey(&sl.LabelStyle) {
					max_priority = labelPriorityKey(&sl.LabelStyle)
					label_style = &sl.LabelStyle
				}
			}
		}

		var spaces_start int
		var spaces_end int
		if label_style == nil {
			spaces_start = 0
			spaces_end = metrics.unicode_width
		} else {
			err = r.SetLabel(severity, *label_style)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(r, "%c", r.chars().PointerLeft)
			if err != nil {
				return err
			}
			err = r.Reset()
			if err != nil {
				return err
			}

			spaces_start = 1
			spaces_end = metrics.unicode_width
		}

		// Only print padding if we are before the end of the last single line caret
		if metrics.byte_index <= max_label_start {
			for i := spaces_start; i < spaces_end; i++ {
				_, err = fmt.Fprintf(r, " ")
				if err != nil {
					return err
				}
			}
		}
	}

	return err
}

// The left of a multi-line label.
//
// ```text
//
//	│
//
// ```
func (r *Renderer) labelMultiLeft(
	severity diagnostic.Severity,
	label_style diagnostic.LabelStyle,
	underline *diagnostic.LabelStyle,
) (err error) {
	if underline == nil {
		_, err = fmt.Fprintf(r, " ")
		if err != nil {
			return err
		}
	} else {
		// Continue an underline horizontally
		// (`Some(label_style)` in Rust shadows the parameter only in this
		// branch, so the parameter must not be overwritten here.)
		err = r.SetLabel(severity, *underline)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(r, "%c", r.chars().MultiTop)
		if err != nil {
			return err
		}

		err = r.Reset()
		if err != nil {
			return err
		}
	}

	err = r.SetLabel(severity, label_style)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(r, "%c", r.chars().MultiLeft)
	if err != nil {
		return err
	}

	err = r.Reset()
	if err != nil {
		return err
	}

	return err
}

// The top-left of a multi-line label.
//
// ```text
//
//	╭
//
// ```
func (r *Renderer) labelMultiTopLeft(
	severity diagnostic.Severity,
	label_style diagnostic.LabelStyle,
) (err error) {
	_, err = fmt.Fprintf(r, " ")
	if err != nil {
		return
	}

	err = r.SetLabel(severity, label_style)
	if err != nil {
		return
	}

	_, err = fmt.Fprintf(r, "%c", r.chars().MultiTopLeft)
	if err != nil {
		return
	}

	err = r.Reset()
	if err != nil {
		return
	}

	return
}

// The bottom left of a multi-line label.
//
// ```text
//
//	╰
//
// ```
func (r *Renderer) labelMultiBottomLeft(
	severity diagnostic.Severity,
	label_style diagnostic.LabelStyle,
) (err error) {
	_, err = fmt.Fprintf(r, " ")
	if err != nil {
		return
	}

	err = r.SetLabel(severity, label_style)
	if err != nil {
		return
	}

	_, err = fmt.Fprintf(r, "%c", r.chars().MultiBottomLeft)
	if err != nil {
		return
	}

	err = r.Reset()
	if err != nil {
		return
	}

	return
}

// Multi-line label top.
//
// ```text
// ─────────────^
// ```
func (r *Renderer) labelMultiTopCaret(
	severity diagnostic.Severity,
	label_style diagnostic.LabelStyle,
	source string,
	start int,
) (err error) {
	err = r.SetLabel(severity, label_style)
	if err != nil {
		return err
	}

	for _, cm := range r.charMetrics(charIndices(source)) {
		metrics := cm.charMetric

		// `take_while(|(metrics, _)| metrics.byte_index < start + 1)` in Rust:
		// the extra column makes up for the space between the gutter and
		// the source, so that the caret is under the first labeled char.
		if metrics.byte_index >= start+1 {
			break
		}

		// FIXME: improve rendering of carets between character boundaries
		for i := 0; i < metrics.unicode_width; i++ { //nolint:intrange // emulate rust code
			_, err = fmt.Fprintf(r, "%c", r.chars().MultiTop)
			if err != nil {
				return err
			}
		}
	}

	var caret_start rune
	switch label_style {
	case diagnostic.Primary:
		caret_start = r.config.Chars.MultiPrimaryCaretStart
	case diagnostic.Secondary:
		caret_start = r.config.Chars.MultiSecondaryCaretStart
	default:
		panic(fmt.Sprintf("Unknown diagnostic.LabelStyle: %v", label_style))
	}

	_, err = fmt.Fprintf(r, "%c", caret_start)
	if err != nil {
		return err
	}
	err = r.Reset()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(r, "")
	if err != nil {
		return err
	}

	return err
}

// Multi-line label bottom, with a message.
//
// ```text
// ─────────────^ expected `Int` but found `String`
// ```
func (r *Renderer) labelMultiBottomCaret(
	severity diagnostic.Severity,
	label_style diagnostic.LabelStyle,
	source string,
	start int,
	message string,
) (err error) {
	err = r.SetLabel(severity, label_style)
	if err != nil {
		return err
	}

	for _, cm := range r.charMetrics(charIndices(source)) {
		metrics := cm.charMetric

		if metrics.byte_index >= start {
			break
		}

		// FIXME: improve rendering of carets between character boundaries
		for i := 0; i < metrics.unicode_width; i++ { //nolint:intrange // emulate rust code
			_, err = fmt.Fprintf(r, "%c", r.chars().MultiBottom)
			if err != nil {
				return err
			}
		}
	}

	// NOTE: codespan-reporting 0.13.1 uses `multi_*_caret_start` here,
	// which looks like a bug: `multi_*_caret_end` are documented as
	// "marking the end of a multi-line label" but are never used.
	var caret_end rune
	switch label_style {
	case diagnostic.Primary:
		caret_end = r.config.Chars.MultiPrimaryCaretEnd
	case diagnostic.Secondary:
		caret_end = r.config.Chars.MultiSecondaryCaretEnd
	default:
		panic(fmt.Sprintf("Unknown diagnostic.LabelStyle: %v", label_style))
	}

	_, err = fmt.Fprintf(r, "%c", caret_end)
	if err != nil {
		return err
	}

	if message != "" {
		_, err = fmt.Fprintf(r, " %s", message)
		if err != nil {
			return err
		}
	}

	err = r.Reset()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(r, "")
	if err != nil {
		return err
	}

	return err
}

// Writes an empty gutter space, or continues an underline horizontally.
func (r *Renderer) innerGutterColumn(
	severity diagnostic.Severity,
	underline *underlineStyle) (err error) {
	if underline == nil {
		err = r.innerGutterSpace()
	} else {
		label_style := underline.LabelStyle
		vertical_bound := underline.verticalBound

		err = r.SetLabel(severity, label_style)
		if err != nil {
			return err
		}

		var ch rune
		switch vertical_bound {
		case verticalBoundTop:
			ch = r.config.Chars.MultiTop
		case verticalBoundBottom:
			ch = r.config.Chars.MultiBottom
		default:
			panic(fmt.Sprintf("Unknown term.verticalBound: %v", vertical_bound))
		}

		_, err = fmt.Fprintf(r, "%c%c", ch, ch)
		if err != nil {
			return err
		}

		err = r.Reset()
		if err != nil {
			return err
		}
	}
	return err
}

// Writes an empty gutter space.
func (r *Renderer) innerGutterSpace() (err error) {
	_, err = fmt.Fprintf(r, "  ")
	return
}

// Writes an inner gutter, with the left lines if necessary.
func (r *Renderer) innerGutter(
	severity diagnostic.Severity,
	num_multi_labels int,
	multi_labels []struct {
		LabelIndex int
		LabelStyle diagnostic.LabelStyle
		multiLabel multiLabel
	},
) (err error) {
	multi_labels_idx := 0

	for label_column := 0; label_column < num_multi_labels; label_column++ { //nolint:intrange // emulate rust code
		if multi_labels_idx < len(multi_labels) {
			ml := multi_labels[multi_labels_idx]

			if ml.LabelIndex == label_column {
				if ml.multiLabel.isLeft() || ml.multiLabel.isBottom() {
					err = r.labelMultiLeft(severity, ml.LabelStyle, nil)
					if err != nil {
						return err
					}
					multi_labels_idx++
				} else if ml.multiLabel.isTop() {
					err = r.innerGutterSpace()
					if err != nil {
						return err
					}
					multi_labels_idx++
				}
			} else {
				err = r.innerGutterSpace()
				if err != nil {
					return err
				}
			}
		} else {
			err = r.innerGutterSpace()
			if err != nil {
				return err
			}
		}
	}

	return err
}

func (r *Renderer) SetHeader(severity diagnostic.Severity) error {
	return r.writer.SetHeader(severity)
}

func (r *Renderer) SetHeaderMessage() error {
	return r.writer.SetHeaderMessage()
}

func (r *Renderer) SetLineNumber() error {
	return r.writer.SetLineNumber()
}

func (r *Renderer) SetNoteBullet() error {
	return r.writer.SetNoteBullet()
}

func (r *Renderer) SetSourceBorder() error {
	return r.writer.SetSourceBorder()
}

func (r *Renderer) SetLabel(severity diagnostic.Severity, label_style diagnostic.LabelStyle) error {
	return r.writer.SetLabel(severity, label_style)
}

func (r *Renderer) Reset() error {
	return r.writer.Reset()
}

type charMetric struct {
	byte_index    int
	unicode_width int
}

// Check if two ranges overlap
// Returns the byte length of the leading whitespace in `source`.
// (`source.len() - source.trim_start().len()` in Rust)
func leadingWhitespaceLen(source string) int {
	return len(source) - len(strings.TrimLeftFunc(source, unicode.IsSpace))
}

// Reports whether two optional label styles are equal.
// (`Option<LabelStyle>` comparison in Rust)
func sameLabelStyle(a, b *diagnostic.LabelStyle) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// Splits `s` into lines like `str::lines()` in Rust: a trailing newline
// does not produce an empty last line, and a trailing `\r` is removed
// from each line.
func lines(s string) []string {
	if s == "" {
		return nil
	}

	ls := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	for i, l := range ls {
		ls[i] = strings.TrimSuffix(l, "\r")
	}
	return ls
}

func isOverlapping(r0, r1 files.Range) bool {
	start := max(r0.Start, r1.Start)
	end := min(r0.End, r1.End)
	return start < end
}

// For prioritizing primary labels over secondary labels when rendering carets.
func labelPriorityKey(label_style *diagnostic.LabelStyle) int {
	switch *label_style {
	case diagnostic.Secondary:
		return 0
	case diagnostic.Primary:
		return 1
	default:
		panic(fmt.Sprintf("Unknown diagnostic.LabelStyle: %v", *label_style))
	}
}

// Return an iterator that yields the labels that require hanging messages
// rendered underneath them.
func hangingLabels(
	single_labels []singleLabel,
	trailing_label *struct {
		Index       int
		singleLabel singleLabel
	},
) []singleLabel {
	labels := []singleLabel{}

	for i, label := range single_labels {
		if label.Message == "" {
			continue
		}
		if trailing_label != nil && trailing_label.Index == i {
			continue
		}

		labels = append(labels, label)
	}

	return labels
}

func charIndices(s string) []struct {
	Index int
	Char  rune
} {
	char_indices := []struct {
		Index int
		Char  rune
	}{}

	for i, r := range s {
		char_indices = append(char_indices, struct {
			Index int
			Char  rune
		}{
			Index: i,
			Char:  r,
		})
	}

	return char_indices
}
