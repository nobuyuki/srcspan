package main

// An example that shows how to implement a simple custom file database.
// The database uses 32-bit file-ids, which could be useful for optimizing
// memory usage.
//
// To run this example, execute the following command from the top level of
// this repository:
//
// ```sh
// go run ./examples/codespan/custom_files/custom_files.go
// ```

import (
	"cmp"
	"fmt"
	"math"
	"os"
	"slices"

	"github.com/nobuyuki/srcspan/diagnostic"
	"github.com/nobuyuki/srcspan/files"
	"github.com/nobuyuki/srcspan/term"
)

func main() {
	files := NewFiles()

	file_id0, ok := files.Add("0.greeting", "hello world!")
	if !ok {
		os.Exit(1)
	}
	file_id1, ok := files.Add("1.greeting", "bye world")
	if !ok {
		os.Exit(1)
	}

	messages := []Message{
		UnwantedGreetings{
			Greetings: []FileRange{
				{FileId: file_id0, Range: newRange(0, 5)},
				{FileId: file_id1, Range: newRange(0, 3)},
			},
		},
		OverTheTopExclamations{
			Exclamations: []FileRange{
				{FileId: file_id0, Range: newRange(11, 12)},
			},
		},
	}

	styles := term.DefaultColor()
	styles.ColorOutput = term.ColorOutputForce
	writer := term.NewStylesWriter(os.Stderr, styles)
	config := term.DefaultConfig()
	for _, message := range messages {
		err := term.EmitToWriteStyle(writer, &config, &files, message.ToDiagnostic())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

// `files` is shadowed by the variable of the same name (as in Rust) in
// `main`, so the package's constructor is aliased here.
var newRange = files.NewRange

// The file implementation.
// (`mod files` in Rust)

// A file that is stored in the database.
type File struct {
	// The name of the file.
	name string
	// The source code of the file.
	source string
	// The starting byte indices in the source code.
	line_starts []int
}

func (f *File) lineStart(line_index int) (int, error) {
	v := cmp.Compare(line_index, len(f.line_starts))
	switch v {
	case -1:
		return f.line_starts[line_index], nil
	case 0:
		return len(f.source), nil
	case 1:
		return 0, files.LineTooLargeError{
			Given: line_index,
			Max:   len(f.line_starts) - 1,
		}
	default:
		panic(fmt.Sprintf("cmp.Compare() returns %d", v))
	}
}

// NOTE: In Rust, the file id is an opaque `FileId(u32)` newtype. The id
// type of `files.Files` is fixed to `int` in Go, so ids are `int` here,
// but `Add` still limits them to 32 bits.

type Files struct {
	files []File
}

// Create a new files database.
func NewFiles() Files {
	return Files{files: []File{}}
}

// Add a file to the database, returning the handle that can be used to
// refer to it again, or false if there are too many files for a 32-bit
// file id.
func (fs *Files) Add(name string, source string) (int, bool) {
	if uint64(len(fs.files)) > math.MaxUint32 {
		return 0, false
	}
	file_id := len(fs.files)

	fs.files = append(fs.files, File{
		name:        name,
		line_starts: files.LineStarts(source),
		source:      source,
	})

	return file_id, true
}

// Get the file corresponding to the given id.
func (fs *Files) get(file_id int) (*File, error) {
	if file_id < 0 || file_id >= len(fs.files) {
		return nil, files.FileMissingError(fmt.Sprint(file_id))
	}
	return &fs.files[file_id], nil
}

// The methods below implement `files.Files`.

func (fs *Files) Name(file_id int) (string, error) {
	f, err := fs.get(file_id)
	if err != nil {
		return "", err
	}
	return f.name, nil
}

func (fs *Files) Source(file_id int) (string, error) {
	f, err := fs.get(file_id)
	if err != nil {
		return "", err
	}
	return f.source, nil
}

func (fs *Files) LineIndex(file_id int, byte_index int) (int, error) {
	f, err := fs.get(file_id)
	if err != nil {
		return 0, err
	}
	i, found := slices.BinarySearch(f.line_starts, byte_index)
	if found {
		return i, nil
	}
	return i - 1, nil
}

// The default implementations are used for these.
// (Default methods of the `Files` trait in Rust)

func (fs *Files) LineNumber(file_id int, line_index int) (int, error) {
	return files.DefaultLineNumber(fs, file_id, line_index)
}

func (fs *Files) ColumnNumber(file_id int, line_index int, byte_index int) (int, error) {
	return files.DefaultColumnNumber(fs, file_id, line_index, byte_index)
}

func (fs *Files) Location(file_id int, byte_index int) (files.Location, error) {
	return files.DefaultLocation(fs, file_id, byte_index)
}

func (fs *Files) LineRange(file_id int, line_index int) (files.Range, error) {
	f, err := fs.get(file_id)
	if err != nil {
		return files.Range{}, err
	}
	line_start, err := f.lineStart(line_index)
	if err != nil {
		return files.Range{}, err
	}
	next_line_start, err := f.lineStart(line_index + 1)
	if err != nil {
		return files.Range{}, err
	}

	return files.NewRange(line_start, next_line_start), nil
}

var _ files.Files = (*Files)(nil)

// A range in a file. (`(FileId, Range<usize>)` in Rust)
type FileRange struct {
	FileId int
	Range  files.Range
}

// A Diagnostic message.
// (`enum Message` in Rust; each variant is a type implementing it.)
type Message interface {
	ToDiagnostic() *diagnostic.Diagnostic
}

type UnwantedGreetings struct {
	Greetings []FileRange
}

func (m UnwantedGreetings) ToDiagnostic() *diagnostic.Diagnostic {
	labels := []diagnostic.Label{}
	for _, greeting := range m.Greetings {
		labels = append(labels,
			diagnostic.NewPrimary(greeting.FileId, greeting.Range).WithMessage("a greeting"))
	}

	return diagnostic.NewError().
		WithMessage("greetings are not allowed").
		WithLabels(labels).
		WithNotes([]string{
			"found greetings!",
			"pleas no greetings :(",
		})
}

type OverTheTopExclamations struct {
	Exclamations []FileRange
}

func (m OverTheTopExclamations) ToDiagnostic() *diagnostic.Diagnostic {
	labels := []diagnostic.Label{}
	for _, exclamation := range m.Exclamations {
		labels = append(labels,
			diagnostic.NewPrimary(exclamation.FileId, exclamation.Range).WithMessage("an exclamation"))
	}

	return diagnostic.NewError().
		WithMessage("over-the-top exclamations").
		WithLabels(labels).
		WithNotes([]string{"ridiculous!"})
}
