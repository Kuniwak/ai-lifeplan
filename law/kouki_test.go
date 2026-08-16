package law

import (
	"os"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/panictest"
)

func koukiRates(t *testing.T, region string) KoukiRatesTable {
	t.Helper()

	return loadedKouki(t, Prefecture(region)).WithGrowth(NoCostGrowth())
}

func TestKoukiRates(t *testing.T) {
	type testCase struct {
		Year       date.Year
		PerCapita  money.Yen
		IncomeRate money.Rate
	}

	testCases := map[string]testCase{
		"令和6・7年度 の初年":                             {Year: 2024, PerCapita: 47_300, IncomeRate: money.NewRate(967, 10_000)},
		"令和6・7年度 の 2 年目":                          {Year: 2025, PerCapita: 47_300, IncomeRate: money.NewRate(967, 10_000)},
		"令和8・9年度 に切り替わる年":                         {Year: 2026, PerCapita: 53_300, IncomeRate: money.NewRate(988, 10_000)},
		"past the record, the last figures stand": {Year: 2064, PerCapita: 53_300, IncomeRate: money.NewRate(988, 10_000)},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			rates := koukiRates(t, "東京都")

			part, ok := rates.PartAt(KoukiMedical, tc.Year)
			if !ok {
				t.Fatalf("医療分に %d 年の行が無い", tc.Year)
			}
			perCapita, incomeRate := part.PerCapita, part.IncomeRate

			if perCapita != tc.PerCapita {
				t.Errorf("均等割額(%d) = %d, want %d", tc.Year, perCapita, tc.PerCapita)
			}
			if incomeRate != tc.IncomeRate {
				t.Errorf("所得割率(%d) = %v, want %v", tc.Year, incomeRate, tc.IncomeRate)
			}
		})
	}
}

func TestLoadKoukiRatesTableShouldRejectAnUnknownRegion(t *testing.T) {
	const region = "北海道"

	_, err := LoadKoukiRatesTable(os.DirFS("../"+LawDirectory), region)

	if err == nil {
		t.Fatalf("LoadKoukiRatesTable(%q) succeeded; an unknown region must be reported", region)
	}
}

func koukiPart(name KoukiPartName) *KoukiPartName { return &name }

func TestTheKoukiPremiumShouldRefuseAYearBeforeItsRecord(t *testing.T) {
	rates := koukiRates(t, "東京都")

	if got := panictest.Recovered(func() { rates.Premium(2_430_000, 2019) }); got == nil {
		t.Error("表の最初の行より前の年を 0 円として通している")
	}
	if got := rates.PremiumOf(KoukiChildSupport, 2_430_000, 2025); got != 0 {
		t.Errorf("令和7年度の子ども分 = %d, want 0", got)
	}
}

func TestTheKoukiGrowthShouldRefuseAPartNobodyDecidedAbout(t *testing.T) {
	if got := panictest.Recovered(func() { koukiGrowthOf(NoCostGrowth(), "そんな区分は無い") }); got == nil {
		t.Error("誰も伸びを決めていない区分が黙って医療の伸びに落ちている")
	}
}

func TestTheKoukiCapsShouldBeMultiplesOfTheTruncation(t *testing.T) {
	rates := koukiRates(t, "東京都")

	seen := 0
	for _, name := range rates.Parts() {
		for year := 2020; year <= 2030; year++ {
			part, ok := rates.PartAt(name, date.Year(year))
			if !ok {
				continue
			}
			if part.Cap%KoukiPremiumUnit != 0 {
				t.Errorf("%s の %d 年度の賦課限度額 %d は %d 円の倍数ではない。"+
					"切捨は頭を打った後に走るので、この額は黙って下がる",
					name, year, part.Cap, KoukiPremiumUnit)
			}
			seen++
		}
	}
	if seen == 0 {
		t.Error("賦課限度額を 1 つも読んでいない")
	}
}

func TestKoukiPremiumShouldNeverFallAsIncomeRises(t *testing.T) {
	rates := koukiRates(t, "東京都")

	rapid.Check(t, func(t *rapid.T) {
		year := date.Year(rapid.IntRange(2024, 2094).Draw(t, "year"))
		poorer := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "poorer"))
		richer := money.Yen(rapid.Int64Range(0, 30_000_000).Draw(t, "richer"))
		if poorer > richer {
			poorer, richer = richer, poorer
		}

		low := rates.Premium(poorer, year)
		high := rates.Premium(richer, year)

		if high < low {
			t.Fatalf("income %d pays %d but income %d pays only %d", poorer, low, richer, high)
		}
		if low < 47_300 {
			t.Fatalf("income %d pays %d, less than the per capita levy", poorer, low)
		}
	})
}

func TestTheKoukiCapShouldBeAppliedBeforeTheGrowth(t *testing.T) {
	const (
		last   = 2026
		year   = 2030
		capped = money.Yen(850_000)
	)
	rates := loadedKouki(t, "東京都").WithGrowth(CostGrowth{
		Medical:     GrowingSteadilyBy(last+1, year, money.NewRate(2, 100)),
		Care:        NoCostGrowthCurve(),
		CarePremium: NoCostGrowthCurve(),
	})

	got := rates.Premium(30_000_000, year)

	if got <= capped {
		t.Errorf("%d 年の保険料 = %d。表の最後の年の限度額 %d を超えないなら、"+
			"限度額が凍っている——伸ばした後に上限を当てている", year, got, capped)
	}
	if got <= rates.Premium(30_000_000, last) {
		t.Errorf("%d 年の保険料 %d が %d 年の %d を上回らない", year, got, last, rates.Premium(30_000_000, last))
	}
}
