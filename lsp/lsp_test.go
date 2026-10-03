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

// A position just after the last character of a line is the end of the line.
func TestPositionAtEndOfLine(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("unicode", UNICODE)

	result, err := PositionToByteIndex(&fs, file_id, Position{Line: 0, Character: 7})
	require.NoError(t, err)
	assert.Equal(t, len(UNICODE), result)

	position, err := ByteIndexToPosition(&fs, file_id, len(UNICODE))
	require.NoError(t, err)
	assert.Equal(t, Position{Line: 0, Character: 7}, position)
}

// Errors are returned for either end of a span or a range.
func TestSpanAndRangeErrors(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("unicode", UNICODE)

	tests := []struct {
		name string
		f    func() error
		want error
	}{
		{
			// A byte index past the end of the file is on the last line,
			// but past the end of that line.
			name: "byte index past the end",
			f: func() error {
				_, err := ByteIndexToPosition(&fs, file_id, len(UNICODE)+3)
				return err
			},
			want: files.ColumnTooLargeError{Given: len(UNICODE) + 3, Max: len(UNICODE)},
		},
		{
			name: "span start inside a UTF-8 sequence",
			f: func() error {
				_, err := ByteSpanToRange(&fs, file_id, files.NewRange(1, 2))
				return err
			},
			want: files.InvalidCharBoundaryError{Given: 1},
		},
		{
			name: "span end inside a UTF-8 sequence",
			f: func() error {
				_, err := ByteSpanToRange(&fs, file_id, files.NewRange(0, 3))
				return err
			},
			want: files.InvalidCharBoundaryError{Given: 3},
		},
		{
			name: "range start past the end of the line",
			f: func() error {
				_, err := RangeToByteSpan(&fs, file_id, Range{Start: Position{Character: 8}, End: Position{Character: 9}})
				return err
			},
			want: files.ColumnTooLargeError{Given: 7, Max: len(UNICODE)},
		},
		{
			name: "range end past the end of the line",
			f: func() error {
				_, err := RangeToByteSpan(&fs, file_id, Range{End: Position{Character: 9}})
				return err
			},
			want: files.ColumnTooLargeError{Given: 7, Max: len(UNICODE)},
		},
		{
			name: "unknown file",
			f: func() error {
				_, err := PositionToByteIndex(&fs, 9, Position{})
				return err
			},
			want: files.FileMissingError("9"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorIs(t, tt.f(), tt.want)
		})
	}
}

// (`source.get(line_span)` in Rust)
func TestLineStr(t *testing.T) {
	source := "aå\nb" // 'å' is 2 bytes

	tests := []struct {
		name string
		span files.Range
		want error
	}{
		{"negative start", files.NewRange(-1, 2), files.IndexTooLargeError{Given: 2, Max: 4}},
		{"start after end", files.NewRange(3, 1), files.IndexTooLargeError{Given: 1, Max: 4}},
		{"end past the source", files.NewRange(0, 6), files.IndexTooLargeError{Given: 6, Max: 4}},
		{"start past the source", files.NewRange(6, 7), files.IndexTooLargeError{Given: 6, Max: 4}},
		{"start inside a UTF-8 sequence", files.NewRange(2, 4), files.IndexTooLargeError{Given: 4, Max: 4}},
		{"end inside a UTF-8 sequence", files.NewRange(0, 2), files.IndexTooLargeError{Given: 2, Max: 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := lineStr(source, tt.span)
			require.ErrorIs(t, err, tt.want)
		})
	}

	got, err := lineStr(source, files.NewRange(0, 4))
	require.NoError(t, err)
	assert.Equal(t, "aå\n", got)
}

// A `files.Files` whose `LineIndex` or `LineRange` fails or returns a range
// outside the source, to check that the errors of an arbitrary
// implementation are returned.
type brokenFiles struct {
	*files.SimpleFiles
	lineIndexErr error
	lineRange    *files.Range
	lineRangeErr error
}

func (fs brokenFiles) LineIndex(file_id int, byte_index int) (int, error) {
	if fs.lineIndexErr != nil {
		return 0, fs.lineIndexErr
	}
	return fs.SimpleFiles.LineIndex(file_id, byte_index)
}

func (fs brokenFiles) LineRange(file_id int, line_index int) (files.Range, error) {
	if fs.lineRangeErr != nil {
		return files.Range{}, fs.lineRangeErr
	}
	if fs.lineRange != nil {
		return *fs.lineRange, nil
	}
	return fs.SimpleFiles.LineRange(file_id, line_index)
}

func TestFilesErrors(t *testing.T) {
	fs := files.NewSimpleFiles()
	file_id := fs.Add("unicode", UNICODE)
	errBroken := files.LineTooLargeError{Given: 1, Max: 0}
	outside := files.NewRange(0, 100)

	tests := []struct {
		name  string
		files brokenFiles
		want  error
	}{
		{"LineIndex fails", brokenFiles{SimpleFiles: &fs, lineIndexErr: errBroken}, errBroken},
		{"LineRange fails", brokenFiles{SimpleFiles: &fs, lineRangeErr: errBroken}, errBroken},
		{"LineRange outside the source", brokenFiles{SimpleFiles: &fs, lineRange: &outside},
			files.IndexTooLargeError{Given: 100, Max: len(UNICODE) - 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ByteIndexToPosition(tt.files, file_id, 0)
			require.ErrorIs(t, err, tt.want)

			if tt.files.lineIndexErr == nil {
				_, err = PositionToByteIndex(tt.files, file_id, Position{})
				require.ErrorIs(t, err, tt.want)
			}
		})
	}
}

// The JSON is the same as the protocol's.
func TestJSON(t *testing.T) {
	data, err := json.Marshal(Range{Start: Position{Line: 1, Character: 2}, End: Position{Line: 3, Character: 4}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"start":{"line":1,"character":2},"end":{"line":3,"character":4}}`, string(data))
}
