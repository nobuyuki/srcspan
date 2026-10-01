// A port of the `unindent` crate in Rust, used by the examples and tests.
package unindent

import (
	"strings"
)

// Removes the common leading spaces of the lines after the first one, and
// the first line if it is empty.
// (A port of `unindent::unindent` from the `unindent` crate in Rust.)
func Unindent(s string) string {
	// Document may start either on the same line as opening quote or on
	// the next line
	ignore_first_line := strings.HasPrefix(s, "\n") || strings.HasPrefix(s, "\r\n")

	lines := strings.Split(s, "\n")

	// Largest number of spaces that can be removed from every
	// non-whitespace-only line after the first
	spaces := -1
	for _, line := range lines[1:] {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.TrimSpace(trimmed) == "" {
			continue
		}
		n := len(line) - len(trimmed)
		if spaces < 0 || n < spaces {
			spaces = n
		}
	}
	spaces = max(spaces, 0)

	var result strings.Builder
	for i, line := range lines {
		if i > 1 || (i == 1 && !ignore_first_line) {
			result.WriteString("\n")
		}
		if i == 0 {
			// Do not un-indent anything on same line as opening quote
			result.WriteString(line)
		} else if len(line) > spaces {
			// Whitespace-only lines may have fewer lines than the number
			// to be removed
			result.WriteString(line[spaces:])
		}
	}
	return result.String()
}
