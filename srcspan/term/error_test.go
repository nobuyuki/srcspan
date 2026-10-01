package term

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"srcspan/srcspan"
	"srcspan/srcspan/diagnostic"
	"srcspan/srcspan/files"
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

	fs := files.NewFiles()
	file_id := fs.Add("test", source)
	inner_start := strings.Index(source, "{\n        1")
	inner_end := strings.Index(source, "};") + 1
	abc := strings.Index(source, "abc")

	d := diagnostic.NewError().WithCode("E0001").WithMessage("message").WithLabels([]diagnostic.Label{
		*diagnostic.NewPrimary(file_id, srcspan.NewSpan(0, inner_end+3)).WithMessage("outer"),
		*diagnostic.NewSecondary(file_id, srcspan.NewSpan(inner_start, inner_end)).WithMessage("inner"),
		*diagnostic.NewPrimary(file_id, srcspan.NewSpan(abc, abc+3)).WithMessage("abc"),
		*diagnostic.NewSecondary(file_id, srcspan.NewSpan(abc+1, abc+2)).WithMessage("b"),
		*diagnostic.NewSecondary(file_id, srcspan.NewSpan(abc+7, abc+7)).WithMessage("end"),
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
