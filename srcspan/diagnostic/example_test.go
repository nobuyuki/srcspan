package diagnostic_test

import (
	"fmt"
	"slices"

	"srcspan/srcspan/diagnostic"
	"srcspan/srcspan/files"
)

func ExampleDiagnostic_WithLabelsIter() {
	ranges := []files.Range{files.NewRange(0, 5), files.NewRange(6, 11)}

	// Labels can be added from any iterator, without collecting them into a
	// slice first.
	labels := func(yield func(diagnostic.Label) bool) {
		for _, range_ := range ranges {
			if !yield(*diagnostic.NewPrimary(0, range_).WithMessage("a word")) {
				return
			}
		}
	}

	d := diagnostic.NewError().WithLabelsIter(labels).WithNotesIter(slices.Values([]string{"a note"}))
	fmt.Println(len(d.Labels), d.Labels[1].Range, d.Notes)
	// Output: 2 {6 11} [a note]
}
