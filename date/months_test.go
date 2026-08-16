package date_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
)

func TestMonthsOfYearShouldSayWhichMonthsAndNotHowMany(t *testing.T) {
	for _, tc := range []struct {
		name         string
		year         date.Year
		from, though date.Date
		want         []int
	}{
		{
			name: "年をまたいで始まり、年内で終わる",
			year: 2064,
			from: date.Date{Year: 2000, Month: 5, Day: 1}, though: date.Date{Year: 2064, Month: 10, Day: 31},
			want: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			name: "年内で始まり、年をまたいで終わる",
			year: 2064,
			from: date.Date{Year: 2064, Month: 11, Day: 1}, though: date.Date{Year: 2090, Month: 12, Day: 31},
			want: []int{11, 12},
		},
		{
			name: "その年に掛からない",
			year: 2064,
			from: date.Date{Year: 2065, Month: 1, Day: 1}, though: date.Date{Year: 2090, Month: 12, Day: 31},
			want: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := date.MonthsOfYearIn(tc.year, tc.from, tc.though)
			if got.Count() != len(tc.want) {
				t.Errorf("%v は %d か月、%d か月のはず", got, got.Count(), len(tc.want))
			}
			for month := 1; month <= date.MonthsAYear; month++ {
				want := false
				for _, m := range tc.want {
					want = want || m == month
				}
				if got.Has(month) != want {
					t.Errorf("%d 月が %v、%v のはず", month, got.Has(month), want)
				}
			}
		})
	}
}

func TestTheUnionOfAPrefixAndASuffixShouldBeTheWholeYear(t *testing.T) {
	turnsSixtyFive := date.MonthsOfYearIn(2058, date.Date{Year: 2000, Month: 1, Day: 1},
		date.Date{Year: 2058, Month: 2, Day: 28})
	turnsForty := date.MonthsOfYearIn(2058, date.Date{Year: 2058, Month: 3, Day: 1},
		date.Date{Year: 2090, Month: 12, Day: 31})

	if got := turnsSixtyFive.Count(); got != 2 {
		t.Errorf("65 歳になる人は %d か月、2 か月のはず", got)
	}
	if got := turnsForty.Count(); got != 10 {
		t.Errorf("40 歳になる人は %d か月、10 か月のはず", got)
	}
	if got := max(turnsSixtyFive.Count(), turnsForty.Count()); got != 10 {
		t.Errorf("いちばん長い人は %d か月、10 か月のはず（これが の誤りである）", got)
	}
	if got := turnsSixtyFive.Union(turnsForty).Count(); got != date.MonthsAYear {
		t.Errorf("世帯は %d か月、12 か月のはず", got)
	}
}

func TestTheOverlapOfTwoWindowsShouldBeTheirIntersection(t *testing.T) {
	firstHalf := date.MonthsOfYearIn(2064, date.Date{Year: 2000, Month: 1, Day: 1},
		date.Date{Year: 2064, Month: 6, Day: 30})
	secondHalf := date.MonthsOfYearIn(2064, date.Date{Year: 2064, Month: 7, Day: 1},
		date.Date{Year: 2090, Month: 12, Day: 31})

	if got := min(firstHalf.Count(), secondHalf.Count()); got != 6 {
		t.Errorf("min は %d か月（これが の誤りである）", got)
	}
	if got := firstHalf.Intersect(secondHalf).Count(); got != 0 {
		t.Errorf("交わりは %d か月、重なっていないので 0 のはず", got)
	}
}
