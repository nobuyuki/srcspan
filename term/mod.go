// Package term is the terminal back-end for emitting diagnostics.
//
// This package corresponds to the `codespan_reporting::term` module in Rust.
package term

import (
	"fmt"
	"io"
	"strings"

	"github.com/nobuyuki/srcspan/diagnostic"
	"github.com/nobuyuki/srcspan/files"
)

// Emit a diagnostic as plain text, without styling, into a new string.
// (`emit_into_string` in Rust)
func EmitIntoString(config *Config, files files.Files, diagnostic *diagnostic.Diagnostic) (string, error) {
	var writer strings.Builder

	err := EmitToString(&writer, config, files, diagnostic)
	if err != nil {
		return "", err
	}

	return writer.String(), nil
}

// Emit a diagnostic as plain text, without styling, appending it to the
// given string builder. Nothing is appended if an error occurs.
// (`emit_to_string` in Rust, which takes a `&mut String`)
func EmitToString(writer *strings.Builder, config *Config, files files.Files, diagnostic *diagnostic.Diagnostic) error {
	var buffer strings.Builder
	err := emitWithStyle(
		newPlainWriter(&buffer),
		config,
		files,
		diagnostic,
	)
	if err != nil {
		return err
	}
	writer.WriteString(buffer.String())
	return nil
}

// Emit a diagnostic to the given writer as plain text, without styling.
// (`emit_to_io_write` in Rust)
func EmitToIoWrite(writer io.Writer, config *Config, files files.Files, diagnostic *diagnostic.Diagnostic) error {
	return emitWithStyle(
		newPlainWriter(writer),
		config,
		files,
		diagnostic,
	)
}

// Emit a diagnostic to the given writer, with the styles of the writer.
// (`emit_to_write_style` in Rust)
func EmitToWriteStyle(writer WriteStyle, config *Config, files files.Files, diagnostic *diagnostic.Diagnostic) error {
	return emitWithStyle(writer, config, files, diagnostic)
}

// Emit a diagnostic using the given writer, context, config, and files.
//
// The return value covers all error cases. These error case can arise if:
//   - a file was removed from the file database.
//   - a file was changed so that it is too small to have an index
//   - IO fails
//
// Deprecated: Since codespan-reporting 0.13.0. Use [EmitToWriteStyle]
// instead or depending on the writer use [EmitToIoWrite] or
// [EmitToString]. (`#[deprecated]` in Rust)
func Emit(writer io.Writer, config *Config, files files.Files, diagnostic *diagnostic.Diagnostic) error {
	return emitWithStyle(newPlainWriter(writer), config, files, diagnostic)
}

func emitWithStyle(writer WriteStyle, config *Config, files files.Files, diagnostic *diagnostic.Diagnostic) error {
	renderer := NewRenderer(writer, config)

	switch config.DisplayStyle {
	case Rich:
		return NewRichDiagnostic(diagnostic, config).Render(files, &renderer)
	case Medium:
		return NewShortDiagnostic(diagnostic, true).Render(files, &renderer)
	case Short:
		return NewShortDiagnostic(diagnostic, false).Render(files, &renderer)
	default:
		panic(fmt.Sprintf("Unknown DisplayStyle: %v", config.DisplayStyle))
	}
}
