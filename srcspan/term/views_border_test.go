package term

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srcspan/srcspan/diagnostic"
	"srcspan/srcspan/files"
)

func emitString(t *testing.T, fs files.Files, d *diagnostic.Diagnostic) string {
	t.Helper()

	config := DefaultConfig()
	out, err := EmitIntoString(&config, fs, d)
	require.NoError(t, err)
	return out
}

// No trailing border after the last snippet when there are no notes.
func TestTrailingBorderWithoutNotes(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", "abc\n")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		*diagnostic.NewPrimary(file_id, files.NewRange(0, 1)).WithMessage("label"),
	)

	assert.Equal(t,
		"error: message\n"+
			"  ┌─ test:1:1\n"+
			"  │\n"+
			"1 │ abc\n"+
			"  │ ^ label\n"+
			"\n",
		emitString(t, &fs, d),
	)
}

// The trailing border is kept before notes.
func TestTrailingBorderWithNotes(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", "abc\n")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		*diagnostic.NewPrimary(file_id, files.NewRange(0, 1)).WithMessage("label"),
	).WithNote("note")

	assert.Equal(t,
		"error: message\n"+
			"  ┌─ test:1:1\n"+
			"  │\n"+
			"1 │ abc\n"+
			"  │ ^ label\n"+
			"  │\n"+
			"  = note\n"+
			"\n",
		emitString(t, &fs, d),
	)
}

// The trailing border is kept between files, but not after the last one.
func TestTrailingBorderMultipleFiles(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id1 := fs.Add("one", "abc\n")
	file_id2 := fs.Add("two", "def\n")
	d := diagnostic.NewError().WithMessage("message").WithLabels([]diagnostic.Label{
		*diagnostic.NewPrimary(file_id1, files.NewRange(0, 1)).WithMessage("first"),
		*diagnostic.NewSecondary(file_id2, files.NewRange(0, 1)).WithMessage("second"),
	})

	assert.Equal(t,
		"error: message\n"+
			"  ┌─ one:1:1\n"+
			"  │\n"+
			"1 │ abc\n"+
			"  │ ^ first\n"+
			"  │\n"+
			"  ┌─ two:1:1\n"+
			"  │\n"+
			"1 │ def\n"+
			"  │ - second\n"+
			"\n",
		emitString(t, &fs, d),
	)
}

// A single skipped line inside a multi-line label is rendered with the
// label's left border, even when the line index exceeds the number of
// collected lines.
func TestSkippedLineKeepsMultiLabelBorder(t *testing.T) {
	source := ""
	for i := range 20 {
		source += fmt.Sprintf("line%02d\n", i)
	}

	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", source)
	start := strings.Index(source, "line10")
	end := strings.Index(source, "line16") + len("line16")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		*diagnostic.NewPrimary(file_id, files.NewRange(start, end)).WithMessage("multi"),
	)

	out := emitString(t, &fs, d)
	assert.Contains(t, out, "15 │ │ line14\n")
}

// `AfterLabelLines` must stop at the end of the file.
func TestAfterLabelLinesStopAtEndOfFile(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", "abc\ndef")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		*diagnostic.NewPrimary(file_id, files.NewRange(4, 5)),
	)

	config := DefaultConfig()
	config.AfterLabelLines = 3
	out, err := EmitIntoString(&config, &fs, d)
	require.NoError(t, err)

	assert.Equal(t,
		"error: message\n"+
			"  ┌─ test:2:1\n"+
			"  │\n"+
			"2 │ def\n"+
			"  │ ^\n"+
			"\n",
		out,
	)
}

// A multi-line label starting after tab indentation is drawn from the
// gutter only, without an extra top caret line.
func TestMultiLineLabelAfterTabIndent(t *testing.T) {
	source := "func f() {\n\tif x {\n\t\ty()\n\t}\n}\n"

	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", source)
	start := strings.Index(source, "if")
	end := strings.Index(source, "\t}") + len("\t}")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		*diagnostic.NewPrimary(file_id, files.NewRange(start, end)).WithMessage("block"),
	)

	assert.Equal(t,
		"error: message\n"+
			"  ┌─ test:2:2\n"+
			"  │  \n"+
			"2 │ ╭     if x {\n"+
			"3 │ │         y()\n"+
			"4 │ │     }\n"+
			"  │ ╰─────^ block\n"+
			"\n",
		emitString(t, &fs, d),
	)
}

func emitStringWithStyle(t *testing.T, style DisplayStyle, fs files.Files, d *diagnostic.Diagnostic) string {
	t.Helper()

	config := DefaultConfig()
	config.DisplayStyle = style
	out, err := EmitIntoString(&config, fs, d)
	require.NoError(t, err)
	return out
}

// Short and medium diagnostics print a header for primary labels only.
func TestShortDiagnosticPrimaryLabelsOnly(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", "abc\ndef\n")
	d := diagnostic.NewError().WithMessage("message").WithLabels([]diagnostic.Label{
		*diagnostic.NewSecondary(file_id, files.NewRange(0, 1)),
		*diagnostic.NewPrimary(file_id, files.NewRange(4, 5)),
	}).WithNote("note")

	assert.Equal(t,
		"test:2:1: error: message\n",
		emitStringWithStyle(t, Short, &fs, d),
	)
	assert.Equal(t,
		"test:2:1: error: message\n"+
			" = note\n",
		emitStringWithStyle(t, Medium, &fs, d),
	)
}

// Without primary labels, a header without a locus is printed.
func TestShortDiagnosticWithoutPrimaryLabels(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", "abc\n")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		*diagnostic.NewSecondary(file_id, files.NewRange(0, 1)),
	)

	assert.Equal(t,
		"error: message\n",
		emitStringWithStyle(t, Short, &fs, d),
	)
}

// Notes without any labels are indented by a single space.
func TestNotesWithoutLabels(t *testing.T) {
	fs := files.NewSimpleFiles()
	d := diagnostic.NewError().WithMessage("message").WithNote("note")

	assert.Equal(t,
		"error: message\n"+
			" = note\n"+
			"\n",
		emitString(t, &fs, d),
	)
}
