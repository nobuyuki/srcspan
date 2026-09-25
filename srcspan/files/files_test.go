package files

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
