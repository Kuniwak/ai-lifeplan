package relation

import (
	"github.com/Kuniwak/lifeplan/date"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type income struct {
	Salary Yen
	Bonus  Yen
}

type Yen int64

func TestMapShouldComputeANewValueForEveryYear(t *testing.T) {
	table := New([]Row[income]{
		{2031, income{Salary: 9_000_000, Bonus: 1_000_000}},
		{2032, income{Salary: 9_500_000, Bonus: 1_100_000}},
	})

	got := Map(table, func(y date.Year, in income) Yen { return in.Salary + in.Bonus })

	want := []Row[Yen]{{2031, 10_000_000}, {2032, 10_600_000}}
	if diff := cmp.Diff(want, got.Rows()); diff != "" {
		t.Errorf("Map mismatch (-want +got):\n%s", diff)
	}
}

func TestMapShouldSelectASubsetOfTheColumns(t *testing.T) {
	table := New([]Row[income]{{2031, income{Salary: 9_000_000, Bonus: 1_000_000}}})

	got := Map(table, func(y date.Year, in income) Yen { return in.Salary })

	want := []Row[Yen]{{2031, 9_000_000}}
	if diff := cmp.Diff(want, got.Rows()); diff != "" {
		t.Errorf("Map mismatch (-want +got):\n%s", diff)
	}
}

func TestMapShouldKeepTheYearSet(t *testing.T) {
	table := New([]Row[int]{{2031, 1}, {2032, 2}, {2033, 3}})

	got := Map(table, func(y date.Year, v int) string { return "x" })

	if diff := cmp.Diff(table.Years(), got.Years()); diff != "" {
		t.Errorf("Map changed the year set (-want +got):\n%s", diff)
	}
}

func TestMapShouldPassTheYear(t *testing.T) {
	table := New([]Row[int]{{2031, 0}, {2032, 0}})

	got := Map(table, func(y date.Year, v int) date.Year { return y })

	want := []Row[date.Year]{{2031, 2031}, {2032, 2032}}
	if diff := cmp.Diff(want, got.Rows()); diff != "" {
		t.Errorf("Map mismatch (-want +got):\n%s", diff)
	}
}

func TestMapOnAnEmptyTable(t *testing.T) {
	var table Table[int]

	got := Map(table, func(y date.Year, v int) int { return v })

	if got.Len() != 0 {
		t.Errorf("Len() = %d, want 0", got.Len())
	}
}

func TestAggregate(t *testing.T) {
	type testCase struct {
		Table    Table[Yen]
		Expected Yen
	}

	testCases := map[string]testCase{
		"lifetime total (representative)": {
			Table:    New([]Row[Yen]{{2031, 100}, {2032, 200}, {2033, 300}}),
			Expected: 600,
		},
		"a single year (boundary value)": {
			Table:    New([]Row[Yen]{{2031, 100}}),
			Expected: 100,
		},
		"no years at all (boundary value)": {
			Table:    Table[Yen]{},
			Expected: 0,
		},
		"outgoings cancel income": {
			Table:    New([]Row[Yen]{{2031, 100}, {2032, -100}}),
			Expected: 0,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := Aggregate(tc.Table, Yen(0), func(acc Yen, y date.Year, v Yen) Yen { return acc + v })

			if diff := cmp.Diff(tc.Expected, got); diff != "" {
				t.Errorf("Aggregate mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAggregateShouldVisitYearsInOrder(t *testing.T) {
	table := New([]Row[int]{{2033, 3}, {2031, 1}, {2032, 2}})

	got := Aggregate(table, []date.Year(nil), func(acc []date.Year, y date.Year, v int) []date.Year { return append(acc, y) })

	want := []date.Year{2031, 2032, 2033}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Aggregate visited the years out of order (-want +got):\n%s", diff)
	}
}

func TestAggregateShouldFindTheFirstYearAssetsRunOut(t *testing.T) {
	assets := New([]Row[Yen]{{2031, 500}, {2032, 100}, {2033, -50}, {2034, -200}})

	got := Aggregate(assets, date.Year(0), func(firstNegative date.Year, y date.Year, v Yen) date.Year {
		if firstNegative != 0 || v >= 0 {
			return firstNegative
		}
		return y
	})

	if got != 2033 {
		t.Errorf("want 2033, got %d", got)
	}
}
