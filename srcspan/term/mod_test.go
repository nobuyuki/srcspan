package term

import (
	"testing"
	//"github.com/stretchr/testify/assert"
)

func TestUnsizedEmit(t *testing.T) {
}

/*
#[cfg(test)]
mod tests {
    use alloc::vec::Vec;

    use super::*;

    use crate::diagnostic::Label;
    use crate::files::SimpleFiles;

    /// Test range of 0 to 0 and check does not crash
    #[test]
    fn unsized_emit() {
        let mut files = SimpleFiles::new();

        let id = files.add("test", "");
        let zero_range = 0..0;
        let diagnostic = Diagnostic::bug().with_label(Label::primary(id, zero_range));
        emit_into_string(&Config::default(), &files, &diagnostic).unwrap();
    }
    }
*/
