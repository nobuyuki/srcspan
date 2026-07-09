package srcspan

type Range struct {
	Start int
	End   int
}

func NewRange(s, e int) Range {
	return Range{
		Start: s,
		End:   e,
	}
}

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
