package term

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srcspan/srcspan"
	"srcspan/srcspan/diagnostic"
	"srcspan/srcspan/files"
)

// Test range of 0 to 0 and check does not crash
func TestUnsizedEmit(t *testing.T) {
	fs := files.NewFiles()

	file_id := fs.Add("test", "")
	d := diagnostic.NewBug().WithLabel(*diagnostic.NewPrimary(file_id, srcspan.NewSpan(0, 0)))

	config := DefaultConfig()
	_, err := EmitIntoString(&config, &fs, d)
	require.NoError(t, err)
}

// A label at the end of the file (e.g. "expected `;`") must be rendered.
func TestEmitEndOfFile(t *testing.T) {
	source := "let x = 1"

	fs := files.NewFiles()
	file_id := fs.Add("test", source)
	d := diagnostic.NewError().WithMessage("expected `;`").WithLabel(
		*diagnostic.NewPrimary(file_id, srcspan.NewSpan(len(source), len(source))).WithMessage("here"),
	)

	config := DefaultConfig()
	out, err := EmitIntoString(&config, &fs, d)
	require.NoError(t, err)
	assert.Contains(t, out, "test:1:10")
	assert.Contains(t, out, "here")
}

/*
#[cfg(test)]
mod tests {
    use alloc::vec::Vec;

    use super::*;

    use crate::diagnostic::Label;
    use crate::files::SimpleFiles;

    /// Test range of 0 to 0 and check does not crash
    #[test]
    fn unsized_emit() {
        let mut files = SimpleFiles::new();

        let id = files.add("test", "");
        let zero_range = 0..0;
        let diagnostic = Diagnostic::bug().with_label(Label::primary(id, zero_range));
        emit_into_string(&Config::default(), &files, &diagnostic).unwrap();
    }
    }
*/

// Nested multi-line labels must not panic when an outer label's
// column is drawn before the inner label's underline is started.
func TestNestedMultiLineLabels(t *testing.T) {
	source := "fn f() {\n    let a = {\n        1\n    };\n}\n"

	fs := files.NewFiles()
	file_id := fs.Add("nested", source)

	inner_start := strings.Index(source, "{\n        1")
	inner_end := strings.Index(source, "};") + 1

	d := diagnostic.NewError().WithMessage("nested").WithLabels([]diagnostic.Label{
		*diagnostic.NewPrimary(file_id, srcspan.NewSpan(0, len(source)-1)).WithMessage("outer"),
		*diagnostic.NewSecondary(file_id, srcspan.NewSpan(inner_start, inner_end)).WithMessage("inner"),
	})

	config := DefaultConfig()
	var out string
	require.NotPanics(t, func() {
		var err error
		out, err = EmitIntoString(&config, &fs, d)
		require.NoError(t, err)
	})
	assert.Contains(t, out, "inner")
	assert.Contains(t, out, "outer")
}
