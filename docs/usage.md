# Using `srcspan`

This is a short guide to rendering diagnostics with `srcspan`. If you
already know Rust's `codespan-reporting`, see
[Porting from Rust](./porting-from-rust.md) instead.


## Overview

Rendering a diagnostic takes three pieces:

1. **Files** (`files.Files`): the source code that diagnostics point
   into. `files.SimpleFiles` is enough for most programs.
2. **A diagnostic** (`diagnostic.Diagnostic`): a severity, a message, and
   labels that underline byte ranges of a file.
3. **An emitter** (`term.Emit*`): renders the diagnostic, with or
   without colors.


## A first diagnostic

```go
package main

import (
	"os"
	"strings"

	"srcspan/diagnostic"
	"srcspan/files"
	"srcspan/term"
)

func main() {
	source := "let x = 1 + \"two\";\n"

	// 1. Register the source. Add returns an id to refer to the file.
	fs := files.NewSimpleFiles()
	file_id := fs.Add("main.calc", source)

	// 2. Build the diagnostic. Ranges are byte offsets into the source.
	start := strings.Index(source, `"two"`)
	d := diagnostic.NewError().
		WithMessage("mismatched types").
		WithCode("E0308").
		WithLabel(diagnostic.NewPrimary(file_id, files.NewRange(start, start+5)).
			WithMessage("expected number, found string")).
		WithLabel(diagnostic.NewSecondary(file_id, files.NewRange(8, 9)).
			WithMessage("this is a number")).
		WithNote("strings can not be added to numbers")

	// 3. Emit it to stderr, with colors.
	writer := term.NewStylesWriter(os.Stderr, term.DefaultStyles())
	config := term.DefaultConfig()
	if err := term.EmitToWriteStyle(writer, &config, &fs, d); err != nil {
		panic(err)
	}
}
```

Output:

```text
error[E0308]: mismatched types
  ┌─ main.calc:1:13
  │
1 │ let x = 1 + "two";
  │         -   ^^^^^ expected number, found string
  │         │
  │         this is a number
  │
  = strings can not be added to numbers
```


## Building diagnostics

### Severity

Create a diagnostic with one of `diagnostic.NewBug`, `NewError`,
`NewWarning`, `NewNote` or `NewHelp`, or with
`NewDiagnostic(severity)`.

### Message, code and notes

- `WithMessage` sets the main message. Keep it on one line, and make it
  understandable on its own: the short display style shows only this
  line.
- `WithCode` sets an optional code such as `E0308`, shown as
  `error[E0308]`.
- `WithNote` / `WithNotes` add notes below the snippet. Notes may
  contain line breaks.

### Labels

A label underlines a byte range of a file:

- `diagnostic.NewPrimary(file_id, r)` marks the cause of the diagnostic
  (underlined with `^`).
- `diagnostic.NewSecondary(file_id, r)` adds context (underlined with
  `-`).
- `.WithMessage(...)` attaches a message to the label.

Add labels with `WithLabel`, `WithLabels` (a slice) or `WithLabelsIter`
(an `iter.Seq`). The order does not matter; labels are always shown in
source order. A label may span several lines, and labels may point to
different files.

### Ranges

`files.Range` is a **half-open byte range** `[Start, End)` into the
source string, created with `files.NewRange(start, end)`. Offsets are
in bytes, not runes, so use the byte offsets your lexer or parser
already tracks (or `strings.Index` and friends). An empty range such as
`NewRange(10, 10)` points between two characters.

`Diagnostic` and `Label` are plain structs with exported fields, so
you can also build or modify them directly.


## Emitting

| Function | Output |
| --- | --- |
| `term.EmitToWriteStyle(w, &config, files, d)` | Styled output through a `term.WriteStyle`, e.g. `term.NewStylesWriter(os.Stderr, term.DefaultStyles())` |
| `term.EmitToIoWrite(w, &config, files, d)` | Plain text to any `io.Writer` |
| `term.EmitIntoString(&config, files, d)` | Plain text, returned as a `string` |
| `term.EmitToString(&sb, &config, files, d)` | Plain text, appended to a `*strings.Builder` |

The error is non-nil when a label refers to a file that does not exist,
or to a byte offset outside the file, or when writing fails.

### Colors

`term.DefaultStyles()` returns the default styles, and `term.NoColor()`
styles without colors. The `ColorOutput` field of `term.Styles` decides
when colors are written:

- `term.ColorOutputAuto` (the default) follows
  [fatih/color](https://github.com/fatih/color): colors are disabled when
  `NO_COLOR` is set, when `TERM=dumb`, or when stdout is not a terminal.
- `term.ColorOutputForce` always writes colors.
- `term.ColorOutputNone` never writes colors.

The other fields of `term.Styles` (`HeaderError`, `PrimaryLabelError`,
...) are `*color.Color` values that you can replace.

### Configuration

`term.DefaultConfig()` returns the default `term.Config`. Commonly
changed fields:

- `DisplayStyle`: `term.Rich` (the default, as above), `term.Short`
  (one line: `main.calc:1:13: error[E0308]: mismatched types`) or
  `term.Medium` (the short line followed by the notes).
- `Chars`: `term.BoxDrawing()` (the default) or `term.Ascii()` for
  terminals without box-drawing characters.
- `TabWidth`: the column width of tabs (default `4`).
- `StartContextLines`, `EndContextLines`, `BeforeLabelLines`,
  `AfterLabelLines`: how many lines of context are shown around labels.

For example, ASCII output:

```go
config := term.DefaultConfig()
config.Chars = term.Ascii()
term.EmitToIoWrite(os.Stdout, &config, &fs, d)
```

```text
error[E0308]: mismatched types
  --> main.calc:1:13
  |
1 | let x = 1 + "two";
  |         -   ^^^^^ expected number, found string
  |         |
  |         this is a number
  |
  = strings can not be added to numbers
```


## Files

### `files.SimpleFiles` and `files.SimpleFile`

`files.SimpleFiles` stores any number of files; `Add(name, source)`
returns the file id. For a program with a single file,
`files.NewSimpleFile(name, source)` also implements `files.Files`, and
ignores the file id.

### `srcspan.Files`

The root package `srcspan` provides `srcspan.Files`, a file database
that also implements `files.Files` and adds:

- `Update(file_id, source)` to replace the source of a file, e.g. in an
  editor or language server.
- `SourceSlice(file_id, span)`, `LineSpan(file_id, line_index)` and
  `SourceSpan(file_id)`, which work with `srcspan.Span`.

`srcspan.Span` is a byte range like `files.Range`, with helpers such as
`Merge` and `Disjoint`. Convert between the two with `span.Range()` and
`srcspan.SpanFromRange(r)`.

### Your own `files.Files`

If your program already keeps its sources somewhere, implement
`files.Files` instead of copying them. `Name`, `Source`, `LineIndex`
and `LineRange` are specific to your storage; for `LineNumber`,
`ColumnNumber` and `Location`, delegate to `files.DefaultLineNumber`,
`files.DefaultColumnNumber` and `files.DefaultLocation`.
`files.LineStarts` computes the line start offsets that `LineIndex` and
`LineRange` need. See
[examples/codespan/custom_files](../examples/codespan/custom_files)
for a complete implementation.


## Other features

- **JSON**: `Diagnostic`, `Label`, `files.Range` and `srcspan.Span`
  marshal to and from JSON with `encoding/json`, in the same format as
  `serde_json` in Rust.
- **Language Server Protocol**: the `srcspan/lsp` package converts
  byte offsets and ranges to LSP positions and ranges, and back
  (`ByteIndexToPosition`, `ByteSpanToRange`, `PositionToByteIndex`,
  `RangeToByteSpan`).


## Tips

- Keep your own error type for your program's errors, and convert it
  to a `diagnostic.Diagnostic` only when reporting. See
  [examples/codespan/reusable_diagnostic](../examples/codespan/reusable_diagnostic).
- To test the output of your diagnostics, render them with
  `term.EmitIntoString` and compare the strings; the output contains no
  escape sequences.
- More examples are in the [examples](../examples) directory.
