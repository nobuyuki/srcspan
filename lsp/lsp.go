// Package lsp provides utilities for translating from srcspan types into
// Language Server Protocol (LSP) types.
//
// (`codespan_lsp` in Rust)
package lsp

import (
	"unicode/utf16"

	"srcspan/files"
)

// A position in a text document: a zero-based line and a zero-based
// character offset counted in UTF-16 code units, as specified by the
// protocol.
//
// (`lsp_types::Position` in Rust. It is defined here, instead of depending
// on a Go LSP library, so that srcspan has no extra dependencies. Its JSON
// is the same as the protocol's.)
type Position struct {
	Line      uint32 `json:"line"`
	Character uint32 `json:"character"`
}

// A range in a text document, from `Start` to `End` (exclusive).
//
// (`lsp_types::Range` in Rust)
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// The functions below take a parameter named `files` (as in Rust), which
// shadows the `files` package in their bodies, so the package's names used
// there are aliased here.
type fileRange = files.Range

var newRange = files.NewRange

func locationToPosition(
	line_str string,
	line int,
	column int,
	byte_index int,
) (Position, error) {
	if column > len(line_str) {
		max_column := len(line_str)
		given := column

		return Position{}, files.ColumnTooLargeError{Given: given, Max: max_column}
	} else if !files.IsRuneBoundary(line_str, column) {
		given := byte_index

		return Position{}, files.InvalidCharBoundaryError{Given: given}
	}

	line_utf16 := utf16.Encode([]rune(line_str[:column]))
	character := uint32(len(line_utf16)) //nolint:gosec // the length of a line fits in uint32, as in Rust

	return Position{Line: uint32(line), Character: character}, nil //nolint:gosec // a line index fits in uint32, as in Rust
}

// Returns the source of the line at `line_span`, like `source.get(line_span)`
// in Rust.
func lineStr(source string, line_span fileRange) (string, error) {
	if line_span.Start < 0 || line_span.Start > line_span.End || line_span.End > len(source) ||
		!files.IsRuneBoundary(source, line_span.Start) || !files.IsRuneBoundary(source, line_span.End) {
		given := line_span.End
		if line_span.Start >= len(source) {
			given = line_span.Start
		}
		return "", files.IndexTooLargeError{Given: given, Max: len(source) - 1}
	}
	return source[line_span.Start:line_span.End], nil
}

// Translates a byte index in a file into an LSP position.
func ByteIndexToPosition(files files.Files, file_id int, byte_index int) (Position, error) {
	source, err := files.Source(file_id)
	if err != nil {
		return Position{}, err
	}

	line_index, err := files.LineIndex(file_id, byte_index)
	if err != nil {
		return Position{}, err
	}
	// NOTE: codespan-lsp 0.13.1 unwraps this (panics on an error), so return
	// the error instead.
	line_span, err := files.LineRange(file_id, line_index)
	if err != nil {
		return Position{}, err
	}

	line_str, err := lineStr(source, line_span)
	if err != nil {
		return Position{}, err
	}
	column := byte_index - line_span.Start

	return locationToPosition(line_str, line_index, column, byte_index)
}

// Translates a byte span in a file into an LSP range.
func ByteSpanToRange(files files.Files, file_id int, span files.Range) (Range, error) {
	start, err := ByteIndexToPosition(files, file_id, span.Start)
	if err != nil {
		return Range{}, err
	}
	end, err := ByteIndexToPosition(files, file_id, span.End)
	if err != nil {
		return Range{}, err
	}

	return Range{Start: start, End: end}, nil
}

func characterToLineOffset(line string, character uint32) (int, error) {
	line_len := len(line)
	character_offset := uint32(0)

	for byte_offset, ch := range line {
		if character_offset == character {
			return byte_offset, nil
		}

		character_offset += uint32(utf16.RuneLen(ch)) //nolint:gosec // 1 or 2
	}

	// Handle positions after the last character on the line
	if character_offset == character {
		return line_len, nil
	}
	return 0, files.ColumnTooLargeError{
		Given: int(character_offset),
		Max:   len(line),
	}
}

// Translates an LSP position into a byte index in a file.
func PositionToByteIndex(files files.Files, file_id int, position Position) (int, error) {
	source, err := files.Source(file_id)
	if err != nil {
		return 0, err
	}

	// NOTE: codespan-lsp 0.13.1 unwraps these (panics on an error, e.g. for a
	// line past the end of the file), so return the errors instead.
	line_span, err := files.LineRange(file_id, int(position.Line))
	if err != nil {
		return 0, err
	}
	line_str, err := lineStr(source, line_span)
	if err != nil {
		return 0, err
	}

	byte_offset, err := characterToLineOffset(line_str, position.Character)
	if err != nil {
		return 0, err
	}

	return line_span.Start + byte_offset, nil
}

// Translates an LSP range into a byte span in a file.
func RangeToByteSpan(files files.Files, file_id int, range_ Range) (fileRange, error) {
	start, err := PositionToByteIndex(files, file_id, range_.Start)
	if err != nil {
		return fileRange{}, err
	}
	end, err := PositionToByteIndex(files, file_id, range_.End)
	if err != nil {
		return fileRange{}, err
	}

	return newRange(start, end), nil
}
