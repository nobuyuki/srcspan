// Copyright 2026 Nobuyuki Tomizawa
// SPDX-License-Identifier: Apache-2.0

// Package srcspan provides utilities for working with source code and
// printing nicely formatted diagnostic information like warnings and errors.
//
// It is a port of codespan, codespan-reporting and codespan-lsp 0.13.1
// (https://github.com/brendanzab/codespan), and follows the structure of
// these Rust crates:
//
//   - This package corresponds to the codespan crate: [Span], and [Files], a
//     database of source files.
//   - [srcspan/files] provides the Files interface,
//     through which diagnostics read source files, and its simple
//     implementations SimpleFile and SimpleFiles.
//   - [srcspan/diagnostic] provides the diagnostic data
//     structures.
//   - [srcspan/term] renders diagnostics to a terminal.
//   - [srcspan/lsp] translates byte indices and spans
//     into Language Server Protocol positions and ranges (codespan-lsp).
//
// A diagnostic is built with diagnostic.NewError (or NewWarning, ...) and its
// With* methods, and emitted with term.EmitToWriteStyle (with colors) or
// term.EmitToString (as plain text).
//
// See docs/porting-from-rust.md in the repository for how the Rust API maps
// to this one.
//
// (`codespan` in Rust)
package srcspan
