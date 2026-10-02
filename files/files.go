// Package files provides source file support for diagnostic reporting.
//
// The main interface defined in this package is the [Files] interface,
// which provides the minimum amount of functionality required for printing
// diagnostics with the `term.Emit*` functions.
//
// Simple implementations of this interface are provided:
//
//   - [SimpleFile]: For single-file use-cases
//   - [SimpleFiles]: For multi-file use-cases
//
// These data structures provide a pretty minimal API, however, so end-users
// are encouraged to create their own implementations for their own specific
// use-cases, such as an implementation that accesses the file system
// directly (and caches the line start locations).
//
// (`codespan_reporting::files` in Rust)
package files

import (
	"cmp"
	"fmt"
	"slices"
)

// A minimal interface for accessing source files when rendering
// diagnostics.
//
// Go interfaces can not have default methods, so the methods that have a
// default implementation in Rust (`LineNumber`, `ColumnNumber` and
// `Location`) are provided as `Default*` functions that implementations
// can delegate to.
type Files interface {
	// The user-facing name of a file.
	Name(id int) (string, error)

	// The source code of a file.
	Source(id int) (string, error)

	// The index of the line at the given byte index.
	// If the byte index is past the end of the file, returns the maximum
	// line index in the file. This means that this function only fails if
	// the file is not present.
	//
	// # Note for implementors
	//
	// This can be implemented efficiently by performing a binary search
	// over a list of line starts that was computed by calling the
	// [LineStarts] function. It might be useful to pre-compute and cache
	// these line starts.
	LineIndex(id int, byte_index int) (int, error)

	// The user-facing line number at the given line index.
	// It is not necessarily checked that the specified line index is
	// actually in the file.
	//
	// # Note for implementors
	//
	// This is usually 1-indexed from the beginning of the file, but can be
	// useful for implementing something like the C preprocessor's `#line`
	// macro. See [DefaultLineNumber] for the default implementation.
	LineNumber(id int, line_index int) (int, error)

	// The user-facing column number at the given line index and byte
	// index.
	//
	// # Note for implementors
	//
	// This is usually 1-indexed from the the start of the line.
	// See [DefaultColumnNumber] for the default implementation, based on
	// the [ColumnIndex] function.
	ColumnNumber(id int, line_index int, byte_index int) (int, error)

	// Convenience method for returning line and column number at the given
	// byte index in the file.
	// See [DefaultLocation] for the default implementation.
	Location(id int, byte_index int) (Location, error)

	// The byte range of line in the source of the file.
	LineRange(id int, line_index int) (Range, error)
}

// The default implementation of [Files.LineNumber].
func DefaultLineNumber(_ Files, _ int, line_index int) (int, error) {
	return line_index + 1, nil
}

// The default implementation of [Files.ColumnNumber].
func DefaultColumnNumber(files Files, id int, line_index int, byte_index int) (int, error) {
	source, err := files.Source(id)
	if err != nil {
		return 0, err
	}
	line_range, err := files.LineRange(id, line_index)
	if err != nil {
		return 0, err
	}
	column_index := ColumnIndex(source, line_range, byte_index)

	return column_index + 1, nil
}

// The default implementation of [Files.Location].
func DefaultLocation(files Files, id int, byte_index int) (Location, error) {
	line_index, err := files.LineIndex(id, byte_index)
	if err != nil {
		return Location{}, err
	}

	line_number, err := files.LineNumber(id, line_index)
	if err != nil {
		return Location{}, err
	}
	column_number, err := files.ColumnNumber(id, line_index, byte_index)
	if err != nil {
		return Location{}, err
	}

	return Location{
		LineNumber:   line_number,
		ColumnNumber: column_number,
	}, nil
}

// A user-facing location in a source file.
//
// Returned by [Files.Location].
type Location struct {
	// The user-facing line number.
	LineNumber int
	// The user-facing column number.
	ColumnNumber int
}

// The column index at the given byte index in the source file.
// This is the number of characters to the given byte index.
//
// If the byte index is smaller than the start of the line, then `0` is
// returned. If the byte index is past the end of the line, the column
// index of the last character `+ 1` is returned.
func ColumnIndex(source string, line_range Range, byte_index int) int {
	end_index := min(byte_index, min(line_range.End, len(source)))

	count := 0
	for byte_index := line_range.Start; byte_index < end_index; byte_index++ {
		if IsRuneBoundary(source, byte_index+1) {
			count++
		}
	}
	return count
}

// Return the starting byte index of each line in the source string.
//
// This can make it easier to implement [Files.LineIndex] by allowing
// implementors of [Files] to pre-compute the line starts, then search for
// the corresponding line range.
func LineStarts(source string) []int {
	starts := []int{0}

	for i := range len(source) {
		if source[i] == '\n' {
			starts = append(starts, i+1)
		}
	}

	return starts
}

// A file database that contains a single source file.
//
// Because there is only single file in this database, the file id is
// ignored. (`()` is used as a `FileId` in Rust.)
//
// This is useful for simple language tests, but it might be worth
// creating a custom implementation when a language scales beyond a
// certain size.
type SimpleFile struct {
	// The name of the file.
	name string
	// The source code of the file.
	source string
	// The starting byte indices in the source code.
	line_starts []int
}

// Create a new source file.
func NewSimpleFile(name string, source string) SimpleFile {
	return SimpleFile{
		name:        name,
		line_starts: LineStarts(source),
		source:      source,
	}
}

// Return the starting byte index of the line with the specified line
// index. Convenience method that already generates errors if necessary.
func (f *SimpleFile) lineStart(line_index int) (int, error) {
	if line_index < 0 {
		return 0, NegativeLineError{
			Given: line_index,
		}
	}

	v := cmp.Compare(line_index, len(f.line_starts))
	switch v {
	case -1:
		return f.line_starts[line_index], nil
	case 0:
		return len(f.source), nil
	case 1:
		return 0, LineTooLargeError{
			Given: line_index,
			Max:   len(f.line_starts) - 1,
		}
	default:
		panic(fmt.Sprintf("cmp.Compare() returns %d", v))
	}
}

func (f *SimpleFile) Name(_ int) (string, error) {
	return f.name, nil
}

func (f *SimpleFile) Source(_ int) (string, error) {
	return f.source, nil
}

func (f *SimpleFile) LineIndex(_ int, byte_index int) (int, error) {
	i, found := slices.BinarySearch(f.line_starts, byte_index)
	if found {
		return i, nil
	}
	return i - 1, nil
}

func (f *SimpleFile) LineNumber(id int, line_index int) (int, error) {
	return DefaultLineNumber(f, id, line_index)
}

func (f *SimpleFile) ColumnNumber(id int, line_index int, byte_index int) (int, error) {
	return DefaultColumnNumber(f, id, line_index, byte_index)
}

func (f *SimpleFile) Location(id int, byte_index int) (Location, error) {
	return DefaultLocation(f, id, byte_index)
}

func (f *SimpleFile) LineRange(_ int, line_index int) (Range, error) {
	line_start, err := f.lineStart(line_index)
	if err != nil {
		return Range{}, err
	}
	next_line_start, err := f.lineStart(line_index + 1)
	if err != nil {
		return Range{}, err
	}

	return NewRange(line_start, next_line_start), nil
}

// A file database that can store multiple source files.
//
// This is useful for simple language tests, but it might be worth
// creating a custom implementation when a language scales beyond a
// certain size. It is a glorified `[]SimpleFile` that implements the
// [Files] interface.
type SimpleFiles struct {
	files []SimpleFile
}

// Create a new files database.
func NewSimpleFiles() SimpleFiles {
	return SimpleFiles{
		files: []SimpleFile{},
	}
}

// Add a file to the database, returning the handle that can be used to
// refer to it again.
func (fs *SimpleFiles) Add(name string, source string) int {
	file_id := len(fs.files)
	fs.files = append(fs.files, NewSimpleFile(name, source))
	return file_id
}

// Get the file corresponding to the given id.
func (fs *SimpleFiles) Get(file_id int) (*SimpleFile, error) {
	if file_id < 0 || file_id >= len(fs.files) {
		return nil, FileMissingError(fmt.Sprint(file_id))
	}
	return &fs.files[file_id], nil
}

func (fs *SimpleFiles) Name(file_id int) (string, error) {
	f, err := fs.Get(file_id)
	if err != nil {
		return "", err
	}
	return f.name, nil
}

func (fs *SimpleFiles) Source(file_id int) (string, error) {
	f, err := fs.Get(file_id)
	if err != nil {
		return "", err
	}
	return f.source, nil
}

func (fs *SimpleFiles) LineIndex(file_id int, byte_index int) (int, error) {
	f, err := fs.Get(file_id)
	if err != nil {
		return 0, err
	}
	return f.LineIndex(file_id, byte_index)
}

func (fs *SimpleFiles) LineNumber(file_id int, line_index int) (int, error) {
	return DefaultLineNumber(fs, file_id, line_index)
}

func (fs *SimpleFiles) ColumnNumber(file_id int, line_index int, byte_index int) (int, error) {
	return DefaultColumnNumber(fs, file_id, line_index, byte_index)
}

func (fs *SimpleFiles) Location(file_id int, byte_index int) (Location, error) {
	return DefaultLocation(fs, file_id, byte_index)
}

func (fs *SimpleFiles) LineRange(file_id int, line_index int) (Range, error) {
	f, err := fs.Get(file_id)
	if err != nil {
		return Range{}, err
	}
	return f.LineRange(file_id, line_index)
}

// Check that the implementations satisfy the interface.
var (
	_ Files = (*SimpleFile)(nil)
	_ Files = (*SimpleFiles)(nil)
)
