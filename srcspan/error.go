package srcspan

import (
	"fmt"
)

// A required file is not in the file database.
type FileMissingError string

func (e FileMissingError) Error() string {
	// write!(f, "file missing"),
	return fmt.Sprintf("File not found: %s", string(e))
}

// The file is present, but does not contain the specified byte index.
type IndexTooLargeError struct {
	Given int
	Max   int
}

func (e IndexTooLargeError) Error() string {
	// write!(f, "invalid index {}, maximum index is {}", given, max)
	return fmt.Sprintf("Index too large: given=%d, max=%d", e.Given, e.Max)
}

// The file is present, but does not contain the specified line index.
type LineTooLargeError struct {
	Given int
	Max   int
}

func (e LineTooLargeError) Error() string {
	// write!(f, "invalid line {}, maximum line is {}", given, max)
	return fmt.Sprintf("Line too large: given=%d, max=%d", e.Given, e.Max)
}

// The file is present and contains the specified line index, but the
// line does not contain the specified column index.
type ColumnTooLargeError struct {
	Given int
	Max   int
}

func (e ColumnTooLargeError) Error() string {
	// write!(f, "invalid column {}, maximum column {}", given, max)
	return fmt.Sprintf("Column too large: given=%d, max=%d", e.Given, e.Max)
}

// The given index is contained in the file, but is not a boundary of a UTF-8 code point.
type InvalidCharBoundaryError struct {
	Given int
}

func (e InvalidCharBoundaryError) Error() string {
	// write!(f, "index is not a code point boundary"),
	return fmt.Sprintf("Invalid char boundary: given=%d", e.Given)
}
