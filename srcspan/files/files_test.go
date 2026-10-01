package files

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const TEST_SOURCE = "foo\nbar\r\n\nbaz"

// (`codespan_reporting::files::test::line_starts` in Rust)
func TestLineStarts(t *testing.T) {
	file := NewSimpleFile("test", TEST_SOURCE)

	assert.Equal(t,
		[]int{
			0,  // "foo\n"
			4,  // "bar\r\n"
			9,  // ""
			10, // "baz"
		},
		file.line_starts,
	)
}

// (`codespan_reporting::files::test::line_span_sources` in Rust)
func TestLineSpanSources(t *testing.T) {
	file := NewSimpleFile("test", TEST_SOURCE)

	line_sources := []string{}
	for line := range 4 {
		line_range, err := file.LineRange(0, line)
		require.NoError(t, err)
		line_sources = append(line_sources, file.source[line_range.Start:line_range.End])
	}

	assert.Equal(t, []string{"foo\n", "bar\r\n", "\n", "baz"}, line_sources)
}

func TestIsRuneBoundary(t *testing.T) {
	source := "aα" // 'α' is 2 bytes: [1, 3)

	assert.True(t, IsRuneBoundary(source, 0))
	assert.True(t, IsRuneBoundary(source, 1))
	assert.False(t, IsRuneBoundary(source, 2))
	assert.True(t, IsRuneBoundary(source, 3))
	assert.False(t, IsRuneBoundary(source, -1))
	assert.False(t, IsRuneBoundary(source, 4))
}

func TestSimpleFilesLocation(t *testing.T) {
	source := "let αβγ = 1;\nfizz₁ : Nat"

	fs := NewSimpleFiles()
	file_id := fs.Add("test", source)

	// 1-based, and the column is counted in characters.
	location, err := fs.Location(file_id, len("let αβγ = "))
	require.NoError(t, err)
	assert.Equal(t, Location{LineNumber: 1, ColumnNumber: 11}, location)

	location, err = fs.Location(file_id, len(source))
	require.NoError(t, err)
	assert.Equal(t, Location{LineNumber: 2, ColumnNumber: 12}, location)
}

func TestLineRangeTooLarge(t *testing.T) {
	fs := NewSimpleFiles()
	file_id := fs.Add("test", TEST_SOURCE)

	// TEST_SOURCE has 4 lines (indices 0..3).
	_, err := fs.LineRange(file_id, 4)
	require.Error(t, err)
}

func TestFileMissing(t *testing.T) {
	fs := NewSimpleFiles()
	fs.Add("test", TEST_SOURCE)

	for _, file_id := range []int{-1, 1} {
		missing := FileMissingError(strconv.Itoa(file_id))

		_, err := fs.Get(file_id)
		require.ErrorIs(t, err, missing)

		_, err = fs.Name(file_id)
		require.ErrorIs(t, err, missing)

		_, err = fs.Source(file_id)
		require.ErrorIs(t, err, missing)

		_, err = fs.LineIndex(file_id, 0)
		require.ErrorIs(t, err, missing)

		_, err = fs.LineRange(file_id, 0)
		require.ErrorIs(t, err, missing)

		_, err = fs.ColumnNumber(file_id, 0, 0)
		require.ErrorIs(t, err, missing)

		_, err = fs.Location(file_id, 0)
		require.ErrorIs(t, err, missing)
	}
}
