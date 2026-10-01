package term

// Snapshot tests ported from `codespan-reporting/tests/term.rs` (0.13.1).
//
// The expected outputs are the upstream `insta` snapshots, copied as-is to
// testdata/snapshots. The source literals are copied from term.rs.

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srcspan/internal/unindent"
	"srcspan/srcspan/diagnostic"
	"srcspan/srcspan/files"
)

// (`tests/support/mod.rs::TestData` in Rust)
type testData struct {
	files       files.Files
	diagnostics []*diagnostic.Diagnostic
}

func (d *testData) emitNoColor(t *testing.T, config *Config) string {
	t.Helper()

	var writer strings.Builder
	for _, diagnostic := range d.diagnostics {
		require.NoError(t, EmitToString(&writer, config, d.files, diagnostic))
	}
	return writer.String()
}

func (d *testData) emitColor(t *testing.T, config *Config) string {
	t.Helper()

	writer := &colorBuffer{}
	for _, diagnostic := range d.diagnostics {
		require.NoError(t, EmitToWriteStyle(writer, config, d.files, diagnostic))
	}
	return writer.String()
}

// A color specification. (`termcolor::ColorSpec` in Rust, limited to the
// attributes used by the default styles.)
type colorSpec struct {
	fg      string
	bold    bool
	intense bool
}

// The default styles. (`impl Default for Styles` in Rust)
var (
	header         = colorSpec{bold: true, intense: true}
	headerBug      = colorSpec{fg: "Red", bold: true, intense: true}
	headerError    = colorSpec{fg: "Red", bold: true, intense: true}
	headerWarning  = colorSpec{fg: "Yellow", bold: true, intense: true}
	headerNote     = colorSpec{fg: "Green", bold: true, intense: true}
	headerHelp     = colorSpec{fg: "Cyan", bold: true, intense: true}
	headerMessage  = header
	primaryLabel   = map[diagnostic.Severity]colorSpec{diagnostic.Bug: {fg: "Red"}, diagnostic.Error: {fg: "Red"}, diagnostic.Warning: {fg: "Yellow"}, diagnostic.Note: {fg: "Green"}, diagnostic.Help: {fg: "Cyan"}}
	secondaryLabel = colorSpec{fg: "Cyan"}
	lineNumber     = colorSpec{fg: "Cyan"}
	sourceBorder   = colorSpec{fg: "Cyan"}
	noteBullet     = colorSpec{fg: "Cyan"}
)

// A `WriteStyle` for creating visually inspectable representations of
// colored output, so they can be easily tested.
//
// A new color is represented as `{style}` and a reset is represented by
// `{/}`. (`tests/support/color_buffer.rs::ColorBuffer` in Rust, used
// through the `WriteStyle` implementation for `WriteColor` with the
// default styles.)
type colorBuffer struct {
	buf   strings.Builder
	color colorSpec
}

func (w *colorBuffer) String() string {
	return w.buf.String()
}

func (w *colorBuffer) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *colorBuffer) setColor(spec colorSpec) error {
	if w.color == spec {
		return nil
	}
	w.color = spec

	if spec == (colorSpec{}) {
		w.buf.WriteString("{/}")
		return nil
	}

	attrs := []string{}
	if spec.fg != "" {
		attrs = append(attrs, "fg:"+spec.fg)
	}
	if spec.bold {
		attrs = append(attrs, "bold")
	}
	if spec.intense {
		attrs = append(attrs, "bright")
	}
	w.buf.WriteString("{" + strings.Join(attrs, " ") + "}")
	return nil
}

func (w *colorBuffer) SetHeader(severity diagnostic.Severity) error {
	switch severity {
	case diagnostic.Bug:
		return w.setColor(headerBug)
	case diagnostic.Error:
		return w.setColor(headerError)
	case diagnostic.Warning:
		return w.setColor(headerWarning)
	case diagnostic.Note:
		return w.setColor(headerNote)
	default:
		return w.setColor(headerHelp)
	}
}

func (w *colorBuffer) SetHeaderMessage() error { return w.setColor(headerMessage) }
func (w *colorBuffer) SetLineNumber() error    { return w.setColor(lineNumber) }
func (w *colorBuffer) SetNoteBullet() error    { return w.setColor(noteBullet) }
func (w *colorBuffer) SetSourceBorder() error  { return w.setColor(sourceBorder) }

func (w *colorBuffer) SetLabel(severity diagnostic.Severity, label_style diagnostic.LabelStyle) error {
	if label_style == diagnostic.Primary {
		return w.setColor(primaryLabel[severity])
	}
	return w.setColor(secondaryLabel)
}

func (w *colorBuffer) Reset() error {
	if w.color != (colorSpec{}) {
		w.buf.WriteString("{/}")
		w.color = colorSpec{}
	}
	return nil
}

var _ WriteStyle = (*colorBuffer)(nil)

// Reads the contents of an `insta` snapshot file, skipping its metadata.
func readSnapshot(t *testing.T, name string) string {
	t.Helper()

	f, err := os.Open(filepath.Join("testdata", "snapshots", name))
	require.NoError(t, err)
	defer f.Close()

	var contents strings.Builder
	scanner := bufio.NewScanner(f)
	in_metadata := false
	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()
		if i == 0 && line == "---" {
			in_metadata = true
			continue
		}
		if in_metadata {
			if line == "---" {
				in_metadata = false
			}
			continue
		}
		contents.WriteString(line)
		contents.WriteString("\n")
	}
	require.NoError(t, scanner.Err())
	return contents.String()
}

// Normalizes snapshot contents like `insta` does before comparing them.
// (`TextSnapshotContents::normalize` in insta)
func normalizeSnapshot(s string) string {
	s = strings.TrimLeft(s, "\r\n")
	s = strings.TrimRightFunc(s, unicode.IsSpace)
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func assertSnapshot(t *testing.T, name string, actual string) {
	t.Helper()

	expected := readSnapshot(t, name)
	assert.Equal(t, normalizeSnapshot(expected), normalizeSnapshot(actual))
}

// A test of `test_emit!` in Rust, or a test function written by hand.
type snapshotTest struct {
	name   string
	config func(config Config) Config
	// Whether to use `emit_color` instead of `emit_no_color`.
	color bool
}

// The variants of `test_emit!`.
var (
	richColor = snapshotTest{"rich_color", func(c Config) Config {
		c.DisplayStyle = Rich
		return c
	}, true}
	mediumColor = snapshotTest{"medium_color", func(c Config) Config {
		c.DisplayStyle = Medium
		return c
	}, true}
	shortColor = snapshotTest{"short_color", func(c Config) Config {
		c.DisplayStyle = Short
		return c
	}, true}

	richNoColor = snapshotTest{"rich_no_color", func(c Config) Config {
		c.DisplayStyle = Rich
		return c
	}, false}
	mediumNoColor = snapshotTest{"medium_no_color", func(c Config) Config {
		c.DisplayStyle = Medium
		return c
	}, false}
	shortNoColor = snapshotTest{"short_no_color", func(c Config) Config {
		c.DisplayStyle = Short
		return c
	}, false}
	richASCIINoColor = snapshotTest{"rich_ascii_no_color", func(c Config) Config {
		c.DisplayStyle = Rich
		c.Chars = Ascii()
		return c
	}, false}
)

func tabWidthNoColor(tab_width int) snapshotTest {
	return snapshotTest{"tab_width_" + strconv.Itoa(tab_width) + "_no_color", func(c Config) Config {
		c.TabWidth = tab_width
		return c
	}, false}
}

var tabWidthDefaultNoColor = snapshotTest{"tab_width_default_no_color", func(c Config) Config {
	return c
}, false}

// A module of term.rs.
type snapshotCase struct {
	name string
	// `TEST_CONFIG` of the module.
	config func() Config
	data   func() *testData
	tests  []snapshotTest
}

func TestSnapshots(t *testing.T) {
	for _, c := range snapshotCases {
		data := c.data()
		for _, test := range c.tests {
			t.Run(c.name+"/"+test.name, func(t *testing.T) {
				config := test.config(c.config())
				var actual string
				if test.color {
					actual = data.emitColor(t, &config)
				} else {
					actual = data.emitNoColor(t, &config)
				}
				assertSnapshot(t, "term__"+c.name+"__"+test.name+".snap", actual)
			})
		}
	}
}

func label(label *diagnostic.Label) diagnostic.Label {
	return *label
}

var snapshotCases = []snapshotCase{
	{
		name:   "empty",
		config: DefaultConfig,
		data: func() *testData {
			fs := files.NewSimpleFiles()

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewBug(),
				diagnostic.NewError(),
				diagnostic.NewWarning(),
				diagnostic.NewNote(),
				diagnostic.NewHelp(),
				diagnostic.NewBug(),
			}

			return &testData{&fs, diagnostics}
		},
		tests: []snapshotTest{richColor, mediumColor, shortColor, richNoColor, mediumNoColor, shortNoColor, richASCIINoColor},
	},
	{
		name:   "same_line",
		config: DefaultConfig,
		data: func() *testData {
			fs := files.NewSimpleFiles()

			file_id1 := fs.Add("one_line.rs", unindent.Unindent("\n                    fn main() {\n                        let mut v = vec![Some(\"foo\"), Some(\"bar\")];\n                        v.push(v.pop().unwrap());\n                    }\n                "))

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewError().
					WithCode("E0499").
					WithMessage("cannot borrow `v` as mutable more than once at a time").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id1, files.NewRange(71, 72)).
							WithMessage("second mutable borrow occurs here")),
						label(diagnostic.NewSecondary(file_id1, files.NewRange(64, 65)).
							WithMessage("first borrow later used by call")),
						label(diagnostic.NewSecondary(file_id1, files.NewRange(66, 70)).
							WithMessage("first mutable borrow occurs here")),
					}),
				diagnostic.NewError().
					WithMessage("aborting due to previous error").
					WithNotes([]string{
						"For more information about this error, try `rustc --explain E0499`.",
					}),
			}

			return &testData{&fs, diagnostics}
		},
		tests: []snapshotTest{richColor, mediumColor, shortColor, richNoColor, mediumNoColor, shortNoColor, richASCIINoColor},
	},
	{
		name:   "overlapping",
		config: DefaultConfig,
		data: func() *testData {
			fs := files.NewSimpleFiles()

			file_id1 := fs.Add("nested_impl_trait.rs", unindent.Unindent("\n                    use std::fmt::Debug;\n\n                    fn fine(x: impl Into<u32>) -> impl Into<u32> { x }\n\n                    fn bad_in_ret_position(x: impl Into<u32>) -> impl Into<impl Debug> { x }\n                "))
			file_id2 := fs.Add("typeck_type_placeholder_item.rs", unindent.Unindent("\n                    fn fn_test1() -> _ { 5 }\n                    fn fn_test2(x: i32) -> (_, _) { (x, x) }\n                "))
			file_id3 := fs.Add("libstd/thread/mod.rs", unindent.Unindent("\n                    #[stable(feature = \"rust1\", since = \"1.0.0\")]\n                    pub fn spawn<F, T>(self, f: F) -> io::Result<JoinHandle<T>>\n                    where\n                        F: FnOnce() -> T,\n                        F: Send + 'static,\n                        T: Send + 'static,\n                    {\n                        unsafe { self.spawn_unchecked(f) }\n                    }\n                "))
			file_id4 := fs.Add("no_send_res_ports.rs", unindent.Unindent("\n                    use std::thread;\n                    use std::rc::Rc;\n\n                    #[derive(Debug)]\n                    struct Port<T>(Rc<T>);\n\n                    fn main() {\n                        #[derive(Debug)]\n                        struct Foo {\n                            _x: Port<()>,\n                        }\n\n                        impl Drop for Foo {\n                            fn drop(&mut self) {}\n                        }\n\n                        fn foo(x: Port<()>) -> Foo {\n                            Foo {\n                                _x: x\n                            }\n                        }\n\n                        let x = foo(Port(Rc::new(())));\n\n                        thread::spawn(move|| {\n                            let y = x;\n                            println!(\"{:?}\", y);\n                        });\n                    }\n                "))

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewError().
					WithCode("E0666").
					WithMessage("nested `impl Trait` is not allowed").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id1, files.NewRange(129, 139)).
							WithMessage("nested `impl Trait` here")),
						label(diagnostic.NewSecondary(file_id1, files.NewRange(119, 140)).
							WithMessage("outer `impl Trait`")),
					}),
				diagnostic.NewError().
					WithCode("E0121").
					WithMessage("the type placeholder `_` is not allowed within types on item signatures").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id2, files.NewRange(17, 18)).
							WithMessage("not allowed in type signatures")),
						label(diagnostic.NewSecondary(file_id2, files.NewRange(17, 18)).
							WithMessage("help: replace with the correct return type: `i32`")),
					}),
				diagnostic.NewError().
					WithCode("E0121").
					WithMessage("the type placeholder `_` is not allowed within types on item signatures").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id2, files.NewRange(49, 50)).
							WithMessage("not allowed in type signatures")),
						label(diagnostic.NewPrimary(file_id2, files.NewRange(52, 53)).
							WithMessage("not allowed in type signatures")),
						label(diagnostic.NewSecondary(file_id2, files.NewRange(48, 54)).
							WithMessage("help: replace with the correct return type: `(i32, i32)`")),
					}),
				diagnostic.NewError().
					WithCode("E0277").
					WithMessage("`std::rc::Rc<()>` cannot be sent between threads safely").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id4, files.NewRange(339, 352)).
							WithMessage("`std::rc::Rc<()>` cannot be sent between threads safely")),
						label(diagnostic.NewSecondary(file_id4, files.NewRange(353, 416)).
							WithMessage("within this `[closure@no_send_res_ports.rs:29:19: 33:6 x:main::Foo]`")),
						label(diagnostic.NewSecondary(file_id3, files.NewRange(141, 145)).
							WithMessage("required by this bound in `std::thread::spawn`")),
					}).
					WithNotes([]string{
						"help: within `[closure@no_send_res_ports.rs:29:19: 33:6 x:main::Foo]`, the trait `std::marker::Send` is not implemented for `std::rc::Rc<()>`",
						"note: required because it appears within the type `Port<()>`",
						"note: required because it appears within the type `main::Foo`",
						"note: required because it appears within the type `[closure@no_send_res_ports.rs:29:19: 33:6 x:main::Foo]`",
					}),
				diagnostic.NewError().
					WithMessage("aborting due 5 previous errors").
					WithNotes([]string{
						"Some errors have detailed explanations: E0121, E0277, E0666.",
						"For more information about an error, try `rustc --explain E0121`.",
					}),
			}

			return &testData{&fs, diagnostics}
		},
		tests: []snapshotTest{richColor, mediumColor, shortColor, richNoColor, mediumNoColor, shortNoColor, richASCIINoColor},
	},
	{
		name:   "message",
		config: DefaultConfig,
		data: func() *testData {
			fs := files.NewSimpleFiles()

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewError().WithMessage("a message"),
				diagnostic.NewWarning().WithMessage("a message"),
				diagnostic.NewNote().WithMessage("a message"),
				diagnostic.NewHelp().WithMessage("a message"),
			}

			return &testData{&fs, diagnostics}
		},
		tests: []snapshotTest{richColor, mediumColor, shortColor, richNoColor, mediumNoColor, shortNoColor, richASCIINoColor},
	},
	{
		name:   "message_and_notes",
		config: DefaultConfig,
		data: func() *testData {
			fs := files.NewSimpleFiles()

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewError().WithMessage("a message").WithNotes([]string{"a note"}),
				diagnostic.NewWarning().WithMessage("a message").WithNotes([]string{"a note"}),
				diagnostic.NewNote().WithMessage("a message").WithNotes([]string{"a note"}),
				diagnostic.NewHelp().WithMessage("a message").WithNotes([]string{"a note"}),
			}

			return &testData{&fs, diagnostics}
		},
		tests: []snapshotTest{richColor, mediumColor, shortColor, richNoColor, mediumNoColor, shortNoColor, richASCIINoColor},
	},
	{
		name:   "message_errorcode",
		config: DefaultConfig,
		data: func() *testData {
			fs := files.NewSimpleFiles()

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewError().WithMessage("a message").WithCode("E0001"),
				diagnostic.NewWarning().WithMessage("a message").WithCode("W001"),
				diagnostic.NewNote().WithMessage("a message").WithCode("N0815"),
				diagnostic.NewHelp().WithMessage("a message").WithCode("H4711"),
				diagnostic.NewError().WithMessage("where did my errorcode go?").WithCode(""),
				diagnostic.NewWarning().WithMessage("where did my errorcode go?").WithCode(""),
				diagnostic.NewNote().WithMessage("where did my errorcode go?").WithCode(""),
				diagnostic.NewHelp().WithMessage("where did my errorcode go?").WithCode(""),
			}

			return &testData{&fs, diagnostics}
		},
		tests: []snapshotTest{richNoColor, shortNoColor, richASCIINoColor},
	},
	{
		name:   "empty_ranges",
		config: DefaultConfig,
		data: func() *testData {
			file := files.NewSimpleFile("hello", "Hello world!\nBye world!\n   ")
			source, _ := file.Source(0)
			eof := len(source)

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewNote().
					WithMessage("middle").
					WithLabels([]diagnostic.Label{label(diagnostic.NewPrimary(0, files.NewRange(6, 6)).WithMessage("middle"))}),
				diagnostic.NewNote().
					WithMessage("end of line").
					WithLabels([]diagnostic.Label{label(diagnostic.NewPrimary(0, files.NewRange(12, 12)).WithMessage("end of line"))}),
				diagnostic.NewNote().
					WithMessage("end of line").
					WithLabels([]diagnostic.Label{label(diagnostic.NewPrimary(0, files.NewRange(23, 23)).WithMessage("end of line"))}),
				diagnostic.NewNote().
					WithMessage("end of file").
					WithLabels([]diagnostic.Label{label(diagnostic.NewPrimary(0, files.NewRange(eof, eof)).WithMessage("end of file"))}),
			}

			return &testData{&file, diagnostics}
		},
		tests: []snapshotTest{richColor, mediumColor, shortColor, richNoColor, mediumNoColor, shortNoColor, richASCIINoColor},
	},
	{
		name:   "same_ranges",
		config: DefaultConfig,
		data: func() *testData {
			file := files.NewSimpleFile("same_range", "::S { }")

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewError().
					WithMessage("Unexpected token").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(0, files.NewRange(4, 4)).WithMessage("Unexpected '{'")),
						label(diagnostic.NewSecondary(0, files.NewRange(4, 4)).WithMessage("Expected '('")),
					}),
			}

			return &testData{&file, diagnostics}
		},
		tests: []snapshotTest{richColor, mediumColor, shortColor, richNoColor, mediumNoColor, shortNoColor, richASCIINoColor},
	},
	{
		name:   "multifile",
		config: DefaultConfig,
		data: func() *testData {
			fs := files.NewSimpleFiles()

			file_id1 := fs.Add("Data/Nat.fun", unindent.Unindent("\n                        module Data.Nat where\n\n                        data Nat : Type where\n                            zero : Nat\n                            succ : Nat → Nat\n\n                        {-# BUILTIN NATRAL Nat #-}\n\n                        infixl 6 _+_ _-_\n\n                        _+_ : Nat → Nat → Nat\n                        zero    + n₂ = n₂\n                        succ n₁ + n₂ = succ (n₁ + n₂)\n\n                        _-_ : Nat → Nat → Nat\n                        n₁      - zero    = n₁\n                        zero    - succ n₂ = zero\n                        succ n₁ - succ n₂ = n₁ - n₂\n                    "))
			file_id2 := fs.Add("Test.fun", unindent.Unindent("\n                        module Test where\n\n                        _ : Nat\n                        _ = 123 + \"hello\"\n                    "))

			diagnostics := []*diagnostic.Diagnostic{
				// Unknown builtin error
				diagnostic.NewError().
					WithMessage("unknown builtin: `NATRAL`").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id1, files.NewRange(96, 102)).WithMessage("unknown builtin")),
					}).
					WithNotes([]string{"there is a builtin with a similar name: `NATURAL`"}),
				// Unused parameter warning
				diagnostic.NewWarning().
					WithMessage("unused parameter pattern: `n₂`").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id1, files.NewRange(285, 289)).WithMessage("unused parameter")),
					}).
					WithNotes([]string{"consider using a wildcard pattern: `_`"}),
				// Unexpected type error
				diagnostic.NewError().
					WithMessage("unexpected type in application of `_+_`").
					WithCode("E0001").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id2, files.NewRange(37, 44)).
							WithMessage("expected `Nat`, found `String`")),
						label(diagnostic.NewSecondary(file_id1, files.NewRange(130, 155)).
							WithMessage("based on the definition of `_+_`")),
					}).
					WithNotes([]string{unindent.Unindent("\n                            expected type `Nat`\n                               found type `String`\n                        ")}),
			}

			return &testData{&fs, diagnostics}
		},
		tests: []snapshotTest{richColor, mediumColor, shortColor, richNoColor, mediumNoColor, shortNoColor, richASCIINoColor},
	},
	{
		name:   "fizz_buzz",
		config: DefaultConfig,
		data: func() *testData {
			fs := files.NewSimpleFiles()

			file_id := fs.Add("FizzBuzz.fun", unindent.Unindent("\n                        module FizzBuzz where\n\n                        fizz₁ : Nat → String\n                        fizz₁ num = case (mod num 5) (mod num 3) of\n                            0 0 => \"FizzBuzz\"\n                            0 _ => \"Fizz\"\n                            _ 0 => \"Buzz\"\n                            _ _ => num\n\n                        fizz₂ : Nat → String\n                        fizz₂ num =\n                            case (mod num 5) (mod num 3) of\n                                0 0 => \"FizzBuzz\"\n                                0 _ => \"Fizz\"\n                                _ 0 => \"Buzz\"\n                                _ _ => num\n                    "))

			diagnostics := []*diagnostic.Diagnostic{
				// Incompatible match clause error
				diagnostic.NewError().
					WithMessage("`case` clauses have incompatible types").
					WithCode("E0308").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id, files.NewRange(163, 166)).
							WithMessage("expected `String`, found `Nat`")),
						label(diagnostic.NewSecondary(file_id, files.NewRange(62, 166)).
							WithMessage("`case` clauses have incompatible types")),
						label(diagnostic.NewSecondary(file_id, files.NewRange(41, 47)).
							WithMessage("expected type `String` found here")),
					}).
					WithNotes([]string{unindent.Unindent("\n                            expected type `String`\n                               found type `Nat`\n                        ")}),
				// Incompatible match clause error
				diagnostic.NewError().
					WithMessage("`case` clauses have incompatible types").
					WithCode("E0308").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id, files.NewRange(328, 331)).
							WithMessage("expected `String`, found `Nat`")),
						label(diagnostic.NewSecondary(file_id, files.NewRange(211, 331)).
							WithMessage("`case` clauses have incompatible types")),
						label(diagnostic.NewSecondary(file_id, files.NewRange(258, 268)).
							WithMessage("this is found to be of type `String`")),
						label(diagnostic.NewSecondary(file_id, files.NewRange(284, 290)).
							WithMessage("this is found to be of type `String`")),
						label(diagnostic.NewSecondary(file_id, files.NewRange(306, 312)).
							WithMessage("this is found to be of type `String`")),
						label(diagnostic.NewSecondary(file_id, files.NewRange(186, 192)).
							WithMessage("expected type `String` found here")),
					}).
					WithNotes([]string{unindent.Unindent("\n                            expected type `String`\n                               found type `Nat`\n                        ")}),
			}

			return &testData{&fs, diagnostics}
		},
		tests: []snapshotTest{richColor, mediumColor, shortColor, richNoColor, mediumNoColor, shortNoColor, richASCIINoColor},
	},
	{
		name:   "multiline_overlapping",
		config: DefaultConfig,
		data: func() *testData {
			file := files.NewSimpleFile("codespan/src/file.rs", "        match line_index.compare(self.last_line_index()) {\n            Ordering::Less => Ok(self.line_starts()[line_index.to_usize()]),\n            Ordering::Equal => Ok(self.source_span().end()),\n            Ordering::Greater => LineIndexOutOfBoundsError {\n                given: line_index,\n                max: self.last_line_index(),\n            },\n        }")

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewError().
					WithMessage("match arms have incompatible types").
					WithCode("E0308").
					WithLabels([]diagnostic.Label{
						// this secondary label is before the primary label to test the locus calculation (see issue #259)
						label(diagnostic.NewSecondary(0, files.NewRange(89, 134)).
							WithMessage("this is found to be of type `Result<ByteIndex, LineIndexOutOfBoundsError>`")),
						label(diagnostic.NewPrimary(0, files.NewRange(230, 351)).
							WithMessage("expected enum `Result`, found struct `LineIndexOutOfBoundsError`")),
						label(diagnostic.NewSecondary(0, files.NewRange(8, 362)).
							WithMessage("`match` arms have incompatible types")),
						label(diagnostic.NewSecondary(0, files.NewRange(167, 195)).
							WithMessage("this is found to be of type `Result<ByteIndex, LineIndexOutOfBoundsError>`")),
					}).
					WithNotes([]string{unindent.Unindent("\n                            expected type `Result<ByteIndex, LineIndexOutOfBoundsError>`\n                               found type `LineIndexOutOfBoundsError`\n                        ")}),
			}

			return &testData{&file, diagnostics}
		},
		tests: []snapshotTest{richColor, mediumColor, shortColor, richNoColor, mediumNoColor, shortNoColor, richASCIINoColor},
	},
	{
		name:   "tabbed",
		config: DefaultConfig,
		data: func() *testData {
			fs := files.NewSimpleFiles()

			file_id := fs.Add("tabbed", unindent.Unindent("\n                Entity:\n                \tArmament:\n                \t\tWeapon: DogJaw\n                \t\tReloadingCondition:\tattack-cooldown\n                \tFoo: Bar"))

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewWarning().
					WithMessage("unknown weapon `DogJaw`").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id, files.NewRange(29, 35)).WithMessage("the weapon")),
					}),
				diagnostic.NewWarning().
					WithMessage("unknown condition `attack-cooldown`").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id, files.NewRange(58, 73)).WithMessage("the condition")),
					}),
				diagnostic.NewWarning().
					WithMessage("unknown field `Foo`").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id, files.NewRange(75, 78)).WithMessage("the field")),
					}),
			}

			return &testData{&fs, diagnostics}
		},
		tests: []snapshotTest{tabWidthDefaultNoColor, tabWidthNoColor(3), tabWidthNoColor(6)},
	},
	{
		name:   "tab_columns",
		config: DefaultConfig,
		data: func() *testData {
			fs := files.NewSimpleFiles()

			source := unindent.Unindent("\thello\n                ∙\thello\n                ∙∙\thello\n                ∙∙∙\thello\n                ∙∙∙∙\thello\n                ∙∙∙∙∙\thello\n                ∙∙∙∙∙∙\thello")
			hello_ranges := []files.Range{}
			for offset := 0; ; {
				i := strings.Index(source[offset:], "hello")
				if i < 0 {
					break
				}
				start := offset + i
				hello_ranges = append(hello_ranges, files.NewRange(start, start+len("hello")))
				offset = start + len("hello")
			}

			file_id := fs.Add("tab_columns", source)

			labels := []diagnostic.Label{}
			for _, range_ := range hello_ranges {
				labels = append(labels, *diagnostic.NewPrimary(file_id, range_))
			}
			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewWarning().WithMessage("tab test").WithLabels(labels),
			}

			return &testData{&fs, diagnostics}
		},
		tests: []snapshotTest{tabWidthDefaultNoColor, tabWidthNoColor(2), tabWidthNoColor(3), tabWidthNoColor(6)},
	},
	{
		name:   "unicode",
		config: DefaultConfig,
		data: func() *testData {
			prefix := "extern "
			abi := "\"路濫狼á́́\""
			suffix := " fn foo() {}"

			file := files.NewSimpleFile("unicode.rs", prefix+abi+suffix)

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewError().
					WithCode("E0703").
					WithMessage("invalid ABI: found `路濫狼á́́`").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(0, files.NewRange(len(prefix), len(prefix)+len(abi))).
							WithMessage("invalid ABI")),
					}).
					WithNotes([]string{unindent.Unindent("\n                            valid ABIs:\n                              - aapcs\n                              - amdgpu-kernel\n                              - C\n                              - cdecl\n                              - efiapi\n                              - fastcall\n                              - msp430-interrupt\n                              - platform-intrinsic\n                              - ptx-kernel\n                              - Rust\n                              - rust-call\n                              - rust-intrinsic\n                              - stdcall\n                              - system\n                              - sysv64\n                              - thiscall\n                              - unadjusted\n                              - vectorcall\n                              - win64\n                              - x86-interrupt\n                        ")}),
				diagnostic.NewError().
					WithMessage("aborting due to previous error").
					WithNotes([]string{
						"For more information about this error, try `rustc --explain E0703`.",
					}),
			}

			return &testData{&file, diagnostics}
		},
		tests: []snapshotTest{richNoColor, mediumNoColor, shortNoColor},
	},
	{
		name:   "unicode_spans",
		config: DefaultConfig,
		data: func() *testData {
			moon_phases := "🐄🌑🐄🌒🐄🌓🐄🌔🐄🌕🐄🌖🐄🌗🐄🌘🐄"
			invalid_start := 1
			invalid_end := len("🐄") - 1
			file := files.NewSimpleFile("moon_jump.rs", moon_phases)

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewError().
					WithCode("E01").
					WithMessage("cow may not jump during new moon.").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(0, files.NewRange(invalid_start, invalid_end)).WithMessage("Invalid jump")),
					}),
				diagnostic.NewNote().
					WithMessage("invalid unicode range").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewSecondary(0, files.NewRange(invalid_start, len("🐄"))).
							WithMessage("Cow range does not start at boundary.")),
					}),
				diagnostic.NewNote().
					WithMessage("invalid unicode range").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewSecondary(0, files.NewRange(len("🐄🌑"), len("🐄🌑🐄")-1)).
							WithMessage("Cow range does not end at boundary.")),
					}),
				diagnostic.NewNote().
					WithMessage("invalid unicode range").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewSecondary(0, files.NewRange(invalid_start, len("🐄🌑🐄")-1)).
							WithMessage("Cow does not start or end at boundary.")),
					}),
			}

			return &testData{&file, diagnostics}
		},
		tests: []snapshotTest{richNoColor, mediumNoColor, shortNoColor},
	},
	{
		name:   "position_indicator",
		config: DefaultConfig,
		data: func() *testData {
			file := files.NewSimpleFile("tests/main.js", unindent.Unindent("\"use strict\";\n                    let zero=0;\n                    function foo() {\n                      \"use strict\";\n                      one=1;\n                    }"))

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewWarning().
					WithCode("ParserWarning").
					WithMessage("The strict mode declaration in the body of function `foo` is redundant, as the outer scope is already in strict mode").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(0, files.NewRange(45, 57)).
							WithMessage("This strict mode declaration is redundant")),
						label(diagnostic.NewSecondary(0, files.NewRange(0, 12)).
							WithMessage("Strict mode is first declared here")),
					}),
			}

			return &testData{&file, diagnostics}
		},
		tests: []snapshotTest{richNoColor, mediumNoColor, shortNoColor, richASCIINoColor},
	},
	{
		name: "multiline_omit",
		config: func() Config {
			config := DefaultConfig()
			config.StartContextLines = 2
			config.EndContextLines = 1
			return config
		},
		data: func() *testData {
			fs := files.NewSimpleFiles()

			file_id1 := fs.Add("empty_if_comments.lua", strings.Join([]string{
				"elseif 3 then", // primary label starts here
				"",              // context line
				"",
				"",
				"",
				"",
				"",
				"",
				"",     // context line
				"else", // primary label ends here
			}, "\n"))

			file_id2 := fs.Add("src/lib.rs", strings.Join([]string{
				"fn main() {",
				"    1",   // primary label starts here
				"    + 1", // context line
				"    + 1", // skip
				"    + 1", // skip
				"    + 1", // skip
				"    +1",  // secondary label here
				"    + 1", // this single line will not be skipped; the previously filtered out label must be retrieved
				"    + 1", // context line
				"    + 1", // primary label ends here
				"}",
			}, "\n"))

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewError().
					WithMessage("empty elseif block").
					WithCode("empty_if").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id1, files.NewRange(0, 23))),
						label(diagnostic.NewSecondary(file_id1, files.NewRange(15, 21)).
							WithMessage("content should be in here")),
					}),
				diagnostic.NewError().
					WithMessage("mismatched types").
					WithCode("E0308").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id2, files.NewRange(17, 80)).WithMessage("expected (), found integer")),
						label(diagnostic.NewSecondary(file_id2, files.NewRange(55, 55)).WithMessage("missing whitespace")),
					}).
					WithNotes([]string{
						"note:\texpected type `()`\n\tfound type `{integer}`",
					}),
			}

			return &testData{&fs, diagnostics}
		},
		tests: []snapshotTest{richNoColor},
	},
	{
		name: "surrounding_lines",
		config: func() Config {
			config := DefaultConfig()
			config.BeforeLabelLines = 2
			config.AfterLabelLines = 1
			return config
		},
		data: func() *testData {
			fs := files.NewSimpleFiles()

			file_id := fs.Add("surroundingLines.fun", unindent.Unindent("\n                #[foo]\n                fn main() {\n                    println!(\n                        \"{}\",\n                        Foo\n                    );\n                }\n\n                struct Foo"))

			diagnostics := []*diagnostic.Diagnostic{
				diagnostic.NewError().
					WithMessage("Unknown attribute macro").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id, files.NewRange(2, 5)).
							WithMessage("No attribute macro `foo` known")),
					}),
				diagnostic.NewError().
					WithMessage("Missing argument for format").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id, files.NewRange(55, 58)).
							WithMessage("No instance of std::fmt::Display exists for type Foo")),
						label(diagnostic.NewSecondary(file_id, files.NewRange(42, 44)).
							WithMessage("Unable to use `{}`-directive to display `Foo`")),
					}),
				diagnostic.NewError().
					WithMessage("Syntax error").
					WithLabels([]diagnostic.Label{
						label(diagnostic.NewPrimary(file_id, files.NewRange(79, 79)).WithMessage("Missing a semicolon")),
					}),
			}

			return &testData{&fs, diagnostics}
		},
		tests: []snapshotTest{richNoColor},
	},
}
