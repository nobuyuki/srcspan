package srcspan

import (
	"testing"
)

// The methods of `Files` must not panic for any source and byte indices.
func FuzzFiles(f *testing.F) {
	f.Add("", "", 0, 0)
	f.Add("let x = 1", "let y = 2\n", 4, 9)
	f.Add("a\r\nb\r\nc", "a\nb", 1, 4)
	f.Add("日本語\n🦀 x\n", "", 1, 12)
	f.Add("abc\xffdef\n", "abc", 3, 4)
	f.Add("abc", "abc", 3, 1)
	f.Add("abc", "abc", -1, 100)

	f.Fuzz(func(t *testing.T, source string, updated string, start int, end int) {
		fs := NewFiles()
		file_id := fs.Add("test", source)

		check := func() {
			_, _ = fs.SourceSpan(file_id)
			_, _ = fs.SourceSlice(file_id, NewSpan(start, end))
			for _, index := range []int{start, end} {
				_, _ = fs.LineIndex(file_id, index)
				_, _ = fs.Location(file_id, index)
				_, _ = fs.LineSpan(file_id, index)
				_, _ = fs.LineRange(file_id, index)
				_, _ = fs.ColumnNumber(file_id, index, end)
			}
		}

		check()
		if err := fs.Update(file_id, updated); err != nil {
			t.Fatal(err)
		}
		check()
	})
}
