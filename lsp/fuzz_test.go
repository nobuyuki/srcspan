package lsp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nobuyuki/srcspan/files"
)

func fuzzSeeds(f *testing.F) {
	f.Add("", 0, 0)
	f.Add("let x = 1", 4, 9)
	f.Add("a\r\nb\r\nc", 1, 4)
	f.Add("日本語\n🦀 x\n", 3, 12)
	f.Add("abc\xffdef\n", 3, 4)
	f.Add("abc", 3, 1)
	f.Add("abc", -1, 100)
}

// Translating any byte span must not panic. A byte index on a character
// boundary must be translated back to the same index.
func FuzzByteSpanToRange(f *testing.F) {
	fuzzSeeds(f)

	f.Fuzz(func(t *testing.T, source string, start int, end int) {
		fs := files.NewSimpleFiles()
		file_id := fs.Add("test", source)

		_, _ = ByteSpanToRange(&fs, file_id, files.NewRange(start, end))

		for _, byte_index := range []int{start, end} {
			position, err := ByteIndexToPosition(&fs, file_id, byte_index)
			if err != nil {
				continue
			}
			got, err := PositionToByteIndex(&fs, file_id, position)
			require.NoError(t, err)
			require.Equal(t, byte_index, got)
		}
	})
}

// Translating any LSP range must not panic.
func FuzzRangeToByteSpan(f *testing.F) {
	f.Add("", uint32(0), uint32(0), uint32(0), uint32(0))
	f.Add("let x = 1", uint32(0), uint32(4), uint32(0), uint32(9))
	f.Add("日本語\n🦀 x\n", uint32(0), uint32(1), uint32(1), uint32(2))
	f.Add("abc\xffdef\n", uint32(0), uint32(3), uint32(5), uint32(0))

	f.Fuzz(func(t *testing.T, source string, start_line, start_character, end_line, end_character uint32) {
		fs := files.NewSimpleFiles()
		file_id := fs.Add("test", source)

		_, _ = RangeToByteSpan(&fs, file_id, Range{
			Start: Position{Line: start_line, Character: start_character},
			End:   Position{Line: end_line, Character: end_character},
		})
	})
}
