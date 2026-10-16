# Porting from Rust (codespan / codespan-reporting) to Go (srcspan)

`srcspan` is a port of [codespan](https://github.com/brendanzab/codespan)
**0.13.1** (`codespan`, `codespan-reporting` and `codespan-lsp`). This
document maps the Rust API to the Go API, so that code using codespan can
be ported to srcspan.

- [General conventions](#general-conventions)
- [Package layout](#package-layout)
- [An example side by side](#an-example-side-by-side)
- [Serialization](#serialization)
- [`codespan_reporting::diagnostic`](#codespan_reportingdiagnostic--githubcomnobuyukisrcspandiagnostic)
- [`codespan_reporting::files`](#codespan_reportingfiles--githubcomnobuyukisrcspanfiles)
- [`codespan_reporting::term`](#codespan_reportingterm--githubcomnobuyukisrcspanterm)
- [`codespan`](#codespan--githubcomnobuyukisrcspan)
- [`codespan_lsp`](#codespan_lsp--githubcomnobuyukisrcspanlsp)


## General conventions

| Rust | Go |
|---|---|
| `snake_case` functions and fields | `CamelCase` methods and fields (`line_index` → `LineIndex`) |
| `Type::new(...)` | `NewType(...)` |
| Other associated constructors (`Diagnostic::error()`, `Label::primary(...)`) | `NewXxx(...)` functions (`diagnostic.NewError()`, `diagnostic.NewPrimary(...)`) |
| Builder methods `with_xxx(self, ...)` | `WithXxx(...)` methods on a pointer, returning the pointer |
| `Result<T, Error>` | `(T, error)` |
| `Option<String>` | `string`, where `""` means `None` |
| `impl Display` / `impl ToString` / `impl Into<String>` arguments | `string` |
| A generic `FileId` (associated type `Files::FileId`) | `int` |
| `Range<usize>` (`a..b`) | `files.Range` (`files.NewRange(a, b)`) |
| `impl Into<Range<usize>>` arguments (accepting a `Span`) | `files.Range`; convert a `Span` with `span.Range()` |
| `Vec<T>` | `[]T` |
| `usize` indices (`ByteIndex`, `LineIndex`, ...) | `int` |
| Traits | Interfaces |
| Default methods of a trait | `DefaultXxx` functions that implementations delegate to |
| `enum` with data | An interface implemented by one type per variant (see the examples) |
| The `serialization` feature (serde) | `encoding/json`, always available; the JSON is the same as serde_json's (see [Serialization](#serialization)) |

### Why `FileId` is fixed to `int`

In Rust, `Files::FileId` is an associated type (`Copy + PartialEq`), so a
`&str` or a user-defined type can be used as a file id. A generic port would
have to thread a type parameter through `Diagnostic`, `Label`, `Files`, and
every `term.Emit*` function — the same scope as in Rust — but Go makes this
noticeably heavier:

- Go has no associated types, so the id type must be an explicit type
  parameter (`Files[ID comparable]`) on every signature.
- Go cannot infer type arguments from return types, so constructors such as
  `diagnostic.NewError[int]()` would always need them spelled out.

Using `any` as the id type would avoid generics but lose type safety
(mismatches and non-comparable ids fail at run time). Fixing the id to `int`
keeps the API simple; map other kinds of ids to `int` on your side, as
`SimpleFiles` does.


## Package layout

The packages correspond to the Rust crates and modules, and depend on each
other in the same direction (`codespan` depends on `codespan-reporting`,
not the other way around).

| Rust | Go import path |
|---|---|
| `codespan_reporting::diagnostic` | `github.com/nobuyuki/srcspan/diagnostic` |
| `codespan_reporting::files` | `github.com/nobuyuki/srcspan/files` |
| `codespan_reporting::term` | `github.com/nobuyuki/srcspan/term` |
| `codespan` | `github.com/nobuyuki/srcspan` |
| `codespan_lsp` | `github.com/nobuyuki/srcspan/lsp` |
| `codespan_reporting::term::termcolor` (re-export of `termcolor`) | [`github.com/fatih/color`](https://github.com/fatih/color) (used by `term.Styles`) |


## An example side by side

Rust:

```rust
use codespan_reporting::diagnostic::{Diagnostic, Label};
use codespan_reporting::files::SimpleFiles;
use codespan_reporting::term::termcolor::{ColorChoice, StandardStream};
use codespan_reporting::term::{self, Config};

let mut files = SimpleFiles::new();
let file_id = files.add("test.fun", "let x = 1");

let diagnostic = Diagnostic::error()
    .with_message("expected `;`")
    .with_code("E0001")
    .with_labels(vec![
        Label::primary(file_id, 9..9).with_message("here"),
    ])
    .with_notes(vec!["add a semicolon".to_owned()]);

let writer = StandardStream::stderr(ColorChoice::Always);
let config = Config::default();
term::emit_to_write_style(&mut writer.lock(), &config, &files, &diagnostic)?;
```

Go:

```go
import (
	"os"

	"github.com/nobuyuki/srcspan/diagnostic"
	"github.com/nobuyuki/srcspan/files"
	"github.com/nobuyuki/srcspan/term"
)

fs := files.NewSimpleFiles()
file_id := fs.Add("test.fun", "let x = 1")

d := diagnostic.NewError().
	WithMessage("expected `;`").
	WithCode("E0001").
	WithLabels([]diagnostic.Label{
		*diagnostic.NewPrimary(file_id, files.NewRange(9, 9)).WithMessage("here"),
	}).
	WithNotes([]string{"add a semicolon"})

styles := term.DefaultColor()
styles.ColorOutput = term.ColorOutputForce
writer := term.NewStylesWriter(os.Stderr, styles)
config := term.DefaultConfig()
err := term.EmitToWriteStyle(writer, &config, &fs, d)
```

More examples are ported in [`examples/codespan`](../examples/codespan).
They produce the same output as their Rust versions, except
`peg_calculator`, whose parser and grammar differ.


## Serialization

The `serialization` feature of codespan and codespan-reporting (serde) is
always available in Go with `encoding/json`, and produces the same JSON as
serde_json in Rust:

```json
{"severity":"Error","code":"E0001","message":"expected `;`",
 "labels":[{"style":"Primary","file_id":0,"range":{"start":9,"end":9},"message":"here"}],
 "notes":["a note"]}
```

| Rust | Go | JSON |
|---|---|---|
| `Severity`, `LabelStyle` | `diagnostic.Severity`, `diagnostic.LabelStyle` | The variant name, e.g. `"Error"`, `"Primary"` |
| `Label<FileId>` | `diagnostic.Label` | `{"style", "file_id", "range", "message"}` |
| `Diagnostic<FileId>` | `diagnostic.Diagnostic` | `{"severity", "code", "message", "labels", "notes"}`; no code is `null` |
| `Range<usize>` | `files.Range` | `{"start", "end"}` |
| `Span` | `srcspan.Span` | `{"start", "end"}` |
| `Location` | `srcspan.Location` | `{"line", "column"}` |

`Some("")` as a code in Rust is decoded as an empty `Code`, which means no
code in Go (and is rendered the same way).

## `codespan_reporting::diagnostic` → `github.com/nobuyuki/srcspan/diagnostic`

| Rust | Go | Notes |
|---|---|---|
| `Severity::{Bug, Error, Warning, Note, Help}` | `diagnostic.{Bug, Error, Warning, Note, Help}` (`Severity`) | Ordered in the same way (`Bug > Error > ... > Help`). |
| `LabelStyle::{Primary, Secondary}` | `diagnostic.{Primary, Secondary}` (`LabelStyle`) | |
| `Label<FileId>` | `diagnostic.Label` | |
| `label.style` / `.file_id` / `.range` / `.message` | `label.Style` / `.FileId` / `.Range` / `.Message` | `Range` is a `files.Range`. |
| `Label::new(style, file_id, range)` | `diagnostic.NewLabel(style, file_id, range)` | Returns `*Label`. |
| `Label::primary(file_id, range)` | `diagnostic.NewPrimary(file_id, range)` | |
| `Label::secondary(file_id, range)` | `diagnostic.NewSecondary(file_id, range)` | |
| `label.with_message(message)` | `label.WithMessage(message)` | |
| `Diagnostic<FileId>` | `diagnostic.Diagnostic` | |
| `diagnostic.severity` / `.code` / `.message` / `.labels` / `.notes` | `d.Severity` / `.Code` / `.Message` / `.Labels` / `.Notes` | `Code` is a `string`; `""` means no code. |
| `Diagnostic::new(severity)` | `diagnostic.NewDiagnostic(severity)` | Returns `*Diagnostic`. |
| `Diagnostic::bug()` / `error()` / `warning()` / `note()` / `help()` | `diagnostic.NewBug()` / `NewError()` / `NewWarning()` / `NewNote()` / `NewHelp()` | |
| `.with_code(code)` / `.with_message(message)` | `.WithCode(code)` / `.WithMessage(message)` | |
| `.with_label(label)` / `.with_labels(labels)` | `.WithLabel(label)` / `.WithLabels(labels)` | Take `Label` values (dereference the result of `NewPrimary` etc.). |
| `.with_note(note)` / `.with_notes(notes)` | `.WithNote(note)` / `.WithNotes(notes)` | |
| `.with_labels_iter(iter)` / `.with_notes_iter(iter)` | `.WithLabelsIter(seq)` / `.WithNotesIter(seq)` | Take an `iter.Seq` (`IntoIterator` in Rust); pass a slice with `slices.Values(xs)`. |


## `codespan_reporting::files` → `github.com/nobuyuki/srcspan/files`

| Rust | Go | Notes |
|---|---|---|
| `trait Files<'a>` | `files.Files` (interface) | The id type is fixed to `int`, and `Name` / `Source` return `string`. |
| `files.name(id)` | `Name(id int) (string, error)` | |
| `files.source(id)` | `Source(id int) (string, error)` | |
| `files.line_index(id, byte_index)` | `LineIndex(id, byte_index int) (int, error)` | |
| `files.line_range(id, line_index)` | `LineRange(id, line_index int) (files.Range, error)` | |
| `files.line_number(id, line_index)` (default method) | `LineNumber(id, line_index int) (int, error)` | Delegate to `files.DefaultLineNumber` for the default behavior. |
| `files.column_number(id, line_index, byte_index)` (default method) | `ColumnNumber(id, line_index, byte_index int) (int, error)` | Delegate to `files.DefaultColumnNumber`. |
| `files.location(id, byte_index)` (default method) | `Location(id, byte_index int) (files.Location, error)` | Delegate to `files.DefaultLocation`. |
| `Location { line_number, column_number }` | `files.Location{LineNumber, ColumnNumber}` | 1-based. |
| `column_index(source, line_range, byte_index)` | `files.ColumnIndex(source, line_range, byte_index)` | |
| `line_starts(source)` (iterator) | `files.LineStarts(source)` | Returns `[]int`. |
| `SimpleFile<Name, Source>` | `files.SimpleFile` | Implements `files.Files`, ignoring the id (`()` in Rust). |
| `SimpleFile::new(name, source)` | `files.NewSimpleFile(name, source)` | |
| `file.name()` / `file.source()` | `file.Name(0)` / `file.Source(0)` | The `files.Files` methods; the id is ignored. |
| `SimpleFiles<Name, Source>` | `files.SimpleFiles` | |
| `SimpleFiles::new()` | `files.NewSimpleFiles()` | |
| `files.add(name, source)` | `fs.Add(name, source)` | Returns the file id (`int`). |
| `files.get(file_id)` | `fs.Get(file_id)` | Returns `(*SimpleFile, error)`. |
| `Error::FileMissing` | `files.FileMissingError` | Holds the file id as a string. |
| `Error::IndexTooLarge { given, max }` | `files.IndexTooLargeError{Given, Max}` | |
| `Error::LineTooLarge { given, max }` | `files.LineTooLargeError{Given, Max}` | |
| `Error::ColumnTooLarge { given, max }` | `files.ColumnTooLargeError{Given, Max}` | |
| `Error::InvalidCharBoundary { given }` | `files.InvalidCharBoundaryError{Given}` | |
| `Error::Io(io::Error)` | The `io` error itself | Write errors are returned as they are. |
| `Error::FormatError` | — | |
| — | `files.NegativeIndexError{Given}` / `files.NegativeLineError{Given}` | Go only, since indices are `int`. |
| `Range<usize>` (`a..b`) | `files.Range{Start, End}` / `files.NewRange(a, b)` | Half-open. |
| `str::is_char_boundary(i)` | `files.IsRuneBoundary(s, i)` | |


## `codespan_reporting::term` → `github.com/nobuyuki/srcspan/term`

### Emitting

| Rust | Go | Notes |
|---|---|---|
| `emit_into_string(config, files, diagnostic)` | `term.EmitIntoString(config, files, diagnostic)` | Returns `(string, error)`. |
| `emit_to_string(&mut String, config, files, diagnostic)` | `term.EmitToString(io.Writer, config, files, diagnostic)` | Plain text, to any `io.Writer`. |
| `emit_to_io_write(&mut io::Write, config, files, diagnostic)` | `term.EmitToIoWrite(io.Writer, config, files, diagnostic)` | Plain text. |
| `emit_to_write_style(&mut WriteStyle, config, files, diagnostic)` | `term.EmitToWriteStyle(WriteStyle, config, files, diagnostic)` | |
| `emit(...)` (deprecated) | `term.Emit(io.Writer, ...)` | Plain text. |
| `files: &impl Files` | `files files.Files` | Pass a pointer to the implementation, e.g. `&fs`. |
| `diagnostic: &Diagnostic<FileId>` | `*diagnostic.Diagnostic` | |

### Config

| Rust | Go | Notes |
|---|---|---|
| `Config` / `Config::default()` | `term.Config` / `term.DefaultConfig()` | |
| `display_style` / `tab_width` / `chars` | `DisplayStyle` / `TabWidth` / `Chars` | |
| `start_context_lines` / `end_context_lines` | `StartContextLines` / `EndContextLines` | |
| `before_label_lines` / `after_label_lines` | `BeforeLabelLines` / `AfterLabelLines` | |
| `DisplayStyle::{Rich, Medium, Short}` | `term.{Rich, Medium, Short}` | |
| `Chars` / `Chars::box_drawing()` / `Chars::ascii()` | `term.Chars` / `term.BoxDrawing()` / `term.Ascii()` | `Chars::default()` is `BoxDrawing()`. The fields have the same names in `CamelCase` (`multi_top_left` → `MultiTopLeft`). |

### Styling

| Rust | Go | Notes |
|---|---|---|
| `trait WriteStyle` | `term.WriteStyle` (interface) | `io.Writer` plus `SetHeader`, `SetHeaderMessage`, `SetLineNumber`, `SetNoteBullet`, `SetSourceBorder`, `SetLabel` and `Reset`. |
| `GeneralWrite` / `GeneralWriteResult` | `io.Writer` / `error` | |
| `Styles` / `Styles::default()` | `term.Styles` / `term.DefaultColor()` | The fields are `*color.Color` (fatih/color) instead of `ColorSpec`. |
| `Styles::no_color()` | `term.NoColor()` | |
| `styles.header(severity)` / `styles.label(severity, style)` | `styles.Header(severity)` / `styles.Label(severity, style)` | |
| `styles.header_message()` / `line_number()` / `note_bullet()` / `source_border()` | `styles.HeaderMessage_()` / `LineNumber_()` / `NoteBullet_()` / `SourceBorder_()` | Suffixed with `_`, since Go does not allow a field and a method with the same name. These just return the exported fields, so reading `styles.HeaderMessage` etc. directly is equivalent (as in Rust, where the fields are also `pub`). The methods are kept for one-to-one correspondence; the fields keep the plain names so that struct literals read naturally. |
| `StylesWriter::new(writer, &styles)` | `term.NewStylesWriter(writer, styles)` | Writes escape codes to `writer`. |
| `StandardStream::stderr(ColorChoice::X)` (termcolor) | `term.NewStylesWriter(os.Stderr, styles)` with `styles.ColorOutput` set | |
| `ColorChoice::Always` / `Auto` / `Never` | `term.ColorOutputForce` / `ColorOutputAuto` / `ColorOutputNone` | `Auto` follows fatih/color (`NO_COLOR`, `TERM=dumb`, whether stdout is a terminal). |
| A `WriteColor` (e.g. a termcolor buffer) as a `WriteStyle` | Implement `term.WriteStyle` | See `SvgWriter` in [`examples/codespan/readme_preview`](../examples/codespan/readme_preview). |

### Others

| Rust | Go | Notes |
|---|---|---|
| `Renderer::new(writer, config)` | `term.NewRenderer(writer, config)` | Rarely needed. As in Rust, only the constructor, the `Render*` methods (`render_header`, `render_snippet_*`, ...) and the `WriteStyle` methods are public. |
| `RichDiagnostic::new(...)` / `ShortDiagnostic::new(...)` / `.render(files, renderer)` | `term.NewRichDiagnostic(...)` / `term.NewShortDiagnostic(...)` / `.Render(files, renderer)` | Rarely needed. |


## `codespan` → `github.com/nobuyuki/srcspan`

| Rust | Go | Notes |
|---|---|---|
| `Span` | `srcspan.Span` | |
| `Span::new(start, end)` | `srcspan.NewSpan(start, end)` | |
| `Span::initial()` | `srcspan.DefaultSpan()` | |
| `Span::from_str(s)` | `srcspan.SpanFromStr(s)` | |
| `span.start()` / `span.end()` | `span.Start()` / `span.End()` | Return `int` (`ByteIndex` in Rust). |
| `span.merge(other)` / `span.disjoint(other)` | `span.Merge(other)` / `span.Disjoint(other)` | |
| `Span::from(range)` (`From<Range<I>>`) | `srcspan.SpanFromRange(range)` | |
| `Range::from(span)` (`From<Span>`) | `span.Range()` | Use this to pass a `Span` to `diagnostic.NewPrimary` etc. |
| `format!("{}", span)` | `span.String()` | `[start, end)`. |
| `Location { line, column }` / `Location::new(line, column)` | `srcspan.Location{Line, Column}` / `srcspan.NewLocation(line, column)` | 0-based. |
| `ByteIndex`, `ByteOffset`, `LineIndex`, `LineNumber`, `LineOffset`, `ColumnIndex`, `ColumnNumber`, `ColumnOffset`, `RawIndex`, `RawOffset` | `int` | Not ported. |
| `FileId` | `int` | |
| `Files` / `Files::new()` | `srcspan.Files` / `srcspan.NewFiles()` | Implements `files.Files`. |
| `files.add(name, source)` | `fs.Add(name, source)` | Returns the file id (`int`). |
| `files.update(file_id, source)` | `fs.Update(file_id, source)` | Returns an `error` for an unknown id (Rust panics). |
| `files.name(file_id)` | `fs.Name(file_id)` | Returns `(string, error)`, as `files.Files`. |
| `files.source(file_id)` | `fs.Source(file_id)` | Returns `(string, error)`, as `files.Files`. |
| `files.line_index(file_id, byte_index)` | `fs.LineIndex(file_id, byte_index)` | Returns `(int, error)`, as `files.Files`. |
| `files.line_span(file_id, line_index)` | `fs.LineSpan(file_id, line_index)` | |
| `files.location(file_id, byte_index)` (0-based `Location`) | — | Not exposed, since `Location` is taken by `files.Files`; `fs.Location` returns the 1-based `files.Location`. |
| `files.source_span(file_id)` | `fs.SourceSpan(file_id)` | Returns `(Span, error)`. |
| `files.source_slice(file_id, span)` | `fs.SourceSlice(file_id, span)` | |


## `codespan_lsp` → `github.com/nobuyuki/srcspan/lsp`

| Rust | Go | Notes |
|---|---|---|
| `lsp_types::Position { line, character }` | `lsp.Position{Line, Character}` | Defined in this package (`uint32` fields), so that srcspan does not depend on a Go LSP library. The JSON is the same as the protocol's. |
| `lsp_types::Range { start, end }` | `lsp.Range{Start, End}` | |
| `byte_index_to_position(files, file_id, byte_index)` | `lsp.ByteIndexToPosition(files, file_id, byte_index)` | `character` is counted in UTF-16 code units, as specified by the protocol. |
| `byte_span_to_range(files, file_id, span)` | `lsp.ByteSpanToRange(files, file_id, span)` | |
| `position_to_byte_index(files, file_id, &position)` | `lsp.PositionToByteIndex(files, file_id, position)` | Returns an error for a line past the end of the file (Rust panics). |
| `range_to_byte_span(files, file_id, &range)` | `lsp.RangeToByteSpan(files, file_id, range)` | |
