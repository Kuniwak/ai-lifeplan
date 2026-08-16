package date

import "testing"

func TestParse(t *testing.T) {
	for name, tc := range map[string]struct {
		field string
		want  Date
		bad   bool
	}{
		"ふつうの日付":      {field: "2020-04-02", want: Date{Year: 2020, Month: 4, Day: 2}},
		"年末の日付":       {field: "2023-12-18", want: Date{Year: 2023, Month: 12, Day: 18}},
		"月の初日":        {field: "2000-03-01", want: Date{Year: 2000, Month: 3, Day: 1}},
		"年だけは受け付けない":  {field: "2022", bad: true},
		"年月だけは受け付けない": {field: "2022-01", bad: true},
		"空欄は受け付けない":   {field: "", bad: true},

		"暦に無い日は New が断る": {field: "2022-02-30", bad: true},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := Parse(tc.field)
			if tc.bad {
				if err == nil {
					t.Fatalf("Parse(%q) = %v, want an error", tc.field, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): %v", tc.field, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q) = %v, want %v", tc.field, got, tc.want)
			}
		})
	}
}

func TestReachesAge(t *testing.T) {
	for name, tc := range map[string]struct {
		born Date
		age  int
		want Date
	}{
		"月の半ばに生まれた人が18歳":      {born: Date{2020, 1, 15}, age: 18, want: Date{2038, 1, 14}},
		"12月生まれが18歳":          {born: Date{2022, 12, 5}, age: 18, want: Date{2040, 12, 4}},
		"月の半ばに生まれた人が3歳":       {born: Date{2020, 1, 15}, age: 3, want: Date{2023, 1, 14}},
		"4月1日生まれは3月31日に達する":   {born: Date{2020, 4, 1}, age: 18, want: Date{2038, 3, 31}},
		"4月2日生まれは4月1日に達する":    {born: Date{2020, 4, 2}, age: 18, want: Date{2038, 4, 1}},
		"1月1日生まれは前年の大晦日に達する":  {born: Date{2020, 1, 1}, age: 3, want: Date{2022, 12, 31}},
		"うるう日生まれは平年の2月28日":    {born: Date{2024, 2, 29}, age: 3, want: Date{2027, 2, 28}},
		"うるう日生まれはうるう年なら2月28日": {born: Date{2024, 2, 29}, age: 4, want: Date{2028, 2, 28}},
	} {
		t.Run(name, func(t *testing.T) {
			if got := tc.born.ReachesAge(tc.age); got != tc.want {
				t.Errorf("%v.ReachesAge(%d) = %v, want %v", tc.born, tc.age, got, tc.want)
			}
		})
	}
}

func TestMonthsOfYearIn(t *testing.T) {
	far := Date{2999, 12, 31}
	for name, tc := range map[string]struct {
		year          int
		from, through Date
		want          int
	}{
		"前の年に始まっていれば通年":    {year: 2044, from: Date{2020, 5, 3}, through: far, want: 12},
		"1 月に始まれば 12 か月":   {year: 2044, from: Date{2044, 1, 1}, through: far, want: 12},
		"12 月に始まれば 1 か月":   {year: 2044, from: Date{2044, 12, 17}, through: far, want: 1},
		"前年末に始まっても通年":      {year: 2042, from: Date{2041, 12, 31}, through: far, want: 12},
		"翌年に始まれば 0 か月":     {year: 2044, from: Date{2045, 1, 1}, through: far, want: 0},
		"2 月に終われば 2 か月":    {year: 2053, from: Date{2013, 1, 1}, through: Date{2053, 2, 28}, want: 2},
		"前の年に終わっていれば 0 か月": {year: 2054, from: Date{2013, 1, 1}, through: Date{2053, 2, 28}, want: 0},
		"同じ年に始まって終わる":      {year: 2044, from: Date{2044, 3, 1}, through: Date{2044, 5, 31}, want: 3},
		"終わりが始まりより前なら 0":   {year: 2044, from: Date{2044, 6, 1}, through: Date{2044, 3, 1}, want: 0},
	} {
		t.Run(name, func(t *testing.T) {
			if got := MonthsOfYearIn(Year(tc.year), tc.from, tc.through).Count(); got != tc.want {
				t.Errorf("MonthsOfYearIn(%d, %v, %v) = %d, want %d", tc.year, tc.from, tc.through, got, tc.want)
			}
		})
	}
}

func TestSchoolYearEndShouldBeTheFirstMarch31OnOrAfter(t *testing.T) {
	for _, c := range []struct {
		in, want Date
	}{
		{Date{Year: 2040, Month: 1, Day: 1}, Date{Year: 2040, Month: 3, Day: 31}},
		{Date{Year: 2040, Month: 3, Day: 30}, Date{Year: 2040, Month: 3, Day: 31}},
		{Date{Year: 2040, Month: 3, Day: 31}, Date{Year: 2040, Month: 3, Day: 31}},
		{Date{Year: 2040, Month: 4, Day: 1}, Date{Year: 2041, Month: 3, Day: 31}},
		{Date{Year: 2040, Month: 12, Day: 31}, Date{Year: 2041, Month: 3, Day: 31}},
	} {
		t.Run(c.in.String(), func(t *testing.T) {
			if got := c.in.SchoolYearEnd(); got != c.want {
				t.Errorf("%s.SchoolYearEnd() = %s, want %s", c.in, got, c.want)
			}
		})
	}
}

func TestNewShouldRefuseADayTheCalendarHasNot(t *testing.T) {
	type testCase struct {
		Year        Year
		Month, Day  int
		WantRefused bool
	}

	testCases := map[string]testCase{
		"ふつうの日":              {Year: 2022, Month: 2, Day: 28},
		"閏年の 2 月 29 日":       {Year: 2024, Month: 2, Day: 29},
		"閏年でない 2 月 29 日":     {Year: 2022, Month: 2, Day: 29, WantRefused: true},
		"2 月 30 日":           {Year: 2022, Month: 2, Day: 30, WantRefused: true},
		"月の下（境界値）":           {Year: 2022, Month: 0, Day: 1, WantRefused: true},
		"月の上（境界値）":           {Year: 2022, Month: 13, Day: 1, WantRefused: true},
		"日の下（境界値）":           {Year: 2022, Month: 1, Day: 0, WantRefused: true},
		"31 日ある月の 31 日（境界値）": {Year: 2022, Month: 1, Day: 31},
		"30 日しかない月の 31 日":    {Year: 2022, Month: 4, Day: 31, WantRefused: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := New(tc.Year, tc.Month, tc.Day)

			switch {
			case tc.WantRefused && err == nil:
				t.Fatalf("断るはずが %v になった", got)
			case !tc.WantRefused && err != nil:
				t.Fatalf("通るはずが %v", err)
			case !tc.WantRefused && got != (Date{Year: tc.Year, Month: tc.Month, Day: tc.Day}):
				t.Errorf("%v が返ってきた", got)
			}
		})
	}
}
