package term

import (
	"fmt"
	"io"

	"github.com/fatih/color"

	"srcspan/srcspan"
)

// Configures how a diagnostic is rendered.

type Config struct {
	/// The display style to use when rendering diagnostics.
	/// Defaults to: [`DisplayStyle::Rich`].
	DisplayStyle DisplayStyle

	/// Column width of tabs.
	/// Defaults to: `4`.
	TabWidth int

	/// Characters to use when rendering the diagnostic.
	Chars Chars

	/// The minimum number of lines to be shown after the line on
	/// which a multiline [`Label`] begins.
	///
	/// Defaults to: `3`.
	///
	/// [`Label`]: crate::diagnostic::Label
	StartContextLines int

	/// The minimum number of lines to be shown before the line on
	/// which a multiline [`Label`] ends.
	///
	/// Defaults to: `1`.
	///
	/// [`Label`]: crate::diagnostic::Label
	EndContextLines int

	/// The minimum number of lines before a label that should be
	/// included for context.
	///
	/// Defaults to: `0`.
	BeforeLabelLines int

	/// The minimum number of lines after a label that should be
	/// included for context.
	///
	/// Defaults to: `0`.
	AfterLabelLines int
}

func DefaultConfig() Config {
	return Config{
		DisplayStyle:      Rich,
		TabWidth:          4,
		Chars:             BoxDrawing(),
		StartContextLines: 3,
		EndContextLines:   1,
		BeforeLabelLines:  0,
		AfterLabelLines:   0,
	}
}

// The display style to use when rendering diagnostics.
type DisplayStyle int

const (
	/// Output a richly formatted diagnostic, with source code previews.
	///
	/// ```text
	/// error[E0001]: unexpected type in `+` application
	///   ┌─ test:2:9
	///   │
	/// 2 │ (+ test "")
	///   │         ^^ expected `Int` but found `String`
	///   │
	///   = expected type `Int`
	///        found type `String`
	///
	/// error[E0002]: Bad config found
	///
	/// ```
	Rich DisplayStyle = iota

	/// Output a condensed diagnostic, with a line number,
	/// severity, message and notes (if any).
	///
	/// ```text
	/// test:2:9: error[E0001]: unexpected type in `+` application
	/// = expected type `Int`
	///      found type `String`
	///
	/// error[E0002]: Bad config found
	/// ```
	Medium

	/// Output a short diagnostic, with a line number, severity,
	/// and message.
	///
	/// ```text
	/// test:2:9: error[E0001]: unexpected type in `+` application
	/// error[E0002]: Bad config found
	/// ```
	Short
)

type Styles struct {
	/// The style to use when rendering bug headers.
	/// Defaults to `fg:red bold intense`.
	HeaderBug *color.Color

	/// The style to use when rendering error headers.
	/// Defaults to `fg:red bold intense`.
	HeaderError *color.Color

	/// The style to use when rendering warning headers.
	/// Defaults to `fg:yellow bold intense`.
	HeaderWarning *color.Color

	/// The style to use when rendering note headers.
	/// Defaults to `fg:green bold intense`.
	HeaderNote *color.Color

	/// The style to use when rendering help headers.
	/// Defaults to `fg:cyan bold intense`.
	HeaderHelp *color.Color

	/// The style to use when the main diagnostic message.
	/// Defaults to `bold intense`.
	HeaderMessage *color.Color

	/// The style to use when rendering bug labels.
	/// Defaults to `fg:red`.
	PrimaryLabelBug *color.Color

	/// The style to use when rendering error labels.
	/// Defaults to `fg:red`.
	PrimaryLabelError *color.Color

	/// The style to use when rendering warning labels.
	/// Defaults to `fg:yellow`.
	PrimaryLabelWarning *color.Color

	/// The style to use when rendering note labels.
	/// Defaults to `fg:green`.
	PrimaryLabelNote *color.Color

	/// The style to use when rendering help labels.
	/// Defaults to `fg:cyan`.
	PrimaryLabelHelp *color.Color

	/// The style to use when rendering secondary labels.
	/// Defaults `fg:blue` (or `fg:cyan` on windows).
	SecondaryLabel *color.Color

	/// The style to use when rendering the line numbers.
	/// Defaults `fg:blue` (or `fg:cyan` on windows).
	LineNumber *color.Color

	/// The style to use when rendering the source code borders.
	/// Defaults `fg:blue` (or `fg:cyan` on windows).
	SourceBorder *color.Color

	/// The style to use when rendering the note bullets.
	/// Defaults `fg:blue` (or `fg:cyan` on windows).
	NoteBullet *color.Color
}

// The style used to mark a header at a given severity.
func (s *Styles) Header(severity srcspan.Severity) *color.Color {
	switch severity {
	case srcspan.Bug:
		return s.HeaderBug
	case srcspan.Error:
		return s.HeaderError
	case srcspan.Warning:
		return s.HeaderWarning
	case srcspan.Note:
		return s.HeaderNote
	case srcspan.Help:
		return s.HeaderHelp
	default:
		panic(fmt.Sprintf("Unknown srcspan.Severity: %v", severity))
	}
}

func (s *Styles) HeaderMessage_() *color.Color {
	return s.HeaderMessage
}

func (s *Styles) LineNumber_() *color.Color {
	return s.LineNumber
}

func (s *Styles) NoteBullet_() *color.Color {
	return s.NoteBullet
}

func (s *Styles) SourceBorder_() *color.Color {
	return s.SourceBorder
}

// The style used to mark a primary or secondary label at a given severity.
func (s *Styles) Label(severity srcspan.Severity, label_style srcspan.LabelStyle) *color.Color {
	switch label_style {
	case srcspan.Primary:
		switch severity {
		case srcspan.Bug:
			return s.PrimaryLabelBug
		case srcspan.Error:
			return s.PrimaryLabelError
		case srcspan.Warning:
			return s.PrimaryLabelWarning
		case srcspan.Note:
			return s.PrimaryLabelNote
		case srcspan.Help:
			return s.PrimaryLabelHelp
		default:
			panic(fmt.Sprintf("Unknown srcspan.Severity: %v", severity))
		}
	case srcspan.Secondary:
		return s.SecondaryLabel
	default:
		panic(fmt.Sprintf("Unknown srcspan.LabelStyle: %v", label_style))

	}
}

func NoColor() *Styles {
	c := color.New()

	return &Styles{
		HeaderBug:     c,
		HeaderError:   c,
		HeaderWarning: c,
		HeaderNote:    c,
		HeaderHelp:    c,
		HeaderMessage: c,

		PrimaryLabelBug:     c,
		PrimaryLabelError:   c,
		PrimaryLabelWarning: c,
		PrimaryLabelNote:    c,
		PrimaryLabelHelp:    c,
		SecondaryLabel:      c,

		LineNumber:   c,
		SourceBorder: c,
		NoteBullet:   c,
	}
}

func DefaultColor() *Styles {
	// Default style

	return &Styles{
		HeaderBug:     color.New(color.FgHiRed),
		HeaderError:   color.New(color.FgHiRed),
		HeaderWarning: color.New(color.FgHiYellow),
		HeaderNote:    color.New(color.FgHiGreen),
		HeaderHelp:    color.New(color.FgHiCyan),
		HeaderMessage: color.New(color.Bold),

		PrimaryLabelBug:     color.New(color.FgHiRed),
		PrimaryLabelError:   color.New(color.FgHiRed),
		PrimaryLabelWarning: color.New(color.FgHiYellow),
		PrimaryLabelNote:    color.New(color.FgHiGreen),
		PrimaryLabelHelp:    color.New(color.FgHiCyan),
		SecondaryLabel:      color.New(color.FgHiCyan),

		LineNumber:   color.New(color.FgHiCyan),
		SourceBorder: color.New(color.FgHiCyan),
		NoteBullet:   color.New(color.FgHiCyan),
	}
}

// A [`WriteStyle`](crate::term::renderer::WriteStyle) implementation that applies custom [`Styles`]
// using termcolor.
//
// Use this to render diagnostics with custom colors
type StylesWriter struct {
	writer io.Writer // WriteStyle
	style  *Styles
}

// Creates a new `StylesWriter` with the given writer and styles.
func NewStylesWriter(writer io.Writer, style *Styles) StylesWriter {
	return StylesWriter{
		writer: writer,
		style:  style,
	}
}

func (sw StylesWriter) Write(p []byte) (int, error) {
	return sw.writer.Write(p)
}

func (sw StylesWriter) Flush() error {
	if f, ok := sw.writer.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

func (sw StylesWriter) SetHeader(severity srcspan.Severity) error {
	sw.style.Header(severity).Set()
	return nil
}

func (sw StylesWriter) SetHeaderMessage() error {
	color.Unset() // nobuyuki
	sw.style.HeaderMessage.Set()
	return nil
}

func (sw StylesWriter) SetLineNumber() error {
	sw.style.LineNumber.Set()
	return nil
}

func (sw StylesWriter) SetNoteBullet() error {
	sw.style.NoteBullet.Set()
	return nil
}

func (sw StylesWriter) SetSourceBorder() error {
	sw.style.SourceBorder.Set()
	return nil
}

func (sw StylesWriter) SetLabel(severity srcspan.Severity, label_style srcspan.LabelStyle) error {
	sw.style.Label(severity, label_style).Set()
	return nil
}

func (sw StylesWriter) Reset() error {
	color.Unset()
	return nil
}

/*
#[cfg(feature = "termcolor")]
impl<T> super::renderer::WriteStyle for T
where
    T: termcolor::WriteColor + ?Sized,
{
    fn set_header(
        &mut self,
        severity: crate::diagnostic::Severity,
    ) -> super::renderer::GeneralWriteResult {
        self.set_color(styles::Styles::default().header(severity))
    }

    fn set_header_message(&mut self) -> super::renderer::GeneralWriteResult {
        self.set_color(&styles::Styles::default().header_message)
    }

    fn set_line_number(&mut self) -> super::renderer::GeneralWriteResult {
        self.set_color(&styles::Styles::default().line_number)
    }

    fn set_note_bullet(&mut self) -> super::renderer::GeneralWriteResult {
        self.set_color(&styles::Styles::default().note_bullet)
    }

    fn set_source_border(&mut self) -> super::renderer::GeneralWriteResult {
        self.set_color(&styles::Styles::default().source_border)
    }

    fn set_label(
        &mut self,
        severity: crate::diagnostic::Severity,
        label_style: crate::diagnostic::LabelStyle,
    ) -> super::renderer::GeneralWriteResult {
        let styles = styles::Styles::default();
        let spec = styles.label(severity, label_style);
        self.set_color(spec)
    }

    fn reset(&mut self) -> super::renderer::GeneralWriteResult {
        self.reset()
    }
}
*/

// Characters to use when rendering the diagnostic.
//
// By using [`Chars::ascii()`] you can switch to an ASCII-only format
// suitable for rendering on terminals that do not support box drawing
// characters.
type Chars struct {
	/// The characters to use for the top-left border of the snippet.
	/// Defaults to: `"┌─"` or `"-->"` with [`Chars::ascii()`].
	SnippetStart string
	/// The character to use for the left border of the source.
	/// Defaults to: `'│'` or `'|'` with [`Chars::ascii()`].
	SourceBorderLeft rune
	/// The character to use for the left border break of the source.
	/// Defaults to: `'·'` or `'.'` with [`Chars::ascii()`].
	SourceBorderLeftBreak rune

	/// The character to use for the note bullet.
	/// Defaults to: `'='`.
	NoteBullet rune

	/// The character to use for marking a single-line primary label.
	/// Defaults to: `'^'`.
	SinglePrimaryCaret rune

	/// The character to use for marking a single-line secondary label.
	/// Defaults to: `'-'`.
	SingleSecondaryCaret rune

	/// The character to use for marking the start of a multi-line primary label.
	/// Defaults to: `'^'`.
	MultiPrimaryCaretStart rune

	/// The character to use for marking the end of a multi-line primary label.
	/// Defaults to: `'^'`.
	MultiPrimaryCaretEnd rune

	/// The character to use for marking the start of a multi-line secondary label.
	/// Defaults to: `'\''`.
	MultiSecondaryCaretStart rune

	/// The character to use for marking the end of a multi-line secondary label.
	/// Defaults to: `'\''`.
	MultiSecondaryCaretEnd rune

	/// The character to use for the top-left corner of a multi-line label.
	/// Defaults to: `'╭'` or `'/'` with [`Chars::ascii()`].
	MultiTopLeft rune

	/// The character to use for the top of a multi-line label.
	/// Defaults to: `'─'` or `'-'` with [`Chars::ascii()`].
	MultiTop rune

	/// The character to use for the bottom-left corner of a multi-line label.
	/// Defaults to: `'╰'` or `'\'` with [`Chars::ascii()`].
	MultiBottomLeft rune

	/// The character to use when marking the bottom of a multi-line label.
	/// Defaults to: `'─'` or `'-'` with [`Chars::ascii()`].
	MultiBottom rune

	/// The character to use for the left of a multi-line label.
	/// Defaults to: `'│'` or `'|'` with [`Chars::ascii()`].
	MultiLeft rune

	/// The character to use for the left of a pointer underneath a caret.
	/// Defaults to: `'│'` or `'|'` with [`Chars::ascii()`].
	PointerLeft rune
}

// A character set that uses Unicode box drawing characters.
func BoxDrawing() Chars {
	return Chars{
		SnippetStart:          "┌─",
		SourceBorderLeft:      '│',
		SourceBorderLeftBreak: '·',

		NoteBullet: '=',

		SinglePrimaryCaret:   '^',
		SingleSecondaryCaret: '-',

		MultiPrimaryCaretStart:   '^',
		MultiPrimaryCaretEnd:     '^',
		MultiSecondaryCaretStart: '\'',
		MultiSecondaryCaretEnd:   '\'',
		MultiTopLeft:             '╭',
		MultiTop:                 '─',
		MultiBottomLeft:          '╰',
		MultiBottom:              '─',
		MultiLeft:                '│',

		PointerLeft: '│',
	}
}

// A character set that only uses ASCII characters.
//
// This is useful if your terminal's font does not support box drawing
// characters well and results in output that looks similar to rustc's
// diagnostic output.
func Ascii() Chars {
	return Chars{
		SnippetStart:          "-->",
		SourceBorderLeft:      '|',
		SourceBorderLeftBreak: '.',

		NoteBullet: '=',

		SinglePrimaryCaret:   '^',
		SingleSecondaryCaret: '-',

		MultiPrimaryCaretStart:   '^',
		MultiPrimaryCaretEnd:     '^',
		MultiSecondaryCaretStart: '\'',
		MultiSecondaryCaretEnd:   '\'',
		MultiTopLeft:             '/',
		MultiTop:                 '-',
		MultiBottomLeft:          '\\',
		MultiBottom:              '-',
		MultiLeft:                '|',

		PointerLeft: '|',
	}
}
