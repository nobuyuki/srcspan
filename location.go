package srcspan

// A location in a source file.
type Location struct {
	// The line index in the source file.
	Line int
	// The column index in the source file.
	Column int
}

// Construct a new location from a line index and a column index.
func NewLocation(line, column int) Location {
	return Location{
		Line:   line,
		Column: column,
	}
}
