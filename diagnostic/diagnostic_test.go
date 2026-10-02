package diagnostic

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nobuyuki/srcspan/files"
)

func Test(t *testing.T) {
	assert.Greater(t, Bug, Error)
	assert.Greater(t, Error, Warning)
	assert.Greater(t, Warning, Note)
	assert.Greater(t, Note, Help)

	assert.Greater(t, Secondary, Primary)
}

func TestWithLabelsIter(t *testing.T) {
	label0 := *NewPrimary(0, files.NewRange(0, 1))
	label1 := *NewSecondary(0, files.NewRange(1, 2))
	label2 := *NewSecondary(0, files.NewRange(2, 3))

	d := NewError().WithLabel(label0).WithLabelsIter(slices.Values([]Label{label1, label2}))
	assert.Equal(t, []Label{label0, label1, label2}, d.Labels)

	// Any `iter.Seq`, not only a slice.
	ranges := func(yield func(Label) bool) {
		for i := range 3 {
			if !yield(*NewPrimary(1, files.NewRange(i, i+1))) {
				return
			}
		}
	}
	d = NewError().WithLabelsIter(ranges)
	assert.Len(t, d.Labels, 3)
	assert.Equal(t, files.NewRange(2, 3), d.Labels[2].Range)
}

func TestWithNotesIter(t *testing.T) {
	d := NewError().WithNote("first").WithNotesIter(strings.SplitSeq("second\nthird", "\n"))
	assert.Equal(t, []string{"first", "second", "third"}, d.Notes)
}
