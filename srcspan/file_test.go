package srcspan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const TEST_SOURCE1 = "foo\nbar\r\n\nbaz"

func TestLineStart1(t *testing.T) {
	file := NewFile("test", TEST_SOURCE1)

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

func TestLineSpanSources1(t *testing.T) {
	file := NewFile("test", TEST_SOURCE1)

	line_sources := []string{}

	for line := range 4 {
		line_span, err := file.LineSpan(line)
		require.NoError(t, err)

		slice, err1 := file.SourceSlice(line_span)
		require.NoError(t, err1)

		line_sources = append(line_sources, slice)
	}

	assert.Equal(t,
		[]string{"foo\n", "bar\r\n", "\n", "baz"},
		line_sources,
	)
}
