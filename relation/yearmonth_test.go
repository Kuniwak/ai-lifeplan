package relation

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
)

func TestMonthsSinceAndYearMonthOfShouldBeInverses(t *testing.T) {
	for year := date.Year(0); year <= 2100; year++ {
		for month := 1; month <= date.MonthsAYear; month++ {
			gotYear, gotMonth := YearMonthOf(MonthsSince(year, month))
			if gotYear != year || gotMonth != month {
				t.Fatalf("%d年%d月 -> %d -> %d年%d月", year, month, MonthsSince(year, month), gotYear, gotMonth)
			}
		}
	}
}

func TestMonthsSinceShouldRise(t *testing.T) {
	previous := MonthsSince(0, 1) - 1
	for year := date.Year(0); year <= 2100; year++ {
		for month := 1; month <= date.MonthsAYear; month++ {
			got := MonthsSince(year, month)
			if got != previous+1 {
				t.Fatalf("%d年%d月 は %d で、一つ前の %d の次ではない", year, month, got, previous)
			}
			previous = got
		}
	}
}

func TestAMonthOutsideTheYearShouldRoll(t *testing.T) {
	cases := map[string]struct {
		Year  date.Year
		Month int
		Want  int
	}{
		"13 月は翌年の 1 月":  {Year: 2020, Month: 13, Want: MonthsSince(2021, 1)},
		"0 月は前年の 12 月":  {Year: 2020, Month: 0, Want: MonthsSince(2019, 12)},
		"24 月は翌年の 12 月": {Year: 2020, Month: 24, Want: MonthsSince(2021, 12)},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {

			got := MonthsSince(c.Year, c.Month)

			if got != c.Want {
				t.Errorf("MonthsSince(%d, %d) = %d, want %d", c.Year, c.Month, got, c.Want)
			}
		})
	}
}
