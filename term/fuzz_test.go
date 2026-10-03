package term

import (
	"testing"

	"github.com/nobuyuki/srcspan/diagnostic"
	"github.com/nobuyuki/srcspan/files"
)

// Builds labels from `data`, 3 bytes per label: the style and the two ends
// of the range. The ends may be past the end of the source, inside a
// character or in reverse order.
func fuzzLabels(file_id int, source string, data []byte) []diagnostic.Label {
	const maxLabels = 8

	labels := []diagnostic.Label{}
	for i := 0; i+3 <= len(data) && len(labels) < maxLabels; i += 3 {
		style := diagnostic.Primary
		if data[i]&1 == 1 {
			style = diagnostic.Secondary
		}
		start := int(data[i+1]) % (len(source) + 3)
		end := int(data[i+2]) % (len(source) + 3)
		label := diagnostic.NewLabel(style, file_id, files.NewRange(start, end))
		if data[i]&2 == 2 {
			label = label.WithMessage("message\nwith a line break")
		}
		labels = append(labels, *label)
	}
	return labels
}

// Rendering must not panic for any source and labels. Errors (e.g. for a
// range past the end of the source) are fine.
func FuzzEmit(f *testing.F) {
	f.Add("", []byte{0, 0, 0}, uint8(0))
	f.Add("let x = 1", []byte{0, 4, 5, 1, 8, 9}, uint8(0))
	f.Add("fn main() {\n    foo\n}\n", []byte{0, 3, 20, 3, 12, 15, 1, 16, 2}, uint8(1))
	f.Add("a\r\nb\r\nc", []byte{0, 1, 4, 1, 2, 6}, uint8(2))
	f.Add("\t日本語\n🦀 x\n", []byte{0, 1, 4, 0, 2, 12, 1, 12, 14}, uint8(0))
	f.Add("abc\xffdef\n", []byte{0, 2, 5}, uint8(0))

	f.Fuzz(func(_ *testing.T, source string, data []byte, style uint8) {
		fs := files.NewSimpleFiles()
		file_id := fs.Add("test", source)

		d := diagnostic.NewError().
			WithMessage("message").
			WithCode("E0001").
			WithLabels(fuzzLabels(file_id, source, data)).
			WithNote("note")

		config := DefaultConfig()
		config.DisplayStyle = []DisplayStyle{Rich, Medium, Short}[int(style)%3]
		_, _ = EmitIntoString(&config, &fs, d)
	})
}
