package term

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srcspan/srcspan"
	"srcspan/srcspan/diagnostic"
	"srcspan/srcspan/files"
)

func emitString(t *testing.T, fs *files.Files, d *diagnostic.Diagnostic) string {
	t.Helper()

	config := DefaultConfig()
	out, err := EmitIntoString(&config, fs, d)
	require.NoError(t, err)
	return out
}

// No trailing border after the last snippet when there are no notes.
func TestTrailingBorderWithoutNotes(t *testing.T) {
	fs := files.NewFiles()
	file_id := fs.Add("test", "abc\n")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		*diagnostic.NewPrimary(file_id, srcspan.NewSpan(0, 1)).WithMessage("label"),
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
	fs := files.NewFiles()
	file_id := fs.Add("test", "abc\n")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		*diagnostic.NewPrimary(file_id, srcspan.NewSpan(0, 1)).WithMessage("label"),
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
	fs := files.NewFiles()
	file_id1 := fs.Add("one", "abc\n")
	file_id2 := fs.Add("two", "def\n")
	d := diagnostic.NewError().WithMessage("message").WithLabels([]diagnostic.Label{
		*diagnostic.NewPrimary(file_id1, srcspan.NewSpan(0, 1)).WithMessage("first"),
		*diagnostic.NewSecondary(file_id2, srcspan.NewSpan(0, 1)).WithMessage("second"),
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

	fs := files.NewFiles()
	file_id := fs.Add("test", source)
	start := strings.Index(source, "line10")
	end := strings.Index(source, "line16") + len("line16")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		*diagnostic.NewPrimary(file_id, srcspan.NewSpan(start, end)).WithMessage("multi"),
	)

	out := emitString(t, &fs, d)
	assert.Contains(t, out, "15 │ │ line14\n")
}
