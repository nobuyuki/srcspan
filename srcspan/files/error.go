package files

// Errors from looking up a file or a piece of content in that file.
// (`codespan_reporting::files::Error` in Rust)

import (
	"fmt"
)

// A required file is not in the file database.
type FileMissingError string

func (FileMissingError) Error() string {
	// write!(f, "file missing"),
	return "file missing"
}

// The file is present, but does not contain the specified byte index.
type IndexTooLargeError struct {
	Given int
	Max   int
}

func (e IndexTooLargeError) Error() string {
	// write!(f, "invalid index {}, maximum index is {}", given, max)
	return fmt.Sprintf("invalid index %d, maximum index is %d", e.Given, e.Max)
}

// The given byte index is negative. (Go only, since indices are `int`.)
type NegativeIndexError struct {
	Given int
}

func (e NegativeIndexError) Error() string {
	return fmt.Sprintf("invalid index %d, index must not be negative", e.Given)
}

// The file is present, but does not contain the specified line index.
type LineTooLargeError struct {
	Given int
	Max   int
}

func (e LineTooLargeError) Error() string {
	// write!(f, "invalid line {}, maximum line is {}", given, max)
	return fmt.Sprintf("invalid line %d, maximum line is %d", e.Given, e.Max)
}

// The given line index is negative. (Go only, since indices are `int`.)
type NegativeLineError struct {
	Given int
}

func (e NegativeLineError) Error() string {
	return fmt.Sprintf("invalid line %d, line must not be negative", e.Given)
}

// The file is present and contains the specified line index, but the
// line does not contain the specified column index.
type ColumnTooLargeError struct {
	Given int
	Max   int
}

func (e ColumnTooLargeError) Error() string {
	// write!(f, "invalid column {}, maximum column {}", given, max)
	return fmt.Sprintf("invalid column %d, maximum column %d", e.Given, e.Max)
}

// The given index is contained in the file, but is not a boundary of a UTF-8 code point.
type InvalidCharBoundaryError struct {
	Given int
}

func (InvalidCharBoundaryError) Error() string {
	// write!(f, "index is not a code point boundary"),
	return "index is not a code point boundary"
}
