package law

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
)

func TestNationalPensionCategoryOf(t *testing.T) {
	for name, tc := range map[string]struct {
		age             int
		cover           Cover
		insured, spouse bool
		want            NationalPensionCategory
	}{
		"被保険者本人が社保なら第2号":   {age: 40, cover: EmployeesHealthInsurance, insured: true, want: SecondCategoryInsured},
		"配偶者が健保の被扶養者なら第3号": {age: 40, cover: EmployeesHealthInsurance, spouse: true, want: ThirdCategoryInsured},
		"子が健保の被扶養者なら第1号":   {age: 20, cover: EmployeesHealthInsurance, want: FirstCategoryInsured},
		"国保なら第1号":          {age: 40, cover: NationalHealthInsurance, want: FirstCategoryInsured},

		"65 歳でも就労中なら第2号":  {age: 65, cover: EmployeesHealthInsurance, insured: true, want: SecondCategoryInsured},
		"65 歳の配偶者は第3号でない": {age: 65, cover: EmployeesHealthInsurance, spouse: true, want: NotNationalPensionInsured},

		"19 歳は被保険者でない（境界）": {age: 19, cover: NationalHealthInsurance, want: NotNationalPensionInsured},
		"60 歳は被保険者でない（境界）": {age: 60, cover: NationalHealthInsurance, want: NotNationalPensionInsured},
		"世帯外の人は被保険者でない":    {age: 30, cover: NoCover, want: NotNationalPensionInsured},
		"後期高齢者は被保険者でない":    {age: 76, cover: LateElderlyHealthCare, want: NotNationalPensionInsured},
	} {
		t.Run(name, func(t *testing.T) {

			got := NationalPensionCategoryOf(tc.age, tc.cover, tc.insured, tc.spouse)

			if got != tc.want {
				t.Errorf("NationalPensionCategoryOf(%d, %q, insured=%v, spouse=%v) = %v, want %v",
					tc.age, tc.cover, tc.insured, tc.spouse, got, tc.want)
			}
		})
	}
}

func TestNationalPensionMonthsIn(t *testing.T) {
	december := date.Date{Year: 2022, Month: 12, Day: 5}
	january := date.Date{Year: 2020, Month: 1, Day: 15}

	for name, tc := range map[string]struct {
		year            date.Year
		born            date.Date
		cover           Cover
		insured, spouse bool
		want            int
	}{
		"12月生まれが 20 歳になる年": {year: 2042, born: december, cover: EmployeesHealthInsurance, want: 1},
		"12月生まれの翌年は通年":     {year: 2043, born: december, cover: EmployeesHealthInsurance, want: 12},
		"12月生まれの前年は無い":     {year: 2041, born: december, cover: EmployeesHealthInsurance, want: 0},

		"1月生まれが 20 歳になる年は通年": {year: 2040, born: january, cover: EmployeesHealthInsurance, want: 12},

		"1月1日生まれは19歳の年に1か月": {year: 2039, born: date.Date{Year: 2020, Month: 1, Day: 1}, cover: NationalHealthInsurance, want: 1},
		"1月1日生まれの翌年は通年":     {year: 2040, born: date.Date{Year: 2020, Month: 1, Day: 1}, cover: NationalHealthInsurance, want: 12},

		"60 歳になる年は誕生月の前月まで": {year: 2052, born: date.Date{Year: 1992, Month: 9, Day: 20}, cover: NationalHealthInsurance, want: 8},
		"61 歳の年はもう無い":       {year: 2053, born: date.Date{Year: 1992, Month: 9, Day: 20}, cover: NationalHealthInsurance, want: 0},

		"第2号は納めない":   {year: 2043, born: december, cover: EmployeesHealthInsurance, insured: true, want: 0},
		"第3号は納めない":   {year: 2043, born: december, cover: EmployeesHealthInsurance, spouse: true, want: 0},
		"世帯外は納めない":   {year: 2043, born: december, cover: NoCover, want: 0},
		"後期高齢者は納めない": {year: 2043, born: december, cover: LateElderlyHealthCare, want: 0},
	} {
		t.Run(name, func(t *testing.T) {

			got := NationalPensionMonthsIn(tc.year, tc.born, tc.cover, tc.insured, tc.spouse)

			if got.Count() != tc.want {
				t.Errorf("NationalPensionMonthsIn(%d, %v, %q, %v, %v) = %d, want %d",
					tc.year, tc.born, tc.cover, tc.insured, tc.spouse, got, tc.want)
			}
		})
	}
}
