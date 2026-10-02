package srcspan

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMerge(t *testing.T) {
	// overlap
	a := NewSpan(1, 5)
	b := NewSpan(3, 10)
	assert.Equal(t, a.Merge(b), NewSpan(1, 10))
	assert.Equal(t, b.Merge(a), NewSpan(1, 10))

	// subset
	two_four := NewSpan(2, 4)
	assert.Equal(t, a.Merge(two_four), NewSpan(1, 5))
	assert.Equal(t, two_four.Merge(a), NewSpan(1, 5))

	// disjoint
	ten_twenty := NewSpan(10, 20)
	assert.Equal(t, a.Merge(ten_twenty), NewSpan(1, 20))
	assert.Equal(t, ten_twenty.Merge(a), NewSpan(1, 20))

	// identity
	assert.Equal(t, a.Merge(a), a)
}

func TestDisjoint(t *testing.T) {
	// overlap
	a := NewSpan(1, 5)
	b := NewSpan(3, 10)

	assert.False(t, a.Disjoint(b))
	assert.False(t, b.Disjoint(a))

	// subset
	two_four := NewSpan(2, 4)
	assert.False(t, a.Disjoint(two_four))
	assert.False(t, two_four.Disjoint(a))

	// disjoint
	ten_twenty := NewSpan(10, 20)
	assert.True(t, a.Disjoint(ten_twenty))
	assert.True(t, ten_twenty.Disjoint(a))

	// identity
	assert.False(t, a.Disjoint(a))

	// off by one (upper bound)
	c := NewSpan(5, 10)
	assert.True(t, a.Disjoint(c))
	assert.True(t, c.Disjoint(a))
	// off by one (lower bound)
	d := NewSpan(0, 1)
	assert.True(t, a.Disjoint(d))
	assert.True(t, d.Disjoint(a))
}

func ExampleDefaultSpan() {
	span := DefaultSpan()

	fmt.Println(span)
	// Output: [0, 0)
}

func ExampleSpanFromStr() {
	span := SpanFromStr("hello")

	fmt.Println(span)
	// Output: [0, 5)
}

func ExampleSpan_Merge() {
	s1 := NewSpan(0, 4)
	s2 := NewSpan(10, 16)

	s0 := s1.Merge(s2)

	fmt.Println(s0)
	// Output: [0, 16)
}

func ExampleSpan_Disjoint() {
	s1 := NewSpan(0, 4)
	s2 := NewSpan(10, 16)

	fmt.Println(s1.Disjoint(s2))
	// Output: true
}

func ExampleSpan_Start() {
	s := NewSpan(0, 4)

	fmt.Println(s.Start())
	// Output: 0
}

func ExampleSpan_End() {
	s := NewSpan(0, 4)

	fmt.Println(s.End())
	// Output: 4
}

func ExampleSpan_String() {
	s := NewSpan(0, 4)

	fmt.Println(s, s.String())
	// Output: [0, 4) [0, 4)
}
