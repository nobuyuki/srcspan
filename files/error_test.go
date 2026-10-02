package files

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The messages are the same as `impl Display for Error` in Rust.
func TestErrorMessages(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{FileMissingError("1"), "file missing"},
		{IndexTooLargeError{Given: 3, Max: 8}, "invalid index 3, maximum index is 8"},
		{LineTooLargeError{Given: 5, Max: 3}, "invalid line 5, maximum line is 3"},
		{ColumnTooLargeError{Given: 9, Max: 4}, "invalid column 9, maximum column 4"},
		{InvalidCharBoundaryError{Given: 2}, "index is not a code point boundary"},
		// Go only
		{NegativeIndexError{Given: -1}, "invalid index -1, index must not be negative"},
		{NegativeLineError{Given: -1}, "invalid line -1, line must not be negative"},
	}

	for _, test := range tests {
		assert.EqualError(t, test.err, test.expected)
	}
}
