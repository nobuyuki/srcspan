package srcspan

import (
	"strings"
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

func TestLocationEndOfFile(t *testing.T) {
	file := NewFile("test", TEST_SOURCE1)
	source_len := len(TEST_SOURCE1)

	location, err := file.Location(source_len)
	require.NoError(t, err)
	assert.Equal(t, NewLocation(3, 3), location)

	_, err = file.Location(source_len + 1)
	assert.ErrorIs(t, err, IndexTooLargeError{Given: source_len + 1, Max: source_len - 1})
}

func TestLocationEmptyFile(t *testing.T) {
	file := NewFile("test", "")

	location, err := file.Location(0)
	require.NoError(t, err)
	assert.Equal(t, NewLocation(0, 0), location)
}

func TestSourceSliceEndOfFile(t *testing.T) {
	file := NewFile("test", TEST_SOURCE1)
	source_len := len(TEST_SOURCE1)

	slice, err := file.SourceSlice(NewSpan(source_len, source_len))
	require.NoError(t, err)
	assert.Empty(t, slice)

	empty := NewFile("empty", "")
	slice, err = empty.SourceSlice(DefaultSpan())
	require.NoError(t, err)
	assert.Empty(t, slice)

	_, err = file.SourceSlice(NewSpan(source_len+1, source_len+1))
	assert.ErrorIs(t, err, IndexTooLargeError{Given: source_len + 1, Max: source_len})
}

func TestLocationCountsCharacters(t *testing.T) {
	source := "let αβγ = 1;\nfizz₁ : Nat"
	file := NewFile("test", source)

	location, err := file.Location(strings.Index(source, "1;"))
	require.NoError(t, err)
	assert.Equal(t, NewLocation(0, 10), location)

	location, err = file.Location(strings.Index(source, "Nat"))
	require.NoError(t, err)
	assert.Equal(t, NewLocation(1, 8), location)
}

func TestSourceSliceStartAfterEnd(t *testing.T) {
	file := NewFile("test", TEST_SOURCE1)

	require.NotPanics(t, func() {
		_, err := file.SourceSlice(NewSpan(3, 1))
		assert.ErrorIs(t, err, IndexTooLargeError{Given: 1, Max: len(TEST_SOURCE1) - 1})
	})
}

func TestSourceSliceCharBoundary(t *testing.T) {
	source := "aαb" // 'α' is 2 bytes: [1, 3)
	file := NewFile("test", source)

	slice, err := file.SourceSlice(NewSpan(1, 3))
	require.NoError(t, err)
	assert.Equal(t, "α", slice)

	// Reported as `IndexTooLarge` like codespan.
	_, err = file.SourceSlice(NewSpan(2, 3))
	assert.ErrorIs(t, err, IndexTooLargeError{Given: 3, Max: len(source) - 1})

	_, err = file.SourceSlice(NewSpan(0, 2))
	assert.ErrorIs(t, err, IndexTooLargeError{Given: 2, Max: len(source) - 1})
}
