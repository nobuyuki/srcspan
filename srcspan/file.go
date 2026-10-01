package srcspan

import (
	"cmp"
	"fmt"
	"slices"
	"unicode/utf8"

	"srcspan/srcspan/files"
)

// A database of source files.
//
// The file id is an index into this database. (`FileId`, a `NonZeroU32`
// newtype, in Rust.)
//
// (`codespan::Files` in Rust)
type Files struct {
	files []file
}

// Create a new, empty database of files.
func NewFiles() Files {
	return Files{
		files: []file{},
	}
}

// Add a file to the database, returning the handle that can be used to
// refer to it again.
func (fs *Files) Add(name string, source string) int {
	file_id := len(fs.files)
	fs.files = append(fs.files, newFile(name, source))
	return file_id
}

// Update a source file in place.
//
// This will mean that any outstanding byte indexes will now point to
// invalid locations.
func (fs *Files) Update(file_id int, source string) error {
	f, err := fs.get(file_id)
	if err != nil {
		return err
	}
	f.update(source)
	return nil
}

// Get the file corresponding to the given id.
//
// NOTE: codespan 0.13.1 panics on an unknown id here (with a FIXME
// "return an option or result?"), so return `files.FileMissingError`
// instead.
func (fs *Files) get(file_id int) (*file, error) {
	if file_id < 0 || file_id >= len(fs.files) {
		return nil, files.FileMissingError(fmt.Sprint(file_id))
	}
	return &fs.files[file_id], nil
}

// Return the span of a given line.
func (fs *Files) LineSpan(file_id int, line_index int) (Span, error) {
	f, err := fs.get(file_id)
	if err != nil {
		return Span{}, err
	}
	return f.lineSpan(line_index)
}

// Return the span of the source of the given file.
func (fs *Files) SourceSpan(file_id int) (Span, error) {
	f, err := fs.get(file_id)
	if err != nil {
		return Span{}, err
	}
	return f.sourceSpan(), nil
}

// Return a slice of the source file, given a span.
func (fs *Files) SourceSlice(file_id int, span Span) (string, error) {
	f, err := fs.get(file_id)
	if err != nil {
		return "", err
	}
	return f.sourceSlice(span)
}

// The methods below implement `files.Files`.
// (`impl codespan_reporting::files::Files for Files` in Rust)
//
// NOTE: In Rust, `name`, `source`, `line_index` and `location` also exist
// as inherent methods with different signatures. Go can not have both,
// so only the `files.Files` ones are provided. The 0-based `Location`
// of a byte index is not exposed.

func (fs *Files) Name(id int) (string, error) {
	f, err := fs.get(id)
	if err != nil {
		return "", err
	}
	return f.name, nil
}

func (fs *Files) Source(id int) (string, error) {
	f, err := fs.get(id)
	if err != nil {
		return "", err
	}
	return f.source, nil
}

func (fs *Files) LineIndex(id int, byte_index int) (int, error) {
	f, err := fs.get(id)
	if err != nil {
		return 0, err
	}
	return f.lineIndex(byte_index), nil
}

func (fs *Files) LineNumber(id int, line_index int) (int, error) {
	return files.DefaultLineNumber(fs, id, line_index)
}

func (fs *Files) ColumnNumber(id int, line_index int, byte_index int) (int, error) {
	return files.DefaultColumnNumber(fs, id, line_index, byte_index)
}

func (fs *Files) Location(id int, byte_index int) (files.Location, error) {
	return files.DefaultLocation(fs, id, byte_index)
}

func (fs *Files) LineRange(id int, line_index int) (files.Range, error) {
	span, err := fs.LineSpan(id, line_index)
	if err != nil {
		return files.Range{}, err
	}
	return span.Range(), nil
}

var _ files.Files = (*Files)(nil)

// A file that is stored in the database.
// (`codespan::File`, private in Rust)
type file struct {
	// The name of the file.
	name string
	// The source code of the file.
	source string
	// The starting byte indices in the source code.
	line_starts []int
}

func newFile(name string, source string) file {
	return file{
		name:        name,
		source:      source,
		line_starts: files.LineStarts(source),
	}
}

func (f *file) update(source string) {
	f.source = source
	f.line_starts = files.LineStarts(source)
}

// Return the starting byte index of the line with the specified line index.
// Convenience method that already generates errors if necessary.
func (f *file) lineStart(line_index int) (int, error) {
	if line_index < 0 {
		return 0, files.NegativeLineError{
			Given: line_index,
		}
	}

	v := cmp.Compare(line_index, f.lastLineIndex())
	switch v {
	case -1:
		return f.line_starts[line_index], nil
	case 0:
		return f.sourceSpan().End(), nil
	case 1:
		return 0, files.LineTooLargeError{
			Given: line_index,
			Max:   f.lastLineIndex(),
		}
	default:
		panic(fmt.Sprintf("cmp.Compare() returns %d", v))
	}
}

func (f *file) lastLineIndex() int {
	return len(f.line_starts)
}

func (f *file) lineSpan(line_index int) (Span, error) {
	line_start, err := f.lineStart(line_index)
	if err != nil {
		return Span{}, err
	}
	next_line_start, err := f.lineStart(line_index + 1)
	if err != nil {
		return Span{}, err
	}

	return NewSpan(line_start, next_line_start), nil
}

func (f *file) lineIndex(byte_index int) int {
	i, found := slices.BinarySearch(f.line_starts, byte_index)
	if found {
		// Found the start of a line
		return i
	}
	return i - 1
}

func (f *file) location(byte_index int) (Location, error) {
	if byte_index < 0 {
		return Location{}, files.NegativeIndexError{Given: byte_index}
	}

	line_index := f.lineIndex(byte_index)
	line_start_index, err := f.lineStart(line_index)
	if err != nil {
		return Location{}, files.IndexTooLargeError{
			Given: byte_index,
			Max:   len(f.source) - 1,
		}
	}

	// `byte_index == len(f.source)` (the end of the file) is valid.
	if byte_index > len(f.source) {
		return Location{}, files.IndexTooLargeError{
			Given: byte_index,
			Max:   len(f.source) - 1,
		}
	}

	line_src := f.source[line_start_index:byte_index]

	if !utf8.ValidString(line_src) {
		return Location{}, files.InvalidCharBoundaryError{
			Given: byte_index,
		}
	}

	// The column is counted in characters, not bytes.
	// (`line_src.chars().count()` in Rust)
	return Location{
		Line:   line_index,
		Column: utf8.RuneCountInString(line_src),
	}, nil
}

func (f *file) sourceSpan() Span {
	return SpanFromStr(f.source)
}

func (f *file) sourceSlice(span Span) (string, error) {
	s := span.Start()
	e := span.End()

	source_len := len(f.source)

	// `s == source_len` is valid for an empty span at the end of the file.
	if s > source_len {
		return "", files.IndexTooLargeError{
			Given: s,
			Max:   source_len,
		}
	}
	if s < 0 {
		return "", files.NegativeIndexError{
			Given: s,
		}
	}

	if e > source_len {
		return "", files.IndexTooLargeError{
			Given: e,
			Max:   source_len,
		}
	}
	if e < 0 {
		return "", files.NegativeIndexError{
			Given: e,
		}
	}

	// A reversed span or an index inside a UTF-8 sequence can not be
	// sliced. Like `source.get(start..end)` failing in Rust, report it as
	// `IndexTooLarge` instead of panicking.
	if s > e || !files.IsRuneBoundary(f.source, s) || !files.IsRuneBoundary(f.source, e) {
		max_index := source_len - 1
		given := e
		if s > max_index {
			given = s
		}
		return "", files.IndexTooLargeError{
			Given: given,
			Max:   max_index,
		}
	}

	return f.source[s:e], nil
}
