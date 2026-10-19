package term

import (
	"math"
	"slices"

	"github.com/tidwall/btree"

	"srcspan/diagnostic"
	"srcspan/files"
)

// The render functions take a parameter named `files` (as in Rust), which
// shadows the `files` package in their bodies, so the package's names used
// there are aliased here.
type fileRange = files.Range

var newRange = files.NewRange

// Calculate the number of decimal digits in `n`.
func countDigits(n int) int {
	if n == 0 {
		return 1
	}
	return int(math.Log10(float64(n)) + 1.0)
}

// Output a richly formatted diagnostic, with source code previews.
type RichDiagnostic struct {
	diagnostic *diagnostic.Diagnostic
	config     *Config
}

func NewRichDiagnostic(diagnostic *diagnostic.Diagnostic, config *Config) *RichDiagnostic {
	return &RichDiagnostic{
		diagnostic: diagnostic,
		config:     config,
	}
}

func (d *RichDiagnostic) Render(
	files files.Files,
	renderer Renderer,
) error {
	// TODO: Make this data structure external, to allow for allocation reuse
	labeled_files := []*labeledFile{}

	// Keep track of the outer padding to use when rendering the
	// snippets of source code.
	outer_padding := 0

	// Group labels by file
	for _, label := range d.diagnostic.Labels {
		//nolint:shadow // to avoid var decls of the the first vars.
		start_line_index, err := files.LineIndex(label.FileId, label.Range.Start)
		if err != nil {
			return err
		}
		start_line_number, err := files.LineNumber(label.FileId, start_line_index)
		if err != nil {
			return err
		}
		start_line_range, err := files.LineRange(label.FileId, start_line_index)
		if err != nil {
			return err
		}
		end_line_index, err := files.LineIndex(label.FileId, label.Range.End)
		if err != nil {
			return err
		}
		end_line_number, err := files.LineNumber(label.FileId, end_line_index)
		if err != nil {
			return err
		}
		end_line_range, err := files.LineRange(label.FileId, end_line_index)
		if err != nil {
			return err
		}

		outer_padding = max(outer_padding,
			countDigits(start_line_number),
			countDigits(end_line_number))

		// NOTE: This could be made more efficient by using an associative
		// data structure like a hashmap or B-tree,  but we use a vector to
		// preserve the order that unique files appear in the list of labels.
		labeled_file_idx := slices.IndexFunc(labeled_files, func(lf *labeledFile) bool {
			return label.FileId == lf.FileId
		})

		var labeled_file *labeledFile

		if labeled_file_idx != -1 {
			// another diagnostic also referenced this file
			labeled_file = labeled_files[labeled_file_idx]
			if labeled_file.MaxLabelStyle > label.Style || (labeled_file.MaxLabelStyle == label.Style && labeled_file.Start > label.Range.Start) {
				// this label has a higher style or has the same style but starts earlier
				labeled_file.Start = label.Range.Start
				l, err := files.Location(label.FileId, label.Range.Start)
				if err != nil {
					return err
				}

				labeled_file.Location = l
				labeled_file.MaxLabelStyle = label.Style
			}
		} else {
			name, err := files.Name(label.FileId)
			if err != nil {
				return err
			}
			l, err := files.Location(label.FileId, label.Range.Start)
			if err != nil {
				return err
			}

			// no other diagnostic referenced this file yet
			lines := btree.Map[int, *labeledLine]{}
			labeled_file = &labeledFile{
				FileId:         label.FileId,
				Start:          label.Range.Start,
				Name:           name,
				Location:       l,
				NumMultiLabels: 0,
				Lines:          &lines,
				MaxLabelStyle:  label.Style,
			}
			labeled_files = append(labeled_files, labeled_file)
		}

		// insert context lines before label
		// start from 1 because 0 would be the start of the label itself
		for offset := 1; offset <= d.config.BeforeLabelLines; offset++ {
			index := 0

			sub := start_line_index - offset
			if sub >= 0 {
				index = sub
			} else {
				// we are going from smallest to largest offset, so if
				// the offset can not be subtracted from the start we
				// reached the first line
				break
			}

			range_, err := files.LineRange(label.FileId, index)
			if err == nil {
				line := labeled_file.getOrInsertLine(index, range_, start_line_number-offset)
				line.MustRender = true
			} else {
				break
			}
		}

		// insert context lines after label
		// start from 1 because 0 would be the end of the label itself
		for offset := 1; offset <= d.config.AfterLabelLines; offset++ {
			index := end_line_index + offset
			if index < 0 {
				panic("line index too big")
			}

			span, err := files.LineRange(label.FileId, index)
			if err == nil {
				line := labeled_file.getOrInsertLine(index, span, end_line_number+offset)
				line.MustRender = true
			} else {
				break
			}
		}

		if start_line_index == end_line_index {
			// Single line
			//
			// ```text
			// 2 │ (+ test "")
			//   │         ^^ expected `Int` but found `String`
			// ```
			label_start := label.Range.Start - start_line_range.Start

			// Ensure that we print at least one caret, even when we
			// have a zero-length source range.
			label_end := max(label.Range.End-start_line_range.Start, label_start+1)

			line := labeled_file.getOrInsertLine(
				start_line_index,
				start_line_range,
				start_line_number,
			)

			// Ensure that the single line labels are lexicographically
			// sorted by the range of source code that they cover.
			index, _ := slices.BinarySearchFunc(
				line.SingleLabels,
				singleLabel{Range: newRange(label_start, label_end)},
				func(sl1 singleLabel, sl2 singleLabel) int {
					diff := sl1.Range.Start - sl2.Range.Start
					if diff != 0 {
						return diff
					}
					return sl1.Range.End - sl2.Range.End
				},
				// If the ranges are the same, order the labels in reverse
				// to how they were originally specified in the diagnostic.
				// This helps with printing in the renderer.
			)

			line.SingleLabels = slices.Insert(line.SingleLabels, index, singleLabel{
				LabelStyle: label.Style,
				Range:      newRange(label_start, label_end),
				Message:    label.Message,
			})

			// If this line is not rendered, the SingleLabel is not visible.
			line.MustRender = true
		} else {
			// Multiple lines
			//
			// ```text
			// 4 │   fizz₁ num = case (mod num 5) (mod num 3) of
			//   │ ╭─────────────^
			// 5 │ │     0 0 => "FizzBuzz"
			// 6 │ │     0 _ => "Fizz"
			// 7 │ │     _ 0 => "Buzz"
			// 8 │ │     _ _ => num
			//   │ ╰──────────────^ `case` clauses have incompatible types
			// ```
			label_index := labeled_file.NumMultiLabels
			labeled_file.NumMultiLabels += 1

			// First labeled line
			label_start := label.Range.Start - start_line_range.Start

			start_line := labeled_file.getOrInsertLine(
				start_line_index,
				start_line_range,
				start_line_number,
			)

			start_line.MultiLabels = append(
				start_line.MultiLabels,
				struct {
					LabelIndex int
					LabelStyle diagnostic.LabelStyle
					multiLabel multiLabel
				}{
					LabelIndex: label_index,
					LabelStyle: label.Style,
					multiLabel: newMultiLabelTop(label_start),
				},
			)

			// The first line has to be rendered so the start of the label is visible.
			start_line.MustRender = true

			// Marked lines
			//
			// ```text
			// 5 │ │     0 0 => "FizzBuzz"
			// 6 │ │     0 _ => "Fizz"
			// 7 │ │     _ 0 => "Buzz"
			// ```
			for line_index := start_line_index + 1; line_index < end_line_index; line_index++ {
				line_range, err := files.LineRange(label.FileId, line_index)
				if err != nil {
					return err
				}
				line_number, err := files.LineNumber(label.FileId, line_index)
				if err != nil {
					return err
				}

				outer_padding = max(outer_padding, countDigits(line_number))

				line := labeled_file.getOrInsertLine(line_index, line_range, line_number)
				line.MultiLabels = append(
					line.MultiLabels,
					struct {
						LabelIndex int
						LabelStyle diagnostic.LabelStyle
						multiLabel multiLabel
					}{
						LabelIndex: label_index,
						LabelStyle: label.Style,
						multiLabel: newMultiLabelLeft(),
					},
				)

				// The line should be rendered to match the configuration of how much context to show.
				line.MustRender = line.MustRender || (
				// Is this line part of the context after the start of the label?
				line_index-start_line_index <= d.config.StartContextLines ||
					// Is this line part of the context before the end of the label?
					end_line_index-line_index <= d.config.EndContextLines)
			}

			// Last labeled line
			//
			// ```text
			// 8 │ │     _ _ => num
			//   │ ╰──────────────^ `case` clauses have incompatible types
			// ```
			label_end := label.Range.End - end_line_range.Start

			end_line := labeled_file.getOrInsertLine(
				end_line_index,
				end_line_range,
				end_line_number,
			)

			end_line.MultiLabels = append(
				end_line.MultiLabels,
				struct {
					LabelIndex int
					LabelStyle diagnostic.LabelStyle
					multiLabel multiLabel
				}{
					LabelIndex: label_index,
					LabelStyle: label.Style,
					multiLabel: newMultiLabelBottom(label_end, label.Message),
				},
			)

			// The last line has to be rendered so the end of the label is visible.
			end_line.MustRender = true
		}
	}

	// Header and message
	//
	// ```text
	// error[E0001]: unexpected type in `+` application
	// ```
	code := d.diagnostic.Code
	err := renderer.RenderHeader(
		nil,
		d.diagnostic.Severity,
		&code,
		d.diagnostic.Message,
	)
	if err != nil {
		return err
	}

	// Source snippets
	//
	// ```text
	//   ┌─ test:2:9
	//   │
	// 2 │ (+ test "")
	//   │         ^^ expected `Int` but found `String`
	//   │
	// ```
	labeled_files_idx := 0
	for labeled_files_idx < len(labeled_files) {
		labeled_file := labeled_files[labeled_files_idx]

		var source string
		source, err = files.Source(labeled_file.FileId)
		if err != nil {
			return err
		}

		// Top left border and locus.
		//
		// ```text
		// ┌─ test:2:9
		// ```
		if labeled_file.Lines.Len() != 0 {
			err = renderer.RenderSnippetStart(
				outer_padding,
				&locus{
					Name:     labeled_file.Name,
					Location: labeled_file.Location,
				},
			)
			if err != nil {
				return err
			}
			err = renderer.RenderSnippetEmpty(
				outer_padding,
				d.diagnostic.Severity,
				labeled_file.NumMultiLabels,
				[]struct {
					LabelIndex int
					LabelStyle diagnostic.LabelStyle
					multiLabel multiLabel
				}{},
			)
			if err != nil {
				return err
			}
		}

		lines := []struct {
			LineIndex   int
			labeledLine *labeledLine
		}{}
		labeled_file.Lines.Scan(func(line_index int, line *labeledLine) bool {
			if line.MustRender {
				line_info := struct {
					LineIndex   int
					labeledLine *labeledLine
				}{
					LineIndex:   line_index,
					labeledLine: line,
				}

				lines = append(lines, line_info)
			}
			return true
		})

		line_index := 0
		for line_index < len(lines) {
			line_info := lines[line_index]
			line := *line_info.labeledLine

			err = renderer.RenderSnippetSource(
				outer_padding,
				line.Number,
				source[line.Range.Start:line.Range.End],
				d.diagnostic.Severity,
				line.SingleLabels,
				labeled_file.NumMultiLabels,
				line.MultiLabels,
			)
			if err != nil {
				return err
			}

			// Check to see if we need to render any intermediate stuff
			// before rendering the next line.
			if line_index+1 < len(lines) {
				next_line := lines[line_index+1]

				switch lines[line_index+1].LineIndex - lines[line_index].LineIndex {
				case 1:
					// Consecutive lines
				case 2:
					// One line between the current line and the next line

					// Write a source line
					file_id := labeled_file.FileId

					// This line was not intended to be rendered initially.
					// To render the line right, we have to get back the original labels.
					labels := []struct {
						LabelIndex int
						LabelStyle diagnostic.LabelStyle
						multiLabel multiLabel
					}{}

					if line, ok := labeled_file.Lines.Get(line_info.LineIndex + 1); ok {
						labels = append(labels, line.MultiLabels...)
					}

					var ln int
					ln, err = files.LineNumber(file_id, line_info.LineIndex+1)
					if err != nil {
						return err
					}

					var lr fileRange
					lr, err = files.LineRange(file_id, line_info.LineIndex+1)
					if err != nil {
						return err
					}
					err = renderer.RenderSnippetSource(
						outer_padding,
						ln,
						source[lr.Start:lr.End],
						d.diagnostic.Severity,
						[]singleLabel{},
						labeled_file.NumMultiLabels,
						labels,
					)
					if err != nil {
						return err
					}

				default:
					// Source break
					//
					// ```text
					// ·
					// ```
					err = renderer.RenderSnippetBreak(
						outer_padding,
						d.diagnostic.Severity,
						labeled_file.NumMultiLabels,
						next_line.labeledLine.MultiLabels,
					)
					if err != nil {
						return err
					}
				}
			}

			line_index++
		}

		// Check to see if we should render a trailing border after the
		// final line of the snippet.
		if labeled_files_idx+1 >= len(labeled_files) && len(d.diagnostic.Notes) == 0 {
			// We don't render a border if we are at the final newline
			// without trailing notes, because it would end up looking too
			// spaced-out in combination with the final new line.
		} else {
			// Render the trailing snippet border.
			err = renderer.RenderSnippetEmpty(
				outer_padding,
				d.diagnostic.Severity,
				labeled_file.NumMultiLabels,
				[]struct {
					LabelIndex int
					LabelStyle diagnostic.LabelStyle
					multiLabel multiLabel
				}{},
			)
			if err != nil {
				return err
			}
		}

		labeled_files_idx++
	}

	// Additional notes
	//
	// ```text
	// = expected type `Int`
	//      found type `String`
	// ```
	for _, note := range d.diagnostic.Notes {
		err = renderer.RenderSnippetNote(outer_padding, note)
		if err != nil {
			return err
		}
	}
	err = renderer.RenderEmpty()
	if err != nil {
		return err
	}

	return err
}

type labeledFile struct {
	FileId         int
	Start          int
	Name           string
	Location       files.Location
	NumMultiLabels int
	Lines          *btree.Map[int, *labeledLine]
	MaxLabelStyle  diagnostic.LabelStyle
}

func (lf *labeledFile) getOrInsertLine(
	line_index int,
	line_range files.Range,
	line_number int,
) *labeledLine {
	//revive:disable:indent-error-flow
	if line, ok := lf.Lines.Get(line_index); !ok {
		line = &labeledLine{
			Range:        line_range,
			Number:       line_number,
			SingleLabels: []singleLabel{},
			MultiLabels: []struct {
				LabelIndex int
				LabelStyle diagnostic.LabelStyle
				multiLabel multiLabel
			}{},
			// This has to be false by default so we know if it must be rendered by another condition already.
			MustRender: false,
		}
		lf.Lines.Set(line_index, line)
		return line
	} else {
		return line
	}
}

type labeledLine struct {
	Number       int
	Range        files.Range
	SingleLabels []singleLabel
	MultiLabels  []struct {
		LabelIndex int
		LabelStyle diagnostic.LabelStyle
		multiLabel multiLabel
	}
	MustRender bool
}

// Output a short diagnostic, with a line number, severity, and message.
type ShortDiagnostic struct {
	diagnostic *diagnostic.Diagnostic
	showNotes  bool
}

func NewShortDiagnostic(diagnostic *diagnostic.Diagnostic, showNotes bool) *ShortDiagnostic {
	return &ShortDiagnostic{
		diagnostic: diagnostic,
		showNotes:  showNotes,
	}
}

func (d *ShortDiagnostic) Render(
	files files.Files,
	renderer Renderer,
) error {
	// Located headers
	//
	// ```text
	// test:2:9: error[E0001]: unexpected type in `+` application
	// ```
	primary_labels_encountered := 0

	// Only primary labels get a header.
	for _, label := range d.diagnostic.Labels {
		if label.Style != diagnostic.Primary {
			continue
		}
		primary_labels_encountered += 1

		name, err := files.Name(label.FileId)
		if err != nil {
			return err
		}
		loc, err := files.Location(label.FileId, label.Range.Start)
		if err != nil {
			return err
		}

		err = renderer.RenderHeader(
			&locus{
				Name:     name,
				Location: loc,
			},
			d.diagnostic.Severity,
			&d.diagnostic.Code,
			d.diagnostic.Message,
		)
		if err != nil {
			return err
		}
	}

	// Fallback to printing a non-located header if no primary labels were encountered
	//
	// ```text
	// error[E0002]: Bad config found
	// ```
	if primary_labels_encountered == 0 {
		err := renderer.RenderHeader(
			nil,
			d.diagnostic.Severity,
			&d.diagnostic.Code,
			d.diagnostic.Message,
		)
		if err != nil {
			return err
		}
	}

	if d.showNotes {
		// Additional notes
		//
		// ```text
		// = expected type `Int`
		//      found type `String`
		// ```
		for _, note := range d.diagnostic.Notes {
			err := renderer.RenderSnippetNote(0, note)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
