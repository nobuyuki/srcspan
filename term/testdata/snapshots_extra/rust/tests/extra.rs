//! Snapshot tests that are not in upstream `codespan-reporting/tests/term.rs`.
//!
//! `tests/support` and `test_emit!` are copied from codespan-reporting 0.13.1.
//! The only change is that `test_emit!` writes the snapshots to
//! `term/testdata/snapshots_extra` instead of `tests/snapshots`.

use codespan_reporting::diagnostic::{Diagnostic, Label};
use codespan_reporting::files::SimpleFile;
use codespan_reporting::term::{Chars, Config, DisplayStyle};
use std::sync::LazyLock;

mod support;

use self::support::TestData;

static TEST_CONFIG: LazyLock<Config> = LazyLock::new(Config::default);

type LazyTestData<'a, T> = LazyLock<TestData<'a, T>>;

macro_rules! test_emit {
    (rich_color) => {
        #[test]
        #[cfg(feature = "termcolor")]
        fn rich_color() {
            let config = Config {
                display_style: DisplayStyle::Rich,
                ..TEST_CONFIG.clone()
            };

            insta::with_settings!({ snapshot_path => "../.." }, {
                insta::assert_snapshot!(TEST_DATA.emit_color(&config));
            });
        }
    };
    (medium_color) => {
        #[test]
        #[cfg(feature = "termcolor")]
        fn medium_color() {
            let config = Config {
                display_style: DisplayStyle::Medium,
                ..TEST_CONFIG.clone()
            };

            insta::with_settings!({ snapshot_path => "../.." }, {
                insta::assert_snapshot!(TEST_DATA.emit_color(&config));
            });
        }
    };
    (short_color) => {
        #[test]
        #[cfg(feature = "termcolor")]
        fn short_color() {
            let config = Config {
                display_style: DisplayStyle::Short,
                ..TEST_CONFIG.clone()
            };

            insta::with_settings!({ snapshot_path => "../.." }, {
                insta::assert_snapshot!(TEST_DATA.emit_color(&config));
            });
        }
    };
    (rich_no_color) => {
        #[test]
        fn rich_no_color() {
            let config = Config {
                display_style: DisplayStyle::Rich,
                ..TEST_CONFIG.clone()
            };

            insta::with_settings!({ snapshot_path => "../.." }, {
                insta::assert_snapshot!(TEST_DATA.emit_no_color(&config));
            });
        }
    };
    (medium_no_color) => {
        #[test]
        fn medium_no_color() {
            let config = Config {
                display_style: DisplayStyle::Medium,
                ..TEST_CONFIG.clone()
            };

            insta::with_settings!({ snapshot_path => "../.." }, {
                insta::assert_snapshot!(TEST_DATA.emit_no_color(&config));
            });
        }
    };
    (short_no_color) => {
        #[test]
        fn short_no_color() {
            let config = Config {
                display_style: DisplayStyle::Short,
                ..TEST_CONFIG.clone()
            };

            insta::with_settings!({ snapshot_path => "../.." }, {
                insta::assert_snapshot!(TEST_DATA.emit_no_color(&config));
            });
        }
    };
    (rich_ascii_no_color) => {
        #[test]
        fn rich_ascii_no_color() {
            let config = Config {
                display_style: DisplayStyle::Rich,
                chars: Chars::ascii(),
                ..TEST_CONFIG.clone()
            };

            insta::with_settings!({ snapshot_path => "../.." }, {
                insta::assert_snapshot!(TEST_DATA.emit_no_color(&config));
            });
        }
    };
}

/// A multi-line label ends on the line where another multi-line label starts.
/// Covers the branch in `render_snippet_source` that draws an inner gutter
/// column for a top label after the current one, or for a bottom label
/// before it.
mod multiline_end_and_start_on_same_line {
    use super::*;

    static TEST_DATA: LazyTestData<'static, SimpleFile<&'static str, String>> =
        LazyLock::new(|| {
            let file = SimpleFile::new(
                "main.rs",
                unindent::unindent(
                    r#"
                    fn main() {
                        let x = foo(
                            1,
                        ); let y = bar(
                            2,
                        );
                    }
                "#,
                ),
            );

            let diagnostics = vec![Diagnostic::error()
                .with_message("mismatched types")
                .with_labels(vec![
                    Label::primary((), 24..45).with_message("expected `u32`"),
                    Label::secondary((), 55..76).with_message("found `i32`"),
                ])];

            TestData {
                files: file,
                diagnostics,
            }
        });

    test_emit!(rich_color);
    test_emit!(medium_color);
    test_emit!(short_color);
    test_emit!(rich_no_color);
    test_emit!(medium_no_color);
    test_emit!(short_no_color);
    test_emit!(rich_ascii_no_color);
}
