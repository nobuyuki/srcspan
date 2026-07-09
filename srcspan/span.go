package srcspan

import (
	"fmt"
)

type Span struct {
	start int
	end   int
}

// Create a new span from a starting and ending span.
func NewSpan(start, end int) Span {
	return Span{
		start: start,
		end:   end,
	}
}

// Gives an empty span at the start of a source.
func DefaultSpan() Span {
	return Span{
		start: 0,
		end:   0,
	}
}

// Measure the span of a string.
func SpanFromStr(s string) Span {
	return NewSpan(0, len(s))
}

// Combine two spans by taking the start of the earlier span
// and the end of the later span.
//
// Note: this will work even if the two spans are disjoint.
// If this doesn't make sense in your application, you should handle it yourself.
// In that case, you can use `Span.Disjoint` as a convenience function.
func (s Span) Merge(other Span) Span {
	st := min(s.start, other.start)
	en := max(s.end, other.end)
	return NewSpan(st, en)
}

// A helper function to tell whether two spans do not overlap.
func (s Span) Disjoint(other Span) bool {
	if s.end < other.end {
		return s.end <= other.start
	}
	return other.end <= s.start
}

// Get the starting byte index.
func (s Span) Start() int {
	return s.start
}

// Get the ending byte index.
func (s Span) End() int {
	return s.end
}

func (s Span) String() string {
	return fmt.Sprintf("[%d, %d)", s.start, s.end)
}
