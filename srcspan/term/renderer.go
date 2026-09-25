package term

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/mattn/go-runewidth"

	"srcspan/srcspan"
	"srcspan/srcspan/diagnostic"
)

// A writer that can apply styling for different parts of a diagnostic renderer.
//
// # Implementations
//
// - [`PlainWriter<W>`](PlainWriter) - no-op styling, plain text output
// - [`StylesWriter<W>`](crate::term::StylesWriter) - custom styles (requires `termcolor` feature)
// - `T: WriteColor` - blanket impl using default styles (requires `termcolor` feature)
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
type PlainWriter struct {
	w io.Writer
}

func init() {
	runewidth.DefaultCondition.EastAsianWidth = false
}

func NewPlainWriter(w io.Writer) PlainWriter {
	return PlainWriter{
		w: w,
	}
}

func (w PlainWriter) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

func (w PlainWriter) Flush() error {
	if f, ok := w.w.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

func (PlainWriter) SetHeader(_ diagnostic.Severity) error {
	return nil
}

func (PlainWriter) SetHeaderMessage() error {
	return nil
}

func (PlainWriter) SetLineNumber() error {
	return nil
}

func (PlainWriter) SetNoteBullet() error {
	return nil
}

func (PlainWriter) SetSourceBorder() error {
	return nil
}

func (PlainWriter) SetLabel(_ diagnostic.Severity, _ diagnostic.LabelStyle) error {
	return nil
}

func (PlainWriter) Reset() error {
	return nil
}

// pub(crate) struct WriteStyleByRef<'a, W: ?Sized> {
//     w: &'a mut W,
// }

// impl<'a, W> WriteStyleByRef<'a, W>
// where
//     W: ?Sized,
// {
//     pub fn new(writer: &'a mut W) -> Self {
//         Self { w: writer }
//     }
// }

// #[cfg(feature = "std")]
// impl<'a, W> std::io::Write for WriteStyleByRef<'a, W>
// where
//     W: std::io::Write + ?Sized,
// {
//     fn write(&mut self, buf: &[u8]) -> std::io::Result<usize> {
//         self.w.write(buf)
//     }

//     fn flush(&mut self) -> std::io::Result<()> {
//         self.w.flush()
//     }
// }

func (r *Renderer) Write(p []byte) (n int, err error) {
	return r.writer.Write(p)
}

// #[cfg(not(feature = "std"))]
// impl<'a, W> core::fmt::Write for WriteStyleByRef<'a, W>
// where
//     W: core::fmt::Write + ?Sized,
// {
//     fn write_str(&mut self, s: &str) -> core::fmt::Result {
//         self.w.write_str(s)
//     }

//     fn write_char(&mut self, c: char) -> core::fmt::Result {
//         self.w.write_char(c)
//     }

//     fn write_fmt(&mut self, args: core::fmt::Arguments<'_>) -> core::fmt::Result {
//         self.w.write_fmt(args)
//     }
// }

// impl<'a, W> WriteStyle for WriteStyleByRef<'a, W>
// where
//     W: WriteStyle + ?Sized,
// {
//     fn set_header(&mut self, severity: Severity) -> GeneralWriteResult {
//         self.w.set_header(severity)
//     }

//     fn set_header_message(&mut self) -> GeneralWriteResult {
//         self.w.set_header_message()
//     }

//     fn set_line_number(&mut self) -> GeneralWriteResult {
//         self.w.set_line_number()
//     }

//     fn set_note_bullet(&mut self) -> GeneralWriteResult {
//         self.w.set_note_bullet()
//     }

//     fn set_source_border(&mut self) -> GeneralWriteResult {
//         self.w.set_source_border()
//     }

//     fn set_label(&mut self, severity: Severity, label_style: LabelStyle) -> GeneralWriteResult {
//         self.w.set_label(severity, label_style)
//     }

//     fn reset(&mut self) -> GeneralWriteResult {
//         self.w.reset()
//     }
// }

// The 'location focus' of a source code snippet.
type Locus struct {
	// The user-facing name of the file.
	Name string

	// The location.
	Location srcspan.Location
}

// Single-line label, with an optional message.
//
// ```text
// ^^^^^^^^^ blah blah
// ```
type SingleLabel struct {
	LabelStyle diagnostic.LabelStyle
	Range      srcspan.Range
	Message    string
}

// A multi-line label to render.
//
// Locations are relative to the start of where the source code is rendered.
type multiLabelTag int

const (
	MultiLabelTop multiLabelTag = iota
	MultiLabelLeft
	MultiLabelBottom
)

type MultiLabel struct {
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

func NewMultiLabelTop(labelStart int) MultiLabel {
	return MultiLabel{
		tag:        MultiLabelTop,
		labelStart: labelStart,
	}
}

func NewMultiLabelLeft() MultiLabel {
	return MultiLabel{
		tag: MultiLabelLeft,
	}
}

func NewMultiLabelBottom(labelEnds int, diagnostic string) MultiLabel {
	return MultiLabel{
		tag:        MultiLabelBottom,
		labelEnds:  labelEnds,
		diagnostic: diagnostic,
	}
}

func (ml MultiLabel) isTop() bool {
	return ml.tag == MultiLabelTop
}

func (ml MultiLabel) asTop() (labelStart int, ok bool) {
	ok = ml.tag == MultiLabelTop
	if ok {
		labelStart = ml.labelStart
	}
	return
}

func (ml MultiLabel) isLeft() bool {
	return ml.tag == MultiLabelLeft
}

func (ml MultiLabel) isBottom() bool {
	return ml.tag == MultiLabelBottom
}

func (ml MultiLabel) asBottom() (labelEnds int, diagnostic string, ok bool) {
	ok = ml.tag == MultiLabelBottom
	if ok {
		labelEnds = ml.labelEnds
		diagnostic = ml.diagnostic
	}
	return
}

func (ml MultiLabel) String() string {
	switch ml.tag {
	case MultiLabelTop:
		return fmt.Sprintf("Top(start=%d)", ml.labelStart)
	case MultiLabelLeft:
		return "Left()"
	case MultiLabelBottom:
		return fmt.Sprintf("Bottom(labenEnds=%+v, diagnostic=%+v)", ml.labelEnds, ml.diagnostic)
	default:
		panic("")
	}
}

type VerticalBound int

const (
	VerticalBoundTop VerticalBound = iota
	VerticalBoundBottom
)

type Underline struct {
	LabelStyle    diagnostic.LabelStyle
	VerticalBound VerticalBound
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

func (r *Renderer) Chars() Chars {
	return r.config.Chars
}

// Diagnostic header, with severity, code, and message.
//
// ```text
// error[E0001]: unexpected type in `+` application
// ```
func (r *Renderer) RenderHeader(
	locus *Locus,
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
		err = r.SnippetLocus(locus)
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
	locus *Locus,
) (err error) {
	err = r.OuterGutter(outer_padding)
	if err != nil {
		return err
	}

	err = r.SetSourceBorder()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(r, "%s", r.Chars().SnippetStart)
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

	err = r.SnippetLocus(locus)
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
	single_labels []SingleLabel,
	num_multi_labels int,
	multi_labels []struct {
		LabelIndex int
		LabelStyle diagnostic.LabelStyle
		MultiLabel MultiLabel
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
		err = r.OuterGutterNumber(line_number, outer_padding)
		if err != nil {
			return err
		}
		err = r.BorderLeft()
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
					if start, ok := ml.MultiLabel.asTop(); ok {
						if start <= len(source)-len(strings.TrimLeft(source, " \t")) {
							err = r.LabelMultiTopLeft(severity, ml.LabelStyle)
							if err != nil {
								return err
							}
						} else {
							err = r.InnerGutterSpace()
							if err != nil {
								return err
							}
						}
					} else if ml.MultiLabel.isLeft() || ml.MultiLabel.isBottom() {
						err = r.LabelMultiLeft(severity, ml.LabelStyle, nil)
						if err != nil {
							return err
						}
					}
					multi_labels_idx++
				} else {
					err = r.InnerGutterSpace()
					if err != nil {
						return err
					}
				}
			} else {
				err = r.InnerGutterSpace()
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

		for _, cm := range r.CharMetrics(CharIndices(source)) {
			metrics := cm.Metrics
			ch := cm.Char

			column_range := srcspan.NewRange(metrics.byte_index, metrics.byte_index+runewidth.RuneWidth(ch))

			// Check if we are overlapping a primary label
			var is_primary bool
			for _, sl := range single_labels {
				if sl.LabelStyle == diagnostic.Primary && isOverlapping(sl.Range, column_range) {
					is_primary = true
				}
			}
			for _, ml := range multi_labels {
				if ml.LabelStyle == diagnostic.Primary {
					if start, ok := ml.MultiLabel.asTop(); ok {
						is_primary = is_primary || column_range.Start >= start
					} else if ml.MultiLabel.isLeft() {
						is_primary = true
					} else if start, _, ok := ml.MultiLabel.asBottom(); ok {
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
			SingleLabel SingleLabel
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
						SingleLabel SingleLabel
					}{
						Index:       label_index,
						SingleLabel: label,
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
					if isOverlapping(trailing_label.SingleLabel.Range, sl.Range) {
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

		err = r.OuterGutter(outer_padding)
		if err != nil {
			return err
		}
		err = r.BorderLeft()
		if err != nil {
			return err
		}
		err = r.InnerGutter(severity, num_multi_labels, multi_labels)
		if err != nil {
			return err
		}
		//             write!(self, " ")?;
		_, err = fmt.Fprintf(r, " ")
		if err != nil {
			return err
		}

		var previous_label_style *diagnostic.LabelStyle
		placeholder_metrics := Metrics{
			byte_index:    len(source),
			unicode_width: 1,
		}

		for _, cm := range append(r.CharMetrics(CharIndices(source)),
			// Add a placeholder source column at the end to allow for
			// printing carets at the end of lines, eg:
			//
			// ```text
			// 1 │ Hello world!
			//   │             ^
			// ```
			struct {
				Metrics Metrics
				Char    rune
			}{
				Metrics: placeholder_metrics,
				Char:    '\000',
			}) {
			metrics := cm.Metrics
			ch := cm.Char

			// Find the current label style at this column
			column_range := srcspan.NewRange(metrics.byte_index, metrics.byte_index+runewidth.RuneWidth(ch))

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
			if previous_label_style != current_label_style {
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
					ch := r.Chars().SinglePrimaryCaret
					caret_ch = &ch
				case diagnostic.Secondary:
					ch := r.Chars().SingleSecondaryCaret
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

			err = r.SetLabel(severity, trailing_label.SingleLabel.LabelStyle)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(r, "%s", trailing_label.SingleLabel.Message)
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
			err = r.OuterGutter(outer_padding)
			if err != nil {
				return err
			}
			err = r.BorderLeft()
			if err != nil {
				return err
			}
			err = r.InnerGutter(severity, num_multi_labels, multi_labels)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(r, " ")
			if err != nil {
				return err
			}

			err = r.CaretPointers(
				severity,
				max_label_start,
				single_labels,
				trailing_label,
				CharIndices(source),
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
			hanging_labels := HangingLabels(single_labels, trailing_label)
			slices.Reverse(hanging_labels)

			for _, sl := range hanging_labels {
				label_style := sl.LabelStyle
				range_ := sl.Range
				message := sl.Message

				err = r.OuterGutter(outer_padding)
				if err != nil {
					return err
				}

				err = r.BorderLeft()
				if err != nil {
					return err
				}

				err = r.InnerGutter(severity, num_multi_labels, multi_labels)
				if err != nil {
					return err
				}

				_, err = fmt.Fprintf(r, " ")
				if err != nil {
					return err
				}

				//.take_while(|(byte_index, _)| *byte_index < range.start),
				char_indices := CharIndices(source)
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

				err = r.CaretPointers(
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
		_label := ml.MultiLabel

		if _label.isLeft() {
			// no label caret needed
			continue
		}
		if start, ok := _label.asTop(); ok {
			// no label caret needed if this can be started in front of the line
			if start <= len(source)-len(strings.TrimLeft(source, " ")) {
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

		err = r.OuterGutter(outer_padding)
		if err != nil {
			return err
		}
		err = r.BorderLeft()
		if err != nil {
			return err
		}

		// Write inner gutter.
		//
		// ```text
		//  │ ╭─│───│
		// ```
		var underline *Underline
		multi_labels_idx := 0
		for label_column := 0; label_column < num_multi_labels; label_column++ { //nolint:intrange // emulate rust code
			if multi_labels_idx < len(multi_labels) {
				ml := multi_labels[multi_labels_idx]

				i := multi_labels_idx
				label_index := ml.LabelIndex
				ls := ml.LabelStyle
				label := ml.MultiLabel

				if label_index == label_column {
					if label.isLeft() {
						err = r.LabelMultiLeft(severity, ls, &underline.LabelStyle)
						if err != nil {
							return err
						}
					} else if _, ok := label.asTop(); ok && multi_label_index > i {
						err = r.LabelMultiLeft(severity, ls, &underline.LabelStyle)
						if err != nil {
							return err
						}
					} else if _, _, ok := label.asBottom(); ok && multi_label_index < i {
						err = r.LabelMultiLeft(severity, ls, &underline.LabelStyle)
						if err != nil {
							return err
						}
					} else if _, ok := label.asTop(); ok && multi_label_index == i {
						underline = &Underline{
							LabelStyle:    ls,
							VerticalBound: VerticalBoundTop,
						}
						err = r.LabelMultiTopLeft(severity, label_style)
						if err != nil {
							return err
						}
					} else if _, _, ok := label.asBottom(); ok && multi_label_index == i {
						err = r.LabelMultiBottomLeft(severity, label_style)
						if err != nil {
							return err
						}
					} else if label.isTop() || label.isBottom() {
						err = r.InnerGutterColumn(severity, underline)
						if err != nil {
							return err
						}
					}
					multi_labels_idx++
				} else {
					err = r.InnerGutterColumn(severity, underline)
					if err != nil {
						return err
					}
				}
			} else {
				err = r.InnerGutterColumn(severity, underline)
				if err != nil {
					return err
				}
			}
		}

		// Finish the top or bottom caret
		if bottom_message == nil {
			err = r.LabelMultiTopCaret(severity, label_style, source, range_)
			if err != nil {
				return err
			}
		} else {
			err = r.LabelMultiBottomCaret(severity, label_style, source, range_, *bottom_message)
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
		MultiLabel MultiLabel
	},
) (err error) {
	err = r.OuterGutter(outer_padding)
	if err != nil {
		return
	}
	err = r.BorderLeft()
	if err != nil {
		return
	}
	err = r.InnerGutter(severity, num_multi_labels, multi_labels)
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
		MultiLabel MultiLabel
	},
) (err error) {
	err = r.OuterGutter(outer_padding)
	if err != nil {
		return err
	}
	err = r.BorderLeftBreak()
	if err != nil {
		return err
	}
	err = r.InnerGutter(severity, num_multi_labels, multi_labels)
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
	for note_line_index, line := range strings.Split(message, "\n") {
		err = r.OuterGutter(outer_padding)
		if err != nil {
			return err
		}

		switch note_line_index {
		case 0:
			err = r.SetNoteBullet()
			if err != nil {
				return err
			}
			fmt.Fprintf(r, "%c", r.Chars().NoteBullet)

			err = r.Reset()
			if err != nil {
				return err
			}
		default:
			fmt.Fprintf(r, " ")
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
func (r *Renderer) CharMetrics(
	char_indices []struct {
		Index int
		Char  rune
	},
) []struct {
	Metrics Metrics
	Char    rune
} {
	tab_width := r.config.TabWidth
	unicode_column := 0

	var char_metrics []struct {
		Metrics Metrics
		Char    rune
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

		metrics := Metrics{
			byte_index:    ci.Index,
			unicode_width: uw,
		}
		unicode_column += metrics.unicode_width

		char_metrics = append(char_metrics,
			struct {
				Metrics Metrics
				Char    rune
			}{
				Metrics: metrics,
				Char:    ci.Char,
			},
		)
	}

	return char_metrics
}

// Location focus.
func (r *Renderer) SnippetLocus(locus *Locus) (err error) {
	// Location.{Line,Column} is Index or 0-based.
	_, err = fmt.Fprintf(r, "%s:%d:%d", locus.Name, locus.Location.Line+1, locus.Location.Column+1)
	return
}

// The outer gutter of a source line.
func (r *Renderer) OuterGutter(
	outer_padding int,
) (err error) {
	_, err = fmt.Fprintf(r, "%*s ", outer_padding, " ")
	return
}

// The outer gutter of a source line, with line number.
func (r *Renderer) OuterGutterNumber(
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
func (r *Renderer) BorderLeft() (err error) {
	err = r.SetSourceBorder()
	if err != nil {
		return
	}

	_, err = fmt.Fprintf(r, "%c", r.Chars().SourceBorderLeft)
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
func (r *Renderer) BorderLeftBreak() (err error) {
	err = r.SetSourceBorder()
	if err != nil {
		return
	}

	_, err = fmt.Fprintf(r, "%c", r.Chars().SourceBorderLeftBreak)
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
func (r *Renderer) CaretPointers(
	severity diagnostic.Severity,
	max_label_start int,
	single_labels []SingleLabel,
	trailing_label *struct {
		Index       int
		SingleLabel SingleLabel
	},
	char_indices []struct {
		Index int
		Char  rune
	},
) (err error) {
	for _, cm := range r.CharMetrics(char_indices) {
		metrics := cm.Metrics
		ch := cm.Char

		column_range_start := metrics.byte_index
		column_range_end := metrics.byte_index + runewidth.RuneWidth(ch)

		var label_style *diagnostic.LabelStyle
		var max_priority int
		for _, sl := range HangingLabels(single_labels, trailing_label) {
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
			_, err = fmt.Fprintf(r, "%c", r.Chars().PointerLeft)
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
					break
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
func (r *Renderer) LabelMultiLeft(
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
		label_style = *underline
		err = r.SetLabel(severity, label_style)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(r, "%c", r.Chars().MultiTop)
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

	_, err = fmt.Fprintf(r, "%c", r.Chars().MultiLeft)
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
func (r *Renderer) LabelMultiTopLeft(
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

	_, err = fmt.Fprintf(r, "%c", r.Chars().MultiTopLeft)
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
func (r *Renderer) LabelMultiBottomLeft(
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

	_, err = fmt.Fprintf(r, "%c", r.Chars().MultiBottomLeft)
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
func (r *Renderer) LabelMultiTopCaret(
	severity diagnostic.Severity,
	label_style diagnostic.LabelStyle,
	source string,
	start int,
) (err error) {
	err = r.SetLabel(severity, label_style)
	if err != nil {
		return err
	}

	for _, cm := range r.CharMetrics(CharIndices(source)) {
		metrics := cm.Metrics

		if metrics.byte_index >= start {
			break
		}

		// FIXME: improve rendering of carets between character boundaries
		for i := 0; i < metrics.unicode_width; i++ { //nolint:intrange // emulate rust code
			fmt.Fprintf(r, "%c", r.Chars().MultiBottom)
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
func (r *Renderer) LabelMultiBottomCaret(
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

	for _, cm := range r.CharMetrics(CharIndices(source)) {
		metrics := cm.Metrics

		if metrics.byte_index >= start {
			break
		}

		// FIXME: improve rendering of carets between character boundaries
		for i := 0; i < metrics.unicode_width; i++ { //nolint:intrange // emulate rust code
			fmt.Fprintf(r, "%c", r.Chars().MultiBottom)
		}
	}

	var caret_end rune
	switch label_style {
	case diagnostic.Primary:
		caret_end = r.config.Chars.MultiPrimaryCaretStart
	case diagnostic.Secondary:
		caret_end = r.config.Chars.MultiSecondaryCaretStart
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
func (r *Renderer) InnerGutterColumn(
	severity diagnostic.Severity,
	underline *Underline,
) (err error) {
	if underline == nil {
		err = r.InnerGutterSpace()
	} else {
		label_style := underline.LabelStyle
		vertical_bound := underline.VerticalBound

		err = r.SetLabel(severity, label_style)
		if err != nil {
			return err
		}

		var ch rune
		switch vertical_bound {
		case VerticalBoundTop:
			ch = r.config.Chars.MultiTop
		case VerticalBoundBottom:
			ch = r.config.Chars.MultiBottom
		default:
			panic(fmt.Sprintf("Unknown srcspan.VerticalBound: %v", vertical_bound))
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
func (r *Renderer) InnerGutterSpace() (err error) {
	_, err = fmt.Fprintf(r, "  ")
	return
}

// Writes an inner gutter, with the left lines if necessary.
func (r *Renderer) InnerGutter(
	severity diagnostic.Severity,
	num_multi_labels int,
	multi_labels []struct {
		LabelIndex int
		LabelStyle diagnostic.LabelStyle
		MultiLabel MultiLabel
	},
) (err error) {
	multi_labels_idx := 0

	for label_column := 0; label_column < num_multi_labels; label_column++ { //nolint:intrange // emulate rust code
		if multi_labels_idx < len(multi_labels) {
			ml := multi_labels[multi_labels_idx]

			if ml.LabelIndex == label_column {
				if ml.MultiLabel.isLeft() || ml.MultiLabel.isBottom() {
					err = r.LabelMultiLeft(severity, ml.LabelStyle, nil)
					if err != nil {
						return err
					}
					multi_labels_idx++
				} else if ml.MultiLabel.isTop() {
					err = r.InnerGutterSpace()
					if err != nil {
						return err
					}
					multi_labels_idx++
				}
			} else {
				err = r.InnerGutterSpace()
				if err != nil {
					return err
				}
			}
		} else {
			err = r.InnerGutterSpace()
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

type Metrics struct {
	byte_index    int
	unicode_width int
}

// Check if two ranges overlap
func isOverlapping(r0, r1 srcspan.Range) bool {
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
func HangingLabels(
	single_labels []SingleLabel,
	trailing_label *struct {
		Index       int
		SingleLabel SingleLabel
	},
) []SingleLabel {
	labels := []SingleLabel{}

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

func CharIndices(s string) []struct {
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
