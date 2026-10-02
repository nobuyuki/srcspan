package srcspan

import (
	"encoding/json"
	"fmt"

	"github.com/nobuyuki/srcspan/files"
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

// Convert a range into a span.
// (`impl From<Range<I>> for Span` in Rust)
func SpanFromRange(r files.Range) Span {
	return NewSpan(r.Start, r.End)
}

// Convert the span into a range.
// (`impl From<Span> for Range<usize>` in Rust)
func (s Span) Range() files.Range {
	return files.NewRange(s.start, s.end)
}

// The JSON representation of `Span`, like serde in Rust (the
// `serialization` feature): {"start":0,"end":4}.
type spanJSON struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Encodes the span like serde in Rust.
func (s Span) MarshalJSON() ([]byte, error) {
	return json.Marshal(spanJSON{Start: s.start, End: s.end})
}

// Decodes a span encoded like serde in Rust.
func (s *Span) UnmarshalJSON(data []byte) error {
	var j spanJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*s = NewSpan(j.Start, j.End)
	return nil
}
