package law

import (
	"fmt"
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

func monthsThrough(first, last int) date.Months {
	var months date.Months
	for month := first; month <= last; month++ {
		months = months.Union(date.MonthsOfYearIn(0,
			date.Date{Year: 0, Month: month, Day: 1}, date.Date{Year: 0, Month: month, Day: 1}))
	}
	return months
}

func TestCoverMonthsInShouldSplitTheYearSomebodyTurnsSeventyFive(t *testing.T) {
	june := date.Date{Year: 1990, Month: 6, Day: 15}

	for _, tc := range []struct {
		name      string
		year      date.Year
		born      date.Date
		otherwise Cover
		want      []CoverMonths
	}{
		{
			name:      "74 歳までは丸ごとそれまでの制度",
			year:      2064,
			born:      june,
			otherwise: NationalHealthInsurance,
			want:      []CoverMonths{{Cover: NationalHealthInsurance, Months: date.WholeYear}},
		},
		{
			name:      "75 歳になる年は 5 か月と 7 か月に割れる",
			year:      2065,
			born:      june,
			otherwise: NationalHealthInsurance,
			want: []CoverMonths{
				{Cover: NationalHealthInsurance, Months: monthsThrough(1, 5)},
				{Cover: LateElderlyHealthCare, Months: monthsThrough(6, 12)},
			},
		},
		{
			name:      "翌年からは丸ごと後期高齢者医療",
			year:      2066,
			born:      june,
			otherwise: NationalHealthInsurance,
			want:      []CoverMonths{{Cover: LateElderlyHealthCare, Months: date.WholeYear}},
		},
		{
			name:      "1 月生まれは 75 歳の年が丸ごと後期高齢者医療",
			year:      2065,
			born:      date.Date{Year: 1990, Month: 1, Day: 5},
			otherwise: NationalHealthInsurance,
			want:      []CoverMonths{{Cover: LateElderlyHealthCare, Months: date.WholeYear}},
		},
		{
			name:      "7 月生まれは半分ずつに割れる",
			year:      2065,
			born:      date.Date{Year: 1990, Month: 7, Day: 1},
			otherwise: EmployeesHealthInsurance,
			want: []CoverMonths{
				{Cover: EmployeesHealthInsurance, Months: monthsThrough(1, 6)},
				{Cover: LateElderlyHealthCare, Months: monthsThrough(7, 12)},
			},
		},
		{
			name:      "世帯の外の人には月が無い",
			year:      2065,
			born:      june,
			otherwise: NoCover,
			want:      nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := CoverMonthsIn(tc.year, tc.born, tc.otherwise)
			if !slices.Equal(got, tc.want) {
				t.Errorf("CoverMonthsIn(%d, %s, %q) = %v, want %v",
					tc.year, tc.born, tc.otherwise, got, tc.want)
			}
		})
	}
}

func TestLongestCoverShouldTakeTheFirstOfATie(t *testing.T) {
	for _, tc := range []struct {
		name   string
		months []CoverMonths
		want   Cover
	}{
		{name: "何も無ければ NoCover", months: nil, want: NoCover},
		{
			name:   "長いほう",
			months: []CoverMonths{{Cover: NationalHealthInsurance, Months: monthsThrough(1, 10)}, {Cover: LateElderlyHealthCare, Months: 2}},
			want:   NationalHealthInsurance,
		},
		{
			name:   "半々なら先に来たほう",
			months: []CoverMonths{{Cover: EmployeesHealthInsurance, Months: monthsThrough(1, 6)}, {Cover: LateElderlyHealthCare, Months: 6}},
			want:   EmployeesHealthInsurance,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := LongestCover(tc.months); got != tc.want {
				t.Errorf("LongestCover(%v) = %q, want %q", tc.months, got, tc.want)
			}
		})
	}
}

func TestEmployeeCoverFromShouldTurnOnTheHoursAndNotThePay(t *testing.T) {
	const notAStudent = false
	at := Workplace{NormalWeeklyHours: 40, Specified: true}

	if got := EmployeeCoverFrom(at, 0, 0, notAStudent); got != NationalHealthInsurance {
		t.Errorf("週 0 時間が %q。%q のはず", got, NationalHealthInsurance)
	}
	if got := EmployeeCoverFrom(at, 40, 400_000, notAStudent); got != EmployeesHealthInsurance {
		t.Errorf("週 40 時間が %q。%q のはず", got, EmployeesHealthInsurance)
	}
	if got := EmployeeCoverFrom(at, 19, 500_000, notAStudent); got != NationalHealthInsurance {
		t.Errorf("週 19 時間・報酬 50 万円が %q。%q のはず", got, NationalHealthInsurance)
	}
}

func TestEmployeeCoverInShouldTakeTheSchemeThatHoldsMostOfTheYear(t *testing.T) {
	born := date.Date{Year: 1990, Month: 6, Day: 15}
	const (
		hours       = 40
		monthly     = money.Yen(400_000)
		notAStudent = false
	)
	at := Workplace{NormalWeeklyHours: 40, Specified: true}

	for _, tc := range []struct {
		year date.Year
		want Cover
	}{
		{year: 2064, want: EmployeesHealthInsurance},
		{year: 2065, want: LateElderlyHealthCare},
	} {
		t.Run(fmt.Sprint(tc.year), func(t *testing.T) {
			if got := EmployeeCoverIn(tc.year, born, at, hours, monthly, notAStudent); got != tc.want {
				t.Errorf("EmployeeCoverIn(%d, %s, 週 %d 時間) = %q, want %q", tc.year, born, hours, got, tc.want)
			}
		})
	}
}
