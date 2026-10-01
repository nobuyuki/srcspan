package term

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"srcspan/srcspan"
	"srcspan/srcspan/diagnostic"
	"srcspan/srcspan/files"
)

// A zero-width label at the end of a line still gets a caret.
func TestCaretAtEndOfLine(t *testing.T) {
	source := "let x = 1"

	fs := files.NewFiles()
	file_id := fs.Add("test", source)
	d := diagnostic.NewError().WithMessage("expected `;`").WithLabel(
		*diagnostic.NewPrimary(file_id, srcspan.NewSpan(len(source), len(source))).WithMessage("here"),
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
	fs := files.NewFiles()
	file_id := fs.Add("test", "a\tb")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		*diagnostic.NewPrimary(file_id, srcspan.NewSpan(1, 3)).WithMessage("tab and b"),
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
	fs := files.NewFiles()
	file_id := fs.Add("test", "abc")
	d := diagnostic.NewError().WithMessage("message").WithLabels([]diagnostic.Label{
		*diagnostic.NewPrimary(file_id, srcspan.NewSpan(0, 3)).WithMessage("whole"),
		*diagnostic.NewSecondary(file_id, srcspan.NewSpan(3, 3)).WithMessage("end"),
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
