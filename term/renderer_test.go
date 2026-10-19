package term

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srcspan/diagnostic"
	"srcspan/files"
)

// A zero-width label at the end of a line still gets a caret.
func TestCaretAtEndOfLine(t *testing.T) {
	source := "let x = 1"

	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", source)
	d := diagnostic.NewError().WithMessage("expected `;`").WithLabel(
		diagnostic.NewPrimary(file_id, files.NewRange(len(source), len(source))).WithMessage("here"),
	)

	assert.Equal(t,
		"error: expected `;`\n"+
			"  ┌─ test:1:10\n"+
			"  │\n"+
			"1 │ let x = 1\n"+
			"  │          ^ here\n"+
			"\n",
		emitString(t, &fs, d),
	)
}

// A label covering a tab underlines the whole expanded tab.
func TestCaretOnTab(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", "a\tb")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		diagnostic.NewPrimary(file_id, files.NewRange(1, 3)).WithMessage("tab and b"),
	)

	assert.Equal(t,
		"error: message\n"+
			"  ┌─ test:1:2\n"+
			"  │\n"+
			"1 │ a   b\n"+
			"  │  ^^^^ tab and b\n"+
			"\n",
		emitString(t, &fs, d),
	)
}

// A zero-width label at the end of a line is still pointed at when it
// hangs below the carets.
func TestHangingLabelAtEndOfLine(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", "abc")
	d := diagnostic.NewError().WithMessage("message").WithLabels([]diagnostic.Label{
		diagnostic.NewPrimary(file_id, files.NewRange(0, 3)).WithMessage("whole"),
		diagnostic.NewSecondary(file_id, files.NewRange(3, 3)).WithMessage("end"),
	})

	assert.Equal(t,
		"error: message\n"+
			"  ┌─ test:1:1\n"+
			"  │\n"+
			"1 │ abc\n"+
			"  │ ^^^- end\n"+
			"  │ │  \n"+
			"  │ whole\n"+
			"\n",
		emitString(t, &fs, d),
	)
}

// The column in the locus is counted in characters, not bytes.
func TestLocusColumnCountsCharacters(t *testing.T) {
	source := "let αβγ = 1;"

	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", source)
	start := strings.Index(source, "1")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		diagnostic.NewPrimary(file_id, files.NewRange(start, start+1)),
	)

	assert.Contains(t, emitString(t, &fs, d), "┌─ test:1:11\n")

	config := DefaultConfig()
	config.DisplayStyle = Short
	out, err := EmitIntoString(&config, &fs, d)
	require.NoError(t, err)
	assert.Equal(t, "test:1:11: error: message\n", out)
}

// Multi-line labels use the top/start and bottom/end characters for their
// top and bottom respectively.
func TestMultiLineCaretStartAndEnd(t *testing.T) {
	source := "a (\n  b\n)\n"

	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", source)
	d := diagnostic.NewError().WithMessage("message").WithLabels([]diagnostic.Label{
		diagnostic.NewPrimary(file_id, files.NewRange(2, 9)).WithMessage("primary"),
	})

	config := DefaultConfig()
	config.Chars.MultiPrimaryCaretStart = 'S'
	config.Chars.MultiPrimaryCaretEnd = 'E'
	config.Chars.MultiTop = 't'
	config.Chars.MultiBottom = 'b'
	out, err := EmitIntoString(&config, &fs, d)
	require.NoError(t, err)

	assert.Equal(t,
		"error: message\n"+
			"  ┌─ test:1:3\n"+
			"  │  \n"+
			"1 │   a (\n"+
			"  │ ╭tttS\n"+
			"2 │ │   b\n"+
			"3 │ │ )\n"+
			"  │ ╰bE primary\n"+
			"\n",
		out,
	)
}

// A `WriteStyle` that writes markers for label styling calls.
type labelMarkerWriter struct {
	strings.Builder
}

func (*labelMarkerWriter) SetHeader(_ diagnostic.Severity) error { return nil }
func (*labelMarkerWriter) SetHeaderMessage() error               { return nil }
func (*labelMarkerWriter) SetLineNumber() error                  { return nil }
func (*labelMarkerWriter) SetNoteBullet() error                  { return nil }
func (*labelMarkerWriter) SetSourceBorder() error                { return nil }

func (w *labelMarkerWriter) SetLabel(_ diagnostic.Severity, label_style diagnostic.LabelStyle) error {
	if label_style == diagnostic.Primary {
		w.WriteString("<P>")
	} else {
		w.WriteString("<S>")
	}
	return nil
}

func (w *labelMarkerWriter) Reset() error {
	w.WriteString("<R>")
	return nil
}

// The label style is only switched when it changes along the caret line.
func TestCaretLineSwitchesStyleOnlyOnChange(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", "abc def\n")
	d := diagnostic.NewError().WithMessage("message").WithLabels([]diagnostic.Label{
		diagnostic.NewPrimary(file_id, files.NewRange(0, 3)),
		diagnostic.NewSecondary(file_id, files.NewRange(4, 7)),
	})

	w := &labelMarkerWriter{}
	config := DefaultConfig()
	require.NoError(t, EmitToWriteStyle(w, &config, &fs, d))

	assert.Contains(t, w.String(), "  │<R> <P>^^^<R> <S>---<R>\n")
}

func TestLines(t *testing.T) {
	assert.Empty(t, lines(""))
	assert.Equal(t, []string{""}, lines("\n"))
	assert.Equal(t, []string{"a"}, lines("a"))
	assert.Equal(t, []string{"a"}, lines("a\n"))
	assert.Equal(t, []string{"a", ""}, lines("a\n\n"))
	assert.Equal(t, []string{"a", "b"}, lines("a\r\nb\r\n"))
}

// A trailing newline in a note does not produce an empty note line.
func TestNoteWithTrailingNewline(t *testing.T) {
	fs := files.NewSimpleFiles()
	d := diagnostic.NewError().WithMessage("message").WithNote("first\r\nsecond\n")

	assert.Equal(t,
		"error: message\n"+
			" = first\n"+
			"   second\n"+
			"\n",
		emitString(t, &fs, d),
	)
}
