package files

// A byte range in a source, half-open: [Start, End).
// (`core::ops::Range<usize>` in Rust)
type Range struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

func NewRange(s, e int) Range {
	return Range{
		Start: s,
		End:   e,
	}
}

// Reports whether the byte index `i` is the first byte of a UTF-8 code
// point or the end of `s`.
// (`str::is_char_boundary` in Rust)
func IsRuneBoundary(s string, i int) bool {
	const utf8Mask = 0xC0
	const utf8Continue = 0x80

	if i < 0 || i > len(s) {
		return false
	}
	if i == len(s) {
		return true
	}
	return (s[i] & utf8Mask) != utf8Continue
}
