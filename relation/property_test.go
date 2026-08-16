package relation

import (
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/panictest"
	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"
)

func genYears() *rapid.Generator[[]date.Year] {
	return rapid.Custom(func(t *rapid.T) []date.Year {
		raw := rapid.SliceOfNDistinct(
			rapid.IntRange(2018, 2090),
			0, 20,
			func(y int) int { return y },
		).Draw(t, "years")

		years := make([]date.Year, 0, len(raw))
		for _, y := range raw {
			years = append(years, date.Year(y))
		}
		slices.Sort(years)
		return years
	})
}

func tableOf(years []date.Year) Table[int] {
	rows := make([]Row[int], 0, len(years))
	for i, y := range years {
		rows = append(rows, Row[int]{Year: y, Value: i})
	}
	return New(rows)
}

func TestNewShouldAlwaysProduceSortedDistinctYears(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		years := genYears().Draw(t, "years")

		got := tableOf(years).Years()

		if !slices.IsSorted(got) {
			t.Fatalf("Years() is not sorted: %v", got)
		}
		for i := 1; i < len(got); i++ {
			if got[i] == got[i-1] {
				t.Fatalf("Years() repeats a year: %v", got)
			}
		}
	})
}

func TestJoinShouldPreserveTheYearSet(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		years := genYears().Draw(t, "years")
		a, b := tableOf(years), tableOf(years)

		got := Join(a, b, func(y date.Year, av, bv int) int { return av + bv })

		if diff := cmp.Diff(a.Years(), got.Years()); diff != "" {
			t.Fatalf("Join changed the year set (-want +got):\n%s", diff)
		}
	})
}

func TestJoinShouldNeverPanicOnEqualYearSets(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		years := genYears().Draw(t, "years")
		a, b := tableOf(years), tableOf(years)

		if msg, refused := panictest.Message(func() {
			Join(a, b, func(y date.Year, av, bv int) int { return av + bv })
		}); refused {
			t.Fatalf("Join panicked although both tables cover %v: %s", years, msg)
		}
	})
}

func TestJoinShouldPanicWheneverTheYearSetsDiffer(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ya := genYears().Draw(t, "a")
		yb := genYears().Draw(t, "b")
		if slices.Equal(ya, yb) {
			t.Skip("the year sets happen to be equal")
		}
		a, b := tableOf(ya), tableOf(yb)

		_, refused := panictest.Message(func() {
			Join(a, b, func(y date.Year, av, bv int) int { return av + bv })
		})

		if !refused {
			t.Fatalf("Join accepted differing year sets %v and %v", ya, yb)
		}
	})
}

func TestLeftJoinShouldKeepExactlyTheLeftYears(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		left := genYears().Draw(t, "left")
		var right []date.Year
		for _, y := range left {
			if rapid.Bool().Draw(t, "keep") {
				right = append(right, y)
			}
		}

		got := LeftJoin(tableOf(left), tableOf(right), -1, func(y date.Year, l, r int) int { return r })

		if diff := cmp.Diff(tableOf(left).Years(), got.Years()); diff != "" {
			t.Fatalf("LeftJoin changed the year set (-want +got):\n%s", diff)
		}
	})
}

func TestLeftJoinShouldStandInForEveryYearTheRightTableLacks(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		left := genYears().Draw(t, "left")
		var right []date.Year
		for _, y := range left {
			if rapid.Bool().Draw(t, "keep") {
				right = append(right, y)
			}
		}
		const standIn = -1

		got := LeftJoin(tableOf(left), tableOf(right), standIn, func(y date.Year, l, r int) int { return r })

		for _, row := range got.Rows() {
			present := slices.Contains(right, row.Year)
			if !present && row.Value != standIn {
				t.Fatalf("year %d is absent on the right but the stand-in was not used: got %d", row.Year, row.Value)
			}
			if present && row.Value == standIn {
				t.Fatalf("year %d is present on the right but the stand-in was used", row.Year)
			}
		}
	})
}
