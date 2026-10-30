package main

import (
	"os"
	"strings"

	"github.com/nobuyuki/srcspan/diagnostic"
	"github.com/nobuyuki/srcspan/files"
	"github.com/nobuyuki/srcspan/term"
)

func main() {
	source := "let x = 1 + \"two\";\n"

	// 1. Register the source. Add returns an id to refer to the file.
	fs := files.NewSimpleFiles()
	file_id := fs.Add("main.calc", source)

	// 2. Build the diagnostic. Ranges are byte offsets into the source.
	start := strings.Index(source, `"two"`)
	d := diagnostic.NewError().
		WithMessage("mismatched types").
		WithCode("E0308").
		WithLabel(diagnostic.NewPrimary(file_id, files.NewRange(start, start+5)).
			WithMessage("expected number, found string")).
		WithLabel(diagnostic.NewSecondary(file_id, files.NewRange(8, 9)).
			WithMessage("this is a number")).
		WithNote("strings can not be added to numbers")

	// 3. Emit it to stderr, with colors.
	writer := term.NewStylesWriter(os.Stderr, term.DefaultStyles())
	config := term.DefaultConfig()
	if err := term.EmitToWriteStyle(writer, &config, &fs, d); err != nil {
		panic(err)
	}
}
