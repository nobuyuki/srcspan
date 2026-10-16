package term

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nobuyuki/srcspan/diagnostic"
	"github.com/nobuyuki/srcspan/files"
)

var errWriteFailed = errors.New("write failed")

// A writer that fails once more than `limit` bytes have been written.
type failingWriter struct {
	limit   int
	written int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	if w.written+len(p) > w.limit {
		return 0, errWriteFailed
	}
	w.written += len(p)
	return len(p), nil
}

// A write error at any point of the output is returned to the caller.
func TestEmitReturnsWriteErrors(t *testing.T) {
	source := "fn f() {\n    let a = {\n        1\n    };\n}\nabc def\n"

	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", source)
	inner_start := strings.Index(source, "{\n        1")
	inner_end := strings.Index(source, "};") + 1
	abc := strings.Index(source, "abc")

	d := diagnostic.NewError().WithCode("E0001").WithMessage("message").WithLabels([]diagnostic.Label{
		diagnostic.NewPrimary(file_id, files.NewRange(0, inner_end+3)).WithMessage("outer"),
		diagnostic.NewSecondary(file_id, files.NewRange(inner_start, inner_end)).WithMessage("inner"),
		diagnostic.NewPrimary(file_id, files.NewRange(abc, abc+3)).WithMessage("abc"),
		diagnostic.NewSecondary(file_id, files.NewRange(abc+1, abc+2)).WithMessage("b"),
		diagnostic.NewSecondary(file_id, files.NewRange(abc+7, abc+7)).WithMessage("end"),
	}).WithNotes([]string{"note", "multi\nline"})

	for _, style := range []DisplayStyle{Rich, Medium, Short} {
		config := DefaultConfig()
		config.DisplayStyle = style

		full, err := EmitIntoString(&config, &fs, d)
		require.NoError(t, err)

		for limit := range len(full) {
			w := &failingWriter{limit: limit}
			err := EmitToString(w, &config, &fs, d)
			require.ErrorIs(t, err, errWriteFailed, "style=%v limit=%d", style, limit)
		}
	}
}

// A label pointing to an unknown file is reported as an error.
func TestEmitFileMissing(t *testing.T) {
	fs := files.NewSimpleFiles()
	fs.Add("test", "abc\n")
	d := diagnostic.NewError().WithMessage("message").WithLabel(
		diagnostic.NewPrimary(1, files.NewRange(0, 1)),
	)

	for _, style := range []DisplayStyle{Rich, Medium, Short} {
		config := DefaultConfig()
		config.DisplayStyle = style

		require.NotPanics(t, func() {
			_, err := EmitIntoString(&config, &fs, d)
			require.ErrorIs(t, err, files.FileMissingError("1"), "style=%v", style)
		})
	}
}
