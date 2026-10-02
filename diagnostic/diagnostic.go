// Package diagnostic provides diagnostic data structures.
//
// (`codespan_reporting::diagnostic` in Rust)
package diagnostic

import (
	"iter"
	"slices"

	"github.com/nobuyuki/srcspan/files"
)

// A severity level for diagnostic messages.
type Severity int

const (
	Help    Severity = 0
	Note    Severity = 1
	Warning Severity = 2
	Error   Severity = 3
	Bug     Severity = 4
)

type LabelStyle int

const (
	Primary   LabelStyle = 0
	Secondary LabelStyle = 1
)

// A label describing an underlined region of code associated with a
// diagnostic.
type Label struct {
	// The style of the label.
	Style LabelStyle `json:"style"`

	// The file that we are labeling.
	FileId int `json:"file_id"`

	// The range in bytes we are going to include in the final
	// snippet.
	Range files.Range `json:"range"`

	// An optional message to provide some additional information
	// for the underlined code. These should not include line
	// breaks.
	Message string `json:"message"`
}

func NewLabel(style LabelStyle, file_id int, range_ files.Range) *Label {
	return &Label{
		Style:   style,
		FileId:  file_id,
		Range:   range_,
		Message: "",
	}
}

// Create a new label with a style of [`LabelStyle::Primary`].
//
// [`LabelStyle::Primary`]: LabelStyle::Primary
func NewPrimary(file_id int, range_ files.Range) *Label {
	return NewLabel(Primary, file_id, range_)
}

// Create a new label with a style of [`LabelStyle::Secondary`].
//
// [`LabelStyle::Secondary`]: LabelStyle::Secondary
func NewSecondary(file_id int, range_ files.Range) *Label {
	return NewLabel(Secondary, file_id, range_)
}

// Add a message to the diagnostic.
func (l *Label) WithMessage(message string) *Label {
	l.Message = message
	return l
}

// Represents a diagnostic message that can provide information like
// errors and warnings to the user.
//
// The position of a Diagnostic is considered to be the position of
// the [`Label`] that has the earliest starting position and has the
// highest style which appears in all the labels of the diagnostic.

type Diagnostic struct {
	// The overall severity of the diagnostic
	Severity Severity `json:"severity"`

	// An optional code that identifies this diagnostic.
	Code string `json:"code"` // e.g. "E0308"

	// The main message associated with this diagnostic.
	//
	// These should not include line breaks, and in order support
	// the 'short' diagnostic display mod, the message should be
	// specific enough to make sense on its own, without
	// additional context provided by labels and notes.
	Message string `json:"message"`

	// Source labels that describe the cause of the diagnostic.
	// The order of the labels inside the vector does not have
	// any meaning.  The labels are always arranged in the order
	// they appear in the source code.
	Labels []Label `json:"labels"`

	// Notes that are associated with the primary cause of the
	// diagnostic.  These can include line breaks for improved
	// formatting.
	Notes []string `json:"notes"`
}

// Create a new diagnostic.
func NewDiagnostic(severity Severity) *Diagnostic {
	return &Diagnostic{
		Severity: severity,
		Code:     "",
		Message:  "",
		Labels:   []Label{},
		Notes:    []string{},
	}
}

// Create a new diagnostic with a severity of [`Severity::Bug`].
//
// [`Severity::Bug`]: Severity::Bug
func NewBug() *Diagnostic {
	return NewDiagnostic(Bug)
}

// Create a new diagnostic with a severity of [`Severity::Error`].
//
// [`Severity::Error`]: Severity::Error
func NewError() *Diagnostic {
	return NewDiagnostic(Error)
}

// Create a new diagnostic with a severity of [`Severity::Warning`].
//
// [`Severity::Warning`]: Severity::Warning
func NewWarning() *Diagnostic {
	return NewDiagnostic(Warning)
}

// Create a new diagnostic with a severity of [`Severity::Note`].
//
// [`Severity::Note`]: Severity::Note
func NewNote() *Diagnostic {
	return NewDiagnostic(Note)
}

// Create a new diagnostic with a severity of [`Severity::Help`].
//
// [`Severity::Help`]: Severity::Help
func NewHelp() *Diagnostic {
	return NewDiagnostic(Help)
}

// Set the error code of the diagnostic.
func (d *Diagnostic) WithCode(code string) *Diagnostic {
	d.Code = code
	return d
}

// Set the message of the diagnostic.
func (d *Diagnostic) WithMessage(message string) *Diagnostic {
	d.Message = message
	return d
}

// Add a label to the diagnostic.
func (d *Diagnostic) WithLabel(label Label) *Diagnostic {
	d.Labels = append(d.Labels, label)
	return d
}

// Add some labels to the diagnostic.
func (d *Diagnostic) WithLabels(labels []Label) *Diagnostic {
	d.Labels = append(d.Labels, labels...)
	return d
}

// Add some labels to the diagnostic.
// (`with_labels_iter` in Rust, which takes an `IntoIterator`)
func (d *Diagnostic) WithLabelsIter(labels iter.Seq[Label]) *Diagnostic {
	d.Labels = slices.AppendSeq(d.Labels, labels)
	return d
}

// Add a note to the diagnostic.
func (d *Diagnostic) WithNote(note string) *Diagnostic {
	d.Notes = append(d.Notes, note)
	return d
}

// Add some notes to the diagnostic.
func (d *Diagnostic) WithNotes(notes []string) *Diagnostic {
	d.Notes = append(d.Notes, notes...)
	return d
}

// Add some notes to the diagnostic.
// (`with_notes_iter` in Rust, which takes an `IntoIterator`)
func (d *Diagnostic) WithNotesIter(notes iter.Seq[string]) *Diagnostic {
	d.Notes = slices.AppendSeq(d.Notes, notes)
	return d
}
