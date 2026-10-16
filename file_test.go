package srcspan

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobuyuki/srcspan/diagnostic"
	"github.com/nobuyuki/srcspan/files"
	"github.com/nobuyuki/srcspan/term"
)

const TEST_SOURCE1 = "foo\nbar\r\n\nbaz"

func TestLineStart1(t *testing.T) {
	file := newFile("test", TEST_SOURCE1)

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
	file := newFile("test", TEST_SOURCE1)

	line_sources := []string{}

	for line := range 4 {
		line_span, err := file.lineSpan(line)
		require.NoError(t, err)

		slice, err1 := file.sourceSlice(line_span)
		require.NoError(t, err1)

		line_sources = append(line_sources, slice)
	}

	assert.Equal(t,
		[]string{"foo\n", "bar\r\n", "\n", "baz"},
		line_sources,
	)
}

func TestLocationEndOfFile(t *testing.T) {
	file := newFile("test", TEST_SOURCE1)
	source_len := len(TEST_SOURCE1)

	location, err := file.location(source_len)
	require.NoError(t, err)
	assert.Equal(t, NewLocation(3, 3), location)

	_, err = file.location(source_len + 1)
	require.ErrorIs(t, err, files.IndexTooLargeError{Given: source_len + 1, Max: source_len - 1})
}

func TestLocationEmptyFile(t *testing.T) {
	file := newFile("test", "")

	location, err := file.location(0)
	require.NoError(t, err)
	assert.Equal(t, NewLocation(0, 0), location)
}

func TestSourceSliceEndOfFile(t *testing.T) {
	file := newFile("test", TEST_SOURCE1)
	source_len := len(TEST_SOURCE1)

	slice, err := file.sourceSlice(NewSpan(source_len, source_len))
	require.NoError(t, err)
	assert.Empty(t, slice)

	empty := newFile("empty", "")
	slice, err = empty.sourceSlice(DefaultSpan())
	require.NoError(t, err)
	assert.Empty(t, slice)

	_, err = file.sourceSlice(NewSpan(source_len+1, source_len+1))
	require.ErrorIs(t, err, files.IndexTooLargeError{Given: source_len + 1, Max: source_len})
}

func TestLocationCountsCharacters(t *testing.T) {
	source := "let αβγ = 1;\nfizz₁ : Nat"
	file := newFile("test", source)

	location, err := file.location(strings.Index(source, "1;"))
	require.NoError(t, err)
	assert.Equal(t, NewLocation(0, 10), location)

	location, err = file.location(strings.Index(source, "Nat"))
	require.NoError(t, err)
	assert.Equal(t, NewLocation(1, 8), location)
}

func TestSourceSliceStartAfterEnd(t *testing.T) {
	file := newFile("test", TEST_SOURCE1)

	require.NotPanics(t, func() {
		_, err := file.sourceSlice(NewSpan(3, 1))
		require.ErrorIs(t, err, files.IndexTooLargeError{Given: 1, Max: len(TEST_SOURCE1) - 1})
	})
}

func TestSourceSliceCharBoundary(t *testing.T) {
	source := "aαb" // 'α' is 2 bytes: [1, 3)
	file := newFile("test", source)

	slice, err := file.sourceSlice(NewSpan(1, 3))
	require.NoError(t, err)
	assert.Equal(t, "α", slice)

	// Reported as `IndexTooLarge` like codespan.
	_, err = file.sourceSlice(NewSpan(2, 3))
	require.ErrorIs(t, err, files.IndexTooLargeError{Given: 3, Max: len(source) - 1})

	_, err = file.sourceSlice(NewSpan(0, 2))
	require.ErrorIs(t, err, files.IndexTooLargeError{Given: 2, Max: len(source) - 1})
}

// (`codespan::file::test::line_starts` in Rust)
func TestFilesLineStarts(t *testing.T) {
	fs := NewFiles()
	file_id := fs.Add("test", TEST_SOURCE1)

	file, err := fs.get(file_id)
	require.NoError(t, err)
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

// (`codespan::file::test::line_span_sources` in Rust)
func TestFilesLineSpanSources(t *testing.T) {
	fs := NewFiles()
	file_id := fs.Add("test", TEST_SOURCE1)

	line_sources := []string{}
	for line := range 4 {
		line_span, err := fs.LineSpan(file_id, line)
		require.NoError(t, err)
		slice, err := fs.SourceSlice(file_id, line_span)
		require.NoError(t, err)
		line_sources = append(line_sources, slice)
	}

	assert.Equal(t, []string{"foo\n", "bar\r\n", "\n", "baz"}, line_sources)
}

// `Files` can be used for rendering diagnostics.
// (`codespan::file::test::interoperability` in Rust)
func TestFilesInteroperability(t *testing.T) {
	fs := NewFiles()
	file_id := fs.Add("test", TEST_SOURCE1)

	d := diagnostic.NewNote().WithMessage("middle").WithLabels([]diagnostic.Label{
		diagnostic.NewPrimary(file_id, NewSpan(4, 7).Range()).WithMessage("middle"),
	})

	config := term.DefaultConfig()
	out, err := term.EmitIntoString(&config, &fs, d)
	require.NoError(t, err)
	assert.Contains(t, out, "test:2:1")
}

func TestFilesUpdate(t *testing.T) {
	fs := NewFiles()
	file_id := fs.Add("test", "foo")

	require.NoError(t, fs.Update(file_id, TEST_SOURCE1))

	source, err := fs.Source(file_id)
	require.NoError(t, err)
	assert.Equal(t, TEST_SOURCE1, source)

	line_span, err := fs.LineSpan(file_id, 3)
	require.NoError(t, err)
	assert.Equal(t, NewSpan(10, 13), line_span)
}

func TestFilesFileMissing(t *testing.T) {
	fs := NewFiles()
	fs.Add("test", TEST_SOURCE1)

	for _, file_id := range []int{-1, 1} {
		missing := files.FileMissingError(strconv.Itoa(file_id))

		require.ErrorIs(t, fs.Update(file_id, ""), missing)

		_, err := fs.LineSpan(file_id, 0)
		require.ErrorIs(t, err, missing)

		_, err = fs.SourceSpan(file_id)
		require.ErrorIs(t, err, missing)

		_, err = fs.SourceSlice(file_id, DefaultSpan())
		require.ErrorIs(t, err, missing)

		_, err = fs.Name(file_id)
		require.ErrorIs(t, err, missing)

		_, err = fs.Location(file_id, 0)
		require.ErrorIs(t, err, missing)
	}
}

// The doc test of `codespan::Files::location` in Rust, which is for the
// 0-based location. (Not exposed in Go, see `Files`.)
func TestFileLocationDocTest(t *testing.T) {
	file := newFile("test", "foo\nbar\r\n\nbaz")

	location, err := file.location(0)
	require.NoError(t, err)
	assert.Equal(t, NewLocation(0, 0), location)

	location, err = file.location(7)
	require.NoError(t, err)
	assert.Equal(t, NewLocation(1, 3), location)

	location, err = file.location(8)
	require.NoError(t, err)
	assert.Equal(t, NewLocation(1, 4), location)

	location, err = file.location(9)
	require.NoError(t, err)
	assert.Equal(t, NewLocation(2, 0), location)

	_, err = file.location(100)
	require.Error(t, err)
}
