package term

import (
	"bytes"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobuyuki/srcspan/diagnostic"
	"github.com/nobuyuki/srcspan/files"
)

func emitWithColorOutput(t *testing.T, color_output ColorOutput) (written, stdout string) {
	t.Helper()

	// Escape sequences must never go to `color.Output` (stdout).
	var out bytes.Buffer
	saved_output := color.Output
	color.Output = &out
	t.Cleanup(func() { color.Output = saved_output })

	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", "abc")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		diagnostic.NewPrimary(file_id, files.NewRange(0, 1)),
	)

	styles := DefaultColor()
	styles.ColorOutput = color_output

	var buf bytes.Buffer
	config := DefaultConfig()
	require.NoError(t, EmitToWriteStyle(NewStylesWriter(&buf, styles), &config, &fs, d))

	return buf.String(), out.String()
}

func TestStylesWriterForce(t *testing.T) {
	saved_no_color := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = saved_no_color })

	written, stdout := emitWithColorOutput(t, ColorOutputForce)

	assert.Contains(t, written, "\x1b[91merror")
	assert.Empty(t, stdout)
	assert.True(t, color.NoColor, "global color.NoColor must not be modified")
}

func TestStylesWriterNone(t *testing.T) {
	saved_no_color := color.NoColor
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = saved_no_color })

	written, stdout := emitWithColorOutput(t, ColorOutputNone)

	assert.NotContains(t, written, "\x1b[")
	assert.Contains(t, written, "error: message")
	assert.Empty(t, stdout)
	assert.False(t, color.NoColor, "global color.NoColor must not be modified")
}
