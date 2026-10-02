package lsp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobuyuki/srcspan/files"
)

// (`codespan_lsp::tests::position` in Rust)
func TestPosition(t *testing.T) {
	text := `
let test = 2
let test1 = ""
test
`
	fs := files.NewSimpleFiles()
	file_id := fs.Add("test", text)
	pos, err := PositionToByteIndex(&fs, file_id, Position{Line: 3, Character: 2})
	require.NoError(t, err)

	location, err := fs.Location(file_id, pos)
	require.NoError(t, err)
	assert.Equal(t,
		files.Location{
			// One-based
			LineNumber:   3 + 1,
			ColumnNumber: 2 + 1,
		},
		location,
	)
}

// The protocol specifies that each `character` in position is a UTF-16
// character. This means that `å` and `ä` here counts as 1 while `𐐀` counts
// as 2.
const UNICODE = "åä t𐐀b"

// (`codespan_lsp::tests::unicode_get_byte_index` in Rust)
func TestUnicodeGetByteIndex(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("unicode", UNICODE)

	result, err := PositionToByteIndex(&fs, file_id, Position{Line: 0, Character: 3})
	require.NoError(t, err)
	assert.Equal(t, 5, result)

	result, err = PositionToByteIndex(&fs, file_id, Position{Line: 0, Character: 6})
	require.NoError(t, err)
	assert.Equal(t, 10, result)
}

// (`codespan_lsp::tests::unicode_get_position` in Rust)
func TestUnicodeGetPosition(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("unicode", UNICODE)
	file_id2 := fs.Add("unicode newline", "\n"+UNICODE)

	result, err := ByteIndexToPosition(&fs, file_id, 5)
	require.NoError(t, err)
	assert.Equal(t, Position{Line: 0, Character: 3}, result)

	result, err = ByteIndexToPosition(&fs, file_id, 10)
	require.NoError(t, err)
	assert.Equal(t, Position{Line: 0, Character: 6}, result)

	result, err = ByteIndexToPosition(&fs, file_id2, 11)
	require.NoError(t, err)
	assert.Equal(t, Position{Line: 1, Character: 6}, result)
}

func TestRangeRoundTrip(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("unicode newline", "\n"+UNICODE)

	span := files.NewRange(3, 11) // "ä t𐐀" on the second line
	range_, err := ByteSpanToRange(&fs, file_id, span)
	require.NoError(t, err)
	assert.Equal(t, Range{Start: Position{Line: 1, Character: 1}, End: Position{Line: 1, Character: 6}}, range_)

	back, err := RangeToByteSpan(&fs, file_id, range_)
	require.NoError(t, err)
	assert.Equal(t, span, back)
}

// Errors are returned where codespan-lsp panics or returns an error.
func TestErrors(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("unicode", UNICODE)

	// A line past the end of the file (panics in Rust).
	require.NotPanics(t, func() {
		_, err := PositionToByteIndex(&fs, file_id, Position{Line: 5, Character: 0})
		require.Error(t, err)
	})

	// A character past the end of the line.
	_, err := PositionToByteIndex(&fs, file_id, Position{Line: 0, Character: 9})
	// `given` is the UTF-16 length of the line (7), as in Rust.
	require.ErrorIs(t, err, files.ColumnTooLargeError{Given: 7, Max: len(UNICODE)})

	// A byte index inside a UTF-8 sequence ('å' is 2 bytes).
	_, err = ByteIndexToPosition(&fs, file_id, 1)
	require.ErrorIs(t, err, files.InvalidCharBoundaryError{Given: 1})

	// An unknown file.
	_, err = ByteIndexToPosition(&fs, 9, 0)
	require.ErrorIs(t, err, files.FileMissingError("9"))
}

// The JSON is the same as the protocol's.
func TestJSON(t *testing.T) {
	data, err := json.Marshal(Range{Start: Position{Line: 1, Character: 2}, End: Position{Line: 3, Character: 4}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"start":{"line":1,"character":2},"end":{"line":3,"character":4}}`, string(data))
}
