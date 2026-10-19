package main

// Renders the preview SVG for the README.
//
// To update the preview, execute the following command from the top level of
// the repository:
//
// ```sh
// go run ./examples/codespan/readme_preview svg > readme_preview.svg
// ```

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"srcspan/diagnostic"
	"srcspan/files"
	"srcspan/term"

	"srcspan/internal/unindent"
)

// (`enum Opts` in Rust)
type Opts interface{ isOpts() }

// Render SVG output
type OptsSvg struct{}

// Render Stderr output
type OptsStderr struct {
	// Configure coloring of output
	Color term.ColorOutput
}

func (OptsSvg) isOpts()    {}
func (OptsStderr) isOpts() {}

func parseArgs() (Opts, error) {
	if len(os.Args) < 2 {
		return nil, fmt.Errorf("missing argument")
	}

	switch value := os.Args[1]; value {
	case "svg":
		return OptsSvg{}, nil
	case "stderr":
		flags := flag.NewFlagSet("stderr", flag.ContinueOnError)
		color := flags.String("color", "auto", "none|auto|color")
		if err := flags.Parse(os.Args[2:]); err != nil {
			return nil, err
		}
		switch *color {
		case "none":
			return OptsStderr{Color: term.ColorOutputNone}, nil
		case "auto":
			return OptsStderr{Color: term.ColorOutputAuto}, nil
		case "color":
			return OptsStderr{Color: term.ColorOutputForce}, nil
		default:
			return nil, fmt.Errorf("--color value should be one of 'none', 'auto', 'color'")
		}
	default:
		return nil, fmt.Errorf("%q: not a valid subcommand", value)
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	file := files.NewSimpleFile(
		"FizzBuzz.fun",
		unindent.Unindent(`
                module FizzBuzz where

                fizz₁ : Nat → String
                fizz₁ num = case (mod num 5) (mod num 3) of
                    0 0 => "FizzBuzz"
                    0 _ => "Fizz"
                    _ 0 => "Buzz"
                    _ _ => num

                fizz₂ : Nat → String
                fizz₂ num =
                    case (mod num 5) (mod num 3) of
                        0 0 => "FizzBuzz"
                        0 _ => "Fizz"
                        _ 0 => "Buzz"
                        _ _ => num
            `),
	)

	diagnostics := []*diagnostic.Diagnostic{diagnostic.NewError().
		WithMessage("`case` clauses have incompatible types").
		WithCode("E0308").
		WithLabels([]diagnostic.Label{
			diagnostic.NewPrimary(0, files.NewRange(328, 331)).WithMessage("expected `String`, found `Nat`"),
			diagnostic.NewSecondary(0, files.NewRange(211, 331)).WithMessage("`case` clauses have incompatible types"),
			diagnostic.NewSecondary(0, files.NewRange(258, 268)).WithMessage("this is found to be of type `String`"),
			diagnostic.NewSecondary(0, files.NewRange(284, 290)).WithMessage("this is found to be of type `String`"),
			diagnostic.NewSecondary(0, files.NewRange(306, 312)).WithMessage("this is found to be of type `String`"),
			diagnostic.NewSecondary(0, files.NewRange(186, 192)).WithMessage("expected type `String` found here"),
		}).
		WithNotes([]string{unindent.Unindent(`
                expected type ` + "`String`" + `
                   found type ` + "`Nat`" + `
            `)})}

	opts, err := parseArgs()
	if err != nil {
		return err
	}

	switch opts := opts.(type) {
	case OptsSvg:
		writer := NewSvgWriter()
		config := term.DefaultConfig()

		for _, diagnostic := range diagnostics {
			if err := term.EmitToWriteStyle(writer, &config, &file, diagnostic); err != nil {
				return err
			}
		}

		num_lines := writer.LineCount()

		padding := 10
		font_size := 12
		line_spacing := 3
		width := 882
		height := padding + num_lines*(font_size+line_spacing) + padding

		content := writer.String()

		_, err := fmt.Fprintf(os.Stdout, svg_template, width, height, padding, font_size, content)
		return err
	case OptsStderr:
		styles := term.DefaultColor()
		styles.ColorOutput = opts.Color
		writer := term.NewStylesWriter(os.Stderr, styles)
		config := term.DefaultConfig()
		for _, diagnostic := range diagnostics {
			if err := term.EmitToWriteStyle(writer, &config, &file, diagnostic); err != nil {
				return err
			}
		}
	}

	return nil
}

// The SVG document. The arguments are width, height, padding, font size
// and content, in this order.
const svg_template = `<svg viewBox="0 0 %[1]d %[2]d" xmlns="http://www.w3.org/2000/svg">
  <style>
    /* https://github.com/aaron-williamson/base16-alacritty/blob/master/colors/base16-tomorrow-night-256.yml */
        pre {
            background: #1d1f21;
            margin: 0;
            padding: %[3]dpx;
            border-radius: 6px;
            color: #ffffff;
            font: %[4]dpx SFMono-Regular, Consolas, Liberation Mono, Menlo, monospace;
        }
        
        pre .bold {
            font-weight: bold;
        }
        
        pre .header-bug,
        pre .header-error {
            color: #cc6666;
            font-weight: bold;
        }
        
        pre .header-warning {
            color: #f0c674;
            font-weight: bold;
        }
        
        pre .header-note {
            color: #b5bd68;
            font-weight: bold;
        }
        
        pre .header-help {
            color: #8abeb7;
            font-weight: bold;
        }
        
        pre .header-message {
            color: #c5c8c6;
            font-weight: bold;
        }
        
        pre .line-number,
        pre .source-border,
        pre .note-bullet {
            color: #81a2be;
        }
        
        pre .label-primary-bug,
        pre .label-primary-error {
            color: #cc6666;
        }
        
        pre .label-primary-warning {
            color: #f0c674;
        }
        
        pre .label-primary-note {
            color: #b5bd68;
        }
        
        pre .label-primary-help {
            color: #8abeb7;
        }
        
        pre .label-secondary-bug,
        pre .label-secondary-error,
        pre .label-secondary-warning,
        pre .label-secondary-note,
        pre .label-secondary-help {
            color: #81a2be;
        }
  </style>

  <foreignObject x="0" y="0" width="%[1]d" height="%[2]d">
    <div xmlns="http://www.w3.org/1999/xhtml">
      <pre>
        %[5]s
      </pre>
    </div>
  </foreignObject>
</svg>`

// A `term.WriteStyle` that renders the styles as HTML spans.
type SvgWriter struct {
	buffer    bytes.Buffer
	span_open bool
}

func NewSvgWriter() *SvgWriter {
	return &SvgWriter{}
}

func (w *SvgWriter) String() string {
	return w.buffer.String()
}

func (w *SvgWriter) LineCount() int {
	return bytes.Count(w.buffer.Bytes(), []byte("\n")) + 1
}

// Close any open span
func (w *SvgWriter) closeSpan() error {
	if w.span_open {
		if _, err := io.WriteString(&w.buffer, "</span>"); err != nil {
			return err
		}
		w.span_open = false
	}
	return nil
}

// Open a new span with the given CSS class
func (w *SvgWriter) openSpan(class string) error {
	// close existing first
	if err := w.closeSpan(); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(&w.buffer, "<span class=\"%s\">", class); err != nil {
		return err
	}
	w.span_open = true
	return nil
}

func (w *SvgWriter) Write(buf []byte) (int, error) {
	last := 0
	for i, b := range buf {
		var escape string
		switch b {
		case '<':
			escape = "&lt;"
		case '>':
			escape = "&gt;"
		case '&':
			escape = "&amp;"
		default:
			continue
		}
		w.buffer.Write(buf[last:i])
		w.buffer.WriteString(escape)
		last = i + 1
	}
	w.buffer.Write(buf[last:])
	return len(buf), nil
}

func (w *SvgWriter) SetHeader(severity diagnostic.Severity) error {
	var class string
	switch severity {
	case diagnostic.Bug:
		class = "header-bug"
	case diagnostic.Error:
		class = "header-error"
	case diagnostic.Warning:
		class = "header-warning"
	case diagnostic.Note:
		class = "header-note"
	case diagnostic.Help:
		class = "header-help"
	}
	return w.openSpan(class)
}

func (w *SvgWriter) SetHeaderMessage() error {
	return w.openSpan("header-message")
}

func (w *SvgWriter) SetLineNumber() error {
	return w.openSpan("line-number")
}

func (w *SvgWriter) SetNoteBullet() error {
	return w.openSpan("note-bullet")
}

func (w *SvgWriter) SetSourceBorder() error {
	return w.openSpan("source-border")
}

func (w *SvgWriter) SetLabel(severity diagnostic.Severity, label_style diagnostic.LabelStyle) error {
	var sev string
	switch severity {
	case diagnostic.Bug:
		sev = "bug"
	case diagnostic.Error:
		sev = "error"
	case diagnostic.Warning:
		sev = "warning"
	case diagnostic.Note:
		sev = "note"
	case diagnostic.Help:
		sev = "help"
	}
	var typ string
	switch label_style {
	case diagnostic.Primary:
		typ = "primary"
	case diagnostic.Secondary:
		typ = "secondary"
	}
	return w.openSpan(fmt.Sprintf("label-%s-%s", typ, sev))
}

func (w *SvgWriter) Reset() error {
	return w.closeSpan()
}

var _ term.WriteStyle = (*SvgWriter)(nil)
