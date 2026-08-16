package law

import (
	"fmt"
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

var theEmploymentInsuranceLeaflets = map[date.Year]string{
	2017: "0.30%", 2018: "0.30%", 2019: "0.30%", 2020: "0.30%", 2021: "0.30%",
	2022: "0.50%",
	2023: "0.60%", 2024: "0.60%", 2025: "0.55%", 2026: "0.50%",
}

var theEmploymentInsuranceLeafletsBeforeApril = map[date.Year]string{
	2022: "0.30%",
}

func TestTheEmploymentInsuranceRateShouldBeTheHighestTheCalendarYearMeets(t *testing.T) {
	loaded := MustLoadEmploymentInsuranceRates(t, os.DirFS("../"+LawDirectory))

	for year := date.Year(2018); year <= 2026; year++ {
		t.Run(fmt.Sprintf("%d年", year), func(t *testing.T) {
			januaryToMarch, ok := theEmploymentInsuranceLeafletsBeforeApril[year-1]
			if !ok {
				januaryToMarch = theEmploymentInsuranceLeaflets[year-1]
			}
			aprilOnwards := theEmploymentInsuranceLeaflets[year]
			if januaryToMarch == "" || aprilOnwards == "" {
				t.Fatalf("%d 年が出会う率のどちらかを告知から読んでいない", year)
			}

			want := higherOf(t, januaryToMarch, aprilOnwards)

			got := loaded.Rate(year)

			if got != want {
				t.Errorf("%d 年の雇用保険料率が %v。この暦年は %s と %s に出会うので、"+
					"高いほうの %v のはず（保険料を大きく見る＝安全側）",
					year, got, januaryToMarch, aprilOnwards, want)
			}
		})
	}
}

func TestTheEmploymentInsuranceRateShouldMatchThePayslips(t *testing.T) {
	loaded := MustLoadEmploymentInsuranceRates(t, os.DirFS("../"+LawDirectory))

	type testCase struct {
		Year    date.Year
		Written string
	}
	testCases := map[string]testCase{
		"2024 は年を通して 0.60%":               {Year: 2024, Written: "0.60%"},
		"2025 は 1〜3 月が 0.60%、4 月から 0.55%": {Year: 2025, Written: "0.60%"},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			want, err := money.ParsePercent(tc.Written)
			if err != nil {
				t.Fatalf("money.ParsePercent: %v", err)
			}
			if got := loaded.Rate(tc.Year); got != want {
				t.Errorf("%d 年の雇用保険料率が %v、給与明細からは %v", tc.Year, got, want)
			}
		})
	}
}

func higherOf(t *testing.T, a, b string) money.Rate {
	t.Helper()

	first, err := money.ParsePercent(a)
	if err != nil {
		t.Fatalf("money.ParsePercent(%q): %v", a, err)
	}
	second, err := money.ParsePercent(b)
	if err != nil {
		t.Fatalf("money.ParsePercent(%q): %v", b, err)
	}
	if first.Cmp(second) >= 0 {
		return first
	}
	return second
}
