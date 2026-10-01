package files

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srcspan/srcspan"
)

const TEST_SOURCE = "foo\nbar\r\n\nbaz"

func TestLineStart(t *testing.T) {
	files := NewFiles()
	file_id := files.Add("test", TEST_SOURCE)
	file, _ := files.Get(file_id)

	assert.Equal(t,
		[]int{
			0,  // "foo\n"
			4,  // "bar\r\n"
			9,  // ""
			10, // "baz"
		},
		file.LineStarts(),
	)
}

func TestLineSpanSources(t *testing.T) {
	files := NewFiles()
	file_id := files.Add("test", TEST_SOURCE)

	line_sources := []string{}

	for line := range 4 {
		line_span, err := files.LineSpan(file_id, line)
		require.NoError(t, err)

		slice, err1 := files.SourceSlice(file_id, line_span)
		require.NoError(t, err1)

		line_sources = append(line_sources, slice)
	}

	assert.Equal(t,
		[]string{"foo\n", "bar\r\n", "\n", "baz"},
		line_sources,
	)
}

func TestUpdate(t *testing.T) {
	files := NewFiles()
	file_id := files.Add("test", "foo")

	files.Update(file_id, TEST_SOURCE)

	source, err := files.Source(file_id)
	require.NoError(t, err)
	assert.Equal(t, TEST_SOURCE, *source)

	file, err := files.Get(file_id)
	require.NoError(t, err)
	assert.Equal(t, []int{0, 4, 9, 10}, file.LineStarts())
}

func TestLineRangeTooLarge(t *testing.T) {
	files := NewFiles()
	file_id := files.Add("test", TEST_SOURCE)

	// TEST_SOURCE has 4 lines (indices 0..3).
	_, err := files.LineRange(file_id, 4)
	require.Error(t, err)

	_, err = files.LineSpan(file_id, 4)
	require.Error(t, err)
}

func TestFileMissing(t *testing.T) {
	files := NewFiles()
	files.Add("test", TEST_SOURCE)

	for _, file_id := range []int{-1, 1} {
		missing := srcspan.FileMissingError(strconv.Itoa(file_id))

		_, err := files.Get(file_id)
		require.ErrorIs(t, err, missing)

		_, err = files.Name(file_id)
		require.ErrorIs(t, err, missing)

		_, err = files.Source(file_id)
		require.ErrorIs(t, err, missing)

		_, err = files.LineIndex(file_id, 0)
		require.ErrorIs(t, err, missing)

		_, err = files.LineSpan(file_id, 0)
		require.ErrorIs(t, err, missing)

		_, err = files.LineRange(file_id, 0)
		require.ErrorIs(t, err, missing)

		_, err = files.SourceSlice(file_id, srcspan.DefaultSpan())
		require.ErrorIs(t, err, missing)

		_, err = files.Location(file_id, 0)
		require.ErrorIs(t, err, missing)

		_, err = files.ColumnNumber(file_id, 0, 0)
		require.ErrorIs(t, err, missing)
	}
}
