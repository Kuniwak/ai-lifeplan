package law

import (
	"os"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

const beforeTheReform = ChildAllowanceReformedFrom - 2

func childAllowanceTable(t *testing.T) ChildAllowanceTable {
	t.Helper()

	parsed := MustLoadChildAllowanceLimits(t, os.DirFS("../"+LawDirectory))
	return parsed
}

func TestTheLimitStepShouldMatchTheTable(t *testing.T) {
	loaded := childAllowanceTable(t)

	for dependents := 1; dependents < len(loaded.limits); dependents++ {
		previous := loaded.limits[dependents-1]
		current := loaded.limits[dependents]

		if got := current.IncomeLimit - previous.IncomeLimit; got != childAllowanceLimitStep {
			t.Errorf("扶養親族等 %d 人で所得制限限度額が %d 増える。1 人あたりの加算 %d と違う", dependents, got, childAllowanceLimitStep)
		}
		if got := current.IncomeCeiling - previous.IncomeCeiling; got != childAllowanceLimitStep {
			t.Errorf("扶養親族等 %d 人で所得上限限度額が %d 増える。1 人あたりの加算 %d と違う", dependents, got, childAllowanceLimitStep)
		}
	}
}

func TestChildAllowanceLimitsFor(t *testing.T) {
	type testCase struct {
		Dependents int
		Expected   ChildAllowanceLimits
	}

	testCases := map[string]testCase{
		"no dependents":  {Dependents: 0, Expected: ChildAllowanceLimits{IncomeLimit: 6_220_000, IncomeCeiling: 8_580_000}},
		"one dependent":  {Dependents: 1, Expected: ChildAllowanceLimits{IncomeLimit: 6_600_000, IncomeCeiling: 8_960_000}},
		"two dependents": {Dependents: 2, Expected: ChildAllowanceLimits{IncomeLimit: 6_980_000, IncomeCeiling: 9_340_000}},
		"five dependents (the last row of the table)":   {Dependents: 5, Expected: ChildAllowanceLimits{IncomeLimit: 8_120_000, IncomeCeiling: 10_480_000}},
		"six dependents (past the table, one more 38万)": {Dependents: 6, Expected: ChildAllowanceLimits{IncomeLimit: 8_500_000, IncomeCeiling: 10_860_000}},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got, ok := childAllowanceTable(t).Limits(beforeTheReform, tc.Dependents)
			if !ok {
				t.Fatalf("%d 年には限度額があるはずだ", beforeTheReform)
			}

			if got != tc.Expected {
				t.Errorf("児童手当の限度額: 扶養親族等 %d 人は %+v のはずだが %+v になった", tc.Dependents, tc.Expected, got)
			}
		})
	}
}

func TestChildAllowanceMonthly(t *testing.T) {
	type testCase struct {
		Age          int
		ThirdOrLater bool
		Income       money.Yen
		Dependents   int
		Expected     money.Yen
	}

	testCases := map[string]testCase{
		"under three":                                                {Age: 0, Income: 5_000_000, Dependents: 0, Expected: 15_000},
		"the last year under three":                                  {Age: 2, Income: 5_000_000, Dependents: 0, Expected: 15_000},
		"at primary school":                                          {Age: 3, Income: 5_000_000, Dependents: 0, Expected: 10_000},
		"the third child at primary school":                          {Age: 8, ThirdOrLater: true, Income: 5_000_000, Dependents: 0, Expected: 15_000},
		"the last year at primary school":                            {Age: 12, Income: 5_000_000, Dependents: 0, Expected: 10_000},
		"at lower secondary school":                                  {Age: 13, Income: 5_000_000, Dependents: 0, Expected: 10_000},
		"the third child no longer counts at lower secondary school": {Age: 13, ThirdOrLater: true, Income: 5_000_000, Dependents: 0, Expected: 10_000},
		"the last year that is paid at all":                          {Age: 15, Income: 5_000_000, Dependents: 0, Expected: 10_000},
		"too old (boundary)":                                         {Age: 16, Income: 5_000_000, Dependents: 0, Expected: 0},

		"one yen below the limit (boundary)": {Age: 0, Income: 6_219_999, Dependents: 0, Expected: 15_000},
		"exactly at the limit (boundary)":    {Age: 0, Income: 6_220_000, Dependents: 0, Expected: 5_000},
		"between the limit and the ceiling":  {Age: 0, Income: 7_500_000, Dependents: 0, Expected: 5_000},

		"one yen below the ceiling (boundary)": {Age: 0, Income: 8_579_999, Dependents: 0, Expected: 5_000},
		"exactly at the ceiling (boundary)":    {Age: 0, Income: 8_580_000, Dependents: 0, Expected: 0},
		"well past the ceiling":                {Age: 0, Income: 12_000_000, Dependents: 1, Expected: 0},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := childAllowanceTable(t).MonthlyBeforeReform(tc.Age, tc.ThirdOrLater, tc.Income, tc.Dependents)

			if got != tc.Expected {
				t.Errorf("児童手当(%d 歳, 第3子以降 %v, 所得 %d, 扶養親族等 %d 人) = %d, want %d",
					tc.Age, tc.ThirdOrLater, tc.Income, tc.Dependents, got, tc.Expected)
			}
		})
	}
}

func TestChildAllowanceShouldNeverRiseWithIncome(t *testing.T) {
	loaded := childAllowanceTable(t)

	rapid.Check(t, func(t *rapid.T) {
		year := rapid.IntRange(2018, 2090).Draw(t, "year")
		age := rapid.IntRange(0, 25).Draw(t, "age")
		thirdOrLater := rapid.Bool().Draw(t, "thirdOrLater")
		dependents := rapid.IntRange(0, 10).Draw(t, "dependents")
		poorer := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "poorer"))
		richer := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "richer"))
		if poorer > richer {
			poorer, richer = richer, poorer
		}

		more := loaded.MonthlyBeforeReform(age, thirdOrLater, poorer, dependents)
		less := loaded.MonthlyBeforeReform(age, thirdOrLater, richer, dependents)

		if less > more {
			t.Fatalf("%d 年: income %d gets %d but income %d gets %d, which is more", year, poorer, more, richer, less)
		}
		if more > ChildAllowanceThirdOrLater || less < 0 {
			t.Fatalf("%d 年: allowance out of range: %d and %d", year, more, less)
		}
	})
}

func TestChildAllowanceMonthlyAfterReform(t *testing.T) {
	for name, tc := range map[string]struct {
		underThree   bool
		thirdOrLater bool
		want         money.Yen
	}{
		"3歳未満": {underThree: true, want: 15_000},
		"3歳以上": {want: 10_000},
		"第3子以降は3歳未満でも3万円": {underThree: true, thirdOrLater: true, want: 30_000},
		"第3子以降は3歳以上でも3万円": {thirdOrLater: true, want: 30_000},
	} {
		t.Run(name, func(t *testing.T) {

			got := ChildAllowanceMonthlyAfterReform(tc.underThree, tc.thirdOrLater)

			if got != tc.want {
				t.Errorf("ChildAllowanceMonthlyAfterReform(%v, %v) = %d, want %d",
					tc.underThree, tc.thirdOrLater, got, tc.want)
			}
		})
	}
}

func TestTheReformShouldSwitchAtTheStatedYear(t *testing.T) {
	loaded := childAllowanceTable(t)
	kid := date.Date{Year: 2020, Month: 1, Day: 15}

	const wellPastTheOldCeiling money.Yen = 20_000_000

	if got := loaded.Yearly(beforeTheReform, kid, false, wellPastTheOldCeiling, 2); got != 0 {
		t.Errorf("%d 年は所得で 0 円のはずだが %d 円になった", beforeTheReform, got)
	}
	if got := loaded.Yearly(ChildAllowanceReformedFrom, kid, false, wellPastTheOldCeiling, 2); got == 0 {
		t.Errorf("%d 年は所得を見ないはずなのに 0 円になった", ChildAllowanceReformedFrom)
	}
}

func TestChildAllowanceMonthsIn(t *testing.T) {
	january := date.Date{Year: 2020, Month: 1, Day: 15}
	december := date.Date{Year: 2022, Month: 12, Day: 5}

	for name, tc := range map[string]struct {
		born date.Date
		year date.Year
		want ChildAllowanceMonths
	}{
		"1月生まれ 生まれた年は翌月から":         {born: january, year: 2020, want: ChildAllowanceMonths{UnderThree: 11}},
		"1月生まれ 2022 年は通年 3 歳未満":    {born: january, year: 2022, want: ChildAllowanceMonths{UnderThree: 12}},
		"1月生まれ 2023 年に段が下がる":       {born: january, year: 2023, want: ChildAllowanceMonths{UnderThree: 1, Older: 11}},
		"1月生まれ 2037 年は通年":          {born: january, year: 2037, want: ChildAllowanceMonths{Older: 12}},
		"1月生まれ 2038 年は年度末までの 3 か月": {born: january, year: 2038, want: ChildAllowanceMonths{Older: 3}},
		"1月生まれ 2039 年は無い":          {born: january, year: 2039, want: ChildAllowanceMonths{}},

		"12月生まれ 生まれた年は 1 か月も無い":  {born: december, year: 2022, want: ChildAllowanceMonths{}},
		"12月生まれ 2025 年は通年 3 歳未満": {born: december, year: 2025, want: ChildAllowanceMonths{UnderThree: 12}},
		"12月生まれ 2026 年から 3 歳以上":  {born: december, year: 2026, want: ChildAllowanceMonths{Older: 12}},
		"12月生まれ 2040 年は通年":       {born: december, year: 2040, want: ChildAllowanceMonths{Older: 12}},
		"12月生まれ 2041 年は 3 か月":    {born: december, year: 2041, want: ChildAllowanceMonths{Older: 3}},
		"12月生まれ 2042 年は無い":       {born: december, year: 2042, want: ChildAllowanceMonths{}},

		"3月1日生まれは3月まで3歳未満": {born: date.Date{Year: 2020, Month: 3, Day: 1}, year: 2023, want: ChildAllowanceMonths{UnderThree: 3, Older: 9}},
		"3月2日生まれも3月まで3歳未満": {born: date.Date{Year: 2020, Month: 3, Day: 2}, year: 2023, want: ChildAllowanceMonths{UnderThree: 3, Older: 9}},

		"4月1日生まれは 18 歳の年の 3 月まで": {born: date.Date{Year: 2020, Month: 4, Day: 1}, year: 2038, want: ChildAllowanceMonths{Older: 3}},
		"4月2日生まれはもう 1 年ある":       {born: date.Date{Year: 2020, Month: 4, Day: 2}, year: 2038, want: ChildAllowanceMonths{Older: 12}},
	} {
		t.Run(name, func(t *testing.T) {
			if got := ChildAllowanceMonthsIn(tc.year, tc.born); got != tc.want {
				t.Errorf("ChildAllowanceMonthsIn(%d, %v) = %+v, want %+v", tc.year, tc.born, got, tc.want)
			}
		})
	}
}

func TestChildAllowanceLimitsShouldNotExistAfterTheReform(t *testing.T) {
	loaded := childAllowanceTable(t)

	if _, ok := loaded.Limits(beforeTheReform, 2); !ok {
		t.Errorf("%d 年には限度額があるはずだ", beforeTheReform)
	}
	if limits, ok := loaded.Limits(ChildAllowanceReformedFrom, 2); ok {
		t.Errorf("%d 年に限度額 %+v を返している。第五条は削除されている",
			ChildAllowanceReformedFrom, limits)
	}
}

func TestTheThirdOrLaterCountShouldStopAtTheStatutesYearEnd(t *testing.T) {
	mustDate := func(t *testing.T, s string) date.Date {
		t.Helper()
		d, err := date.Parse(s)
		if err != nil {
			t.Fatalf("date.Parse(%q): %v", s, err)
		}
		return d
	}

	for name, c := range map[string]struct {
		born date.Date
		year date.Year
		want bool
	}{
		"拡充前・18歳年度末の年":   {born: mustDate(t, "2002-04-02"), year: 2021, want: true},
		"拡充前・その翌年":       {born: mustDate(t, "2002-04-02"), year: 2022, want: false},
		"拡充前・3月生まれは一年早い": {born: mustDate(t, "2002-03-31"), year: 2021, want: false},

		"拡充後・22歳年度末の年":   {born: mustDate(t, "2002-04-02"), year: 2025, want: true},
		"拡充後・その翌年":       {born: mustDate(t, "2002-04-02"), year: 2026, want: false},
		"拡充後は 18 歳超も数える": {born: mustDate(t, "2002-04-02"), year: 2025, want: true},

		"生まれた年は当然数える": {born: mustDate(t, "2020-04-02"), year: 2022, want: true},
	} {
		t.Run(name, func(t *testing.T) {
			if got := ChildAllowanceCountsTowardsThirdOrLater(c.year, c.born); got != c.want {
				t.Errorf("ChildAllowanceCountsTowardsThirdOrLater(%d, %s) = %v, want %v",
					c.year, c.born, got, c.want)
			}
		})
	}
}

func TestTheLimitsTableShouldStartWhereTheUpperLimitCameIn(t *testing.T) {
	table := MustLoadChildAllowanceLimits(t, os.DirFS("../"+LawDirectory))

	for name, tc := range map[string]struct {
		Year date.Year
		Want bool
	}{
		"所得上限限度額のできた年（境界値）": {Year: 2022, Want: true},
		"その 1 年前（境界値）":      {Year: 2021},
		"はるか前":              {Year: 1990},
		"撤廃の前年（境界値）":        {Year: ChildAllowanceReformedFrom - 1, Want: true},
		"撤廃の年（境界値）":         {Year: ChildAllowanceReformedFrom},
	} {
		t.Run(name, func(t *testing.T) {
			limits, tested := table.Limits(tc.Year, 0)

			if tested != tc.Want {
				t.Errorf("Limits(%d) tested=%v（%v のはず）limits=%+v", tc.Year, tested, tc.Want, limits)
			}
		})
	}
}
