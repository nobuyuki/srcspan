package srcspan

import (
	"cmp"
	"fmt"
	"slices"
	"unicode/utf8"
)

type Source = string

type File struct {
	// The name of the file.
	name string

	// The source code of the file.
	source Source

	// The starting byte indices in the source code.
	line_starts []int
}

func NewFile(name string, source Source) File {
	return File{
		name:        name,
		line_starts: lineStarts(source),
		source:      source,
	}
}

// Return the name of the file.
func (f *File) Name() string {
	return f.name
}

// Return the source of the file.
func (f *File) Source() *Source {
	return &f.source
}

func (f *File) Update(s Source) {
	f.source = s
	f.line_starts = lineStarts(s)
}

// Return the starting byte index of the line with the specified line index.
// Convenience method that already generates errors if necessary.
func (f *File) LineStart(line_index int) (int, error) {
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
		return len(*f.Source()), nil
	case 1:
		return 0, LineTooLargeError{
			Given: line_index,
			Max:   len(f.line_starts) - 1,
		}
	default:
		panic(fmt.Sprintf("cmp.Compare() returns %d", v))
	}
}

// / exported just for unit testing...
func (f *File) LineStarts() []int {
	return f.line_starts
}

func (f *File) LastLineIndex() int {
	return len(f.line_starts)
}

func (f *File) LineSpan(line_index int) (Span, error) {
	line_start, err := f.LineStart(line_index)
	if err != nil {
		return DefaultSpan(), err
	}

	next_line_start, err := f.LineStart(line_index + 1)
	if err != nil {
		return DefaultSpan(), err
	}

	return NewSpan(line_start, next_line_start), nil
}

func (f *File) LineIndex(byte_index int) int {
	i, found := slices.BinarySearch(f.line_starts, byte_index)

	if found {
		return i
	}
	return i - 1
}

func (f *File) Location(byte_index int) (Location, error) {
	if byte_index < 0 {
		return Location{}, NegativeIndexError{Given: byte_index}
	}

	line_index := f.LineIndex(byte_index)
	line_start_index, err := f.LineStart(line_index)
	if err != nil {
		return Location{}, IndexTooLargeError{
			Given: byte_index,
			Max:   len(f.source) - 1,
		}
	}

	// `byte_index == len(f.source)` (the end of the file) is valid.
	if byte_index > len(f.source) {
		return Location{}, IndexTooLargeError{
			Given: byte_index,
			Max:   len(f.source) - 1,
		}
	}

	line_src := f.source[line_start_index:byte_index]

	if !utf8.ValidString(line_src) {
		return Location{}, InvalidCharBoundaryError{
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

func (f *File) SourceSpan() Span {
	return SpanFromStr(*f.Source())
}

func (f *File) SourceSlice(span Span) (string, error) {
	s := span.Start()
	e := span.End()

	source_len := len(f.source)

	// `s == source_len` is valid for an empty span at the end of the file.
	if s > source_len {
		return "", IndexTooLargeError{
			Given: s,
			Max:   source_len,
		}
	}
	if s < 0 {
		return "", NegativeIndexError{
			Given: s,
		}
	}

	if e > source_len {
		return "", IndexTooLargeError{
			Given: e,
			Max:   source_len,
		}
	}
	if e < 0 {
		return "", NegativeIndexError{
			Given: e,
		}
	}

	// A reversed span or an index inside a UTF-8 sequence can not be
	// sliced. Like `source.get(start..end)` failing in Rust, report it as
	// `IndexTooLarge` instead of panicking.
	if s > e || !IsRuneBoundary(f.source, s) || !IsRuneBoundary(f.source, e) {
		max_index := source_len - 1
		given := e
		if s > max_index {
			given = s
		}
		return "", IndexTooLargeError{
			Given: given,
			Max:   max_index,
		}
	}

	return f.source[s:e], nil
}

func lineStarts(source Source) []int {
	starts := []int{0}

	for i := range len(source) {
		if source[i] == '\n' {
			starts = append(starts, i+1)
		}
	}

	return starts
}
