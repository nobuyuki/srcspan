package srcspan_test

// Ports of the doc tests of `codespan::Files` in Rust.
// (The doc tests of `codespan::Span` are in span_test.go.)

import (
	"fmt"

	"srcspan"
)

func ExampleFiles_Name() {
	name := "test"

	files := srcspan.NewFiles()
	file_id := files.Add(name, "hello world!")

	fmt.Println(files.Name(file_id))
	// Output: test <nil>
}

func ExampleFiles_LineSpan() {
	files := srcspan.NewFiles()
	file_id := files.Add("test", "foo\nbar\r\n\nbaz")

	for line := range 4 {
		line_span, _ := files.LineSpan(file_id, line)
		fmt.Println(line, line_span)
	}

	_, err := files.LineSpan(file_id, 4)
	fmt.Println(err != nil)
	// Output:
	// 0 [0, 4)
	// 1 [4, 9)
	// 2 [9, 10)
	// 3 [10, 13)
	// true
}

func ExampleFiles_LineIndex() {
	files := srcspan.NewFiles()
	file_id := files.Add("test", "foo\nbar\r\n\nbaz")

	for _, byte_index := range []int{0, 7, 8, 9, 100} {
		line_index, _ := files.LineIndex(file_id, byte_index)
		fmt.Println(byte_index, line_index)
	}
	// Output:
	// 0 0
	// 7 1
	// 8 1
	// 9 2
	// 100 3
}

// The user-facing (1-based) location of a byte index.
//
// NOTE: The doc test of `codespan::Files::location` in Rust is for the
// 0-based location, which is not exposed in Go (see `Files`). It is ported
// as an internal test instead.
func ExampleFiles_Location() {
	files := srcspan.NewFiles()
	file_id := files.Add("test", "foo\nbar\r\n\nbaz")

	for _, byte_index := range []int{0, 7, 8, 9} {
		location, _ := files.Location(file_id, byte_index)
		fmt.Printf("%d %d:%d\n", byte_index, location.LineNumber, location.ColumnNumber)
	}
	// Output:
	// 0 1:1
	// 7 2:4
	// 8 2:5
	// 9 3:1
}

func ExampleFiles_Source() {
	source := "hello world!"

	files := srcspan.NewFiles()
	file_id := files.Add("test", source)

	fmt.Println(files.Source(file_id))
	// Output: hello world! <nil>
}

func ExampleFiles_SourceSpan() {
	source := "hello world!"

	files := srcspan.NewFiles()
	file_id := files.Add("test", source)

	span, _ := files.SourceSpan(file_id)
	fmt.Println(span == srcspan.SpanFromStr(source))
	// Output: true
}

func ExampleFiles_SourceSlice() {
	files := srcspan.NewFiles()
	file_id := files.Add("test", "hello world!")

	fmt.Println(files.SourceSlice(file_id, srcspan.NewSpan(0, 5)))

	_, err := files.SourceSlice(file_id, srcspan.NewSpan(0, 100))
	fmt.Println(err != nil)
	// Output:
	// hello <nil>
	// true
}
