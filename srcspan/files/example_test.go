package files_test

// Ports of the doc tests of `codespan_reporting::files` in Rust.

import (
	"fmt"
	"slices"

	"srcspan/srcspan/files"
)

func ExampleColumnIndex() {
	source := "\n\n🗻∈🌏\n\n"

	fmt.Println(files.ColumnIndex(source, files.NewRange(0, 1), 0))
	fmt.Println(files.ColumnIndex(source, files.NewRange(2, 13), 0))
	fmt.Println(files.ColumnIndex(source, files.NewRange(2, 13), 2+0))
	fmt.Println(files.ColumnIndex(source, files.NewRange(2, 13), 2+1))
	fmt.Println(files.ColumnIndex(source, files.NewRange(2, 13), 2+4))
	fmt.Println(files.ColumnIndex(source, files.NewRange(2, 13), 2+8))
	fmt.Println(files.ColumnIndex(source, files.NewRange(2, 13), 2+10))
	fmt.Println(files.ColumnIndex(source, files.NewRange(2, 13), 2+11))
	fmt.Println(files.ColumnIndex(source, files.NewRange(2, 13), 2+12))
	// Output:
	// 0
	// 0
	// 0
	// 0
	// 1
	// 2
	// 2
	// 3
	// 3
}

func ExampleLineStarts() {
	source := "foo\nbar\r\n\nbaz"
	line_starts := files.LineStarts(source)

	fmt.Println(line_starts)

	line_index := func(line_starts []int, byte_index int) int {
		line, found := slices.BinarySearch(line_starts, byte_index)
		if found {
			return line
		}
		return line - 1
	}

	fmt.Println(line_index(line_starts, 5))
	// Output:
	// [0 4 9 10]
	// 1
}
