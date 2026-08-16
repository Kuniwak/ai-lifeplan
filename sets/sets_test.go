package sets

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDifferenceShouldReturnElementsNotPresentInTheOther(t *testing.T) {
	type testCase struct {
		A        []string
		B        []string
		Expected []string
	}

	testCases := map[string]testCase{
		"some missing (representative value)": {
			A:        []string{"a", "b", "c"},
			B:        []string{"b"},
			Expected: []string{"a", "c"},
		},
		"none missing (boundary value)": {
			A:        []string{"a", "b"},
			B:        []string{"a", "b"},
			Expected: nil,
		},
		"all missing (boundary value)": {
			A:        []string{"a", "b"},
			B:        nil,
			Expected: []string{"a", "b"},
		},
		"empty a (boundary value)": {
			A:        nil,
			B:        []string{"a"},
			Expected: nil,
		},
		"b has extra elements (representative value)": {
			A:        []string{"a"},
			B:        []string{"a", "b", "c"},
			Expected: nil,
		},
		"duplicates in a are preserved (representative value)": {
			A:        []string{"a", "a", "b"},
			B:        []string{"b"},
			Expected: []string{"a", "a"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := Difference(tc.A, tc.B)

			if diff := cmp.Diff(tc.Expected, got); diff != "" {
				t.Errorf("Difference mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDifferenceShouldPreserveOrderOfTheFirstArgument(t *testing.T) {
	a := []int{3, 1, 2}
	b := []int{1}

	got := Difference(a, b)

	want := []int{3, 2}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Difference mismatch (-want +got):\n%s", diff)
	}
}
