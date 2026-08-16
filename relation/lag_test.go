package relation

import (
	"github.com/Kuniwak/lifeplan/date"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLagShouldTakeThePreviousYearsValue(t *testing.T) {
	years := New([]Row[int]{{2031, 0}, {2032, 0}, {2033, 0}})
	income := New([]Row[int]{{2031, 1000}, {2032, 1100}, {2033, 1200}})

	got := Lag(years, income, -1, func(y date.Year, _ int, lastYear int) int { return lastYear })

	want := []Row[int]{
		{2031, -1},
		{2032, 1000},
		{2033, 1100},
	}
	if diff := cmp.Diff(want, got.Rows()); diff != "" {
		t.Errorf("Lag mismatch (-want +got):\n%s", diff)
	}
}

func TestLagShouldKeepTheYearsOfTheLeftTable(t *testing.T) {
	type testCase struct {
		Left      Table[int]
		Right     Table[int]
		WantYears []date.Year
	}

	testCases := map[string]testCase{
		"the right table reaches further back": {
			Left:      New([]Row[int]{{2032, 0}, {2033, 0}}),
			Right:     New([]Row[int]{{2029, 1}, {2030, 2}, {2031, 3}, {2032, 4}}),
			WantYears: []date.Year{2032, 2033},
		},
		"the right table is empty": {
			Left:      New([]Row[int]{{2031, 0}}),
			Right:     Table[int]{},
			WantYears: []date.Year{2031},
		},
		"a gap in the left table": {
			Left:      New([]Row[int]{{2031, 0}, {2040, 0}}),
			Right:     New([]Row[int]{{2030, 1}, {2039, 2}}),
			WantYears: []date.Year{2031, 2040},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := Lag(tc.Left, tc.Right, 0, func(y date.Year, l, prev int) int { return prev })

			if diff := cmp.Diff(tc.WantYears, got.Years()); diff != "" {
				t.Errorf("Lag changed the year set (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLagShouldUseTheGivenValueWhenThePreviousYearIsAbsent(t *testing.T) {
	left := New([]Row[int]{{2031, 0}, {2032, 0}, {2033, 0}})
	right := New([]Row[int]{{2032, 500}})
	const standIn = -1

	got := Lag(left, right, standIn, func(y date.Year, l, prev int) int { return prev })

	want := []Row[int]{
		{2031, standIn},
		{2032, standIn},
		{2033, 500},
	}
	if diff := cmp.Diff(want, got.Rows()); diff != "" {
		t.Errorf("Lag mismatch (-want +got):\n%s", diff)
	}
}

func TestLagShouldPassTheCurrentYearToTheCombiner(t *testing.T) {
	left := New([]Row[int]{{2032, 7}})
	right := New([]Row[int]{{2031, 500}})

	got := Lag(left, right, 0, func(y date.Year, l, prev int) []int { return []int{int(y), l, prev} })

	want := []Row[[]int]{{2032, []int{2032, 7, 500}}}
	if diff := cmp.Diff(want, got.Rows()); diff != "" {
		t.Errorf("Lag mismatch (-want +got):\n%s", diff)
	}
}
