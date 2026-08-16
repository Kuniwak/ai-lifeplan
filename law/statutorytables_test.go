package law

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/panictest"
)

func loadedSocialInsuranceRates(t *testing.T) SocialInsuranceRates {
	t.Helper()

	return MustLoadSocialInsuranceRates(t, os.DirFS("../"+LawDirectory))
}

func loadedKouki(t *testing.T, region Prefecture) KoukiRatesTable {
	t.Helper()

	table, err := LoadKoukiRatesTable(os.DirFS("../"+LawDirectory), region)
	if err != nil {
		t.Fatalf("law.LoadKoukiRatesTable(%q): %v", region, err)
	}
	return table
}

func loadedKokuho(t *testing.T, region Municipality) KokuhoTable {
	t.Helper()

	table, err := LoadKokuhoTable(os.DirFS("../"+LawDirectory), region)
	if err != nil {
		t.Fatalf("law.LoadKokuhoTable(%q): %v", region, err)
	}
	return table
}

func loadedNursingCare(t *testing.T, region Municipality) NursingCarePremiumTable {
	t.Helper()

	table, err := LoadNursingCarePremiumTable(os.DirFS("../"+LawDirectory), region)
	if err != nil {
		t.Fatalf("law.LoadNursingCarePremiumTable(%q): %v", region, err)
	}
	return table
}

func TestEveryTableShouldRefuseAPairOnlyHalfAnsweredWhenHandedOne(t *testing.T) {
	one := GrowingSteadilyBy(2018, 2090, money.NewRate(1, 100))
	rates := loadedSocialInsuranceRates(t)
	kouki := loadedKouki(t, "東京都")
	kokuho := loadedKokuho(t, "世田谷区")
	nursingCare := loadedNursingCare(t, "世田谷区")

	for name, withGrowth := range map[string]func(CostGrowth){
		"健康保険・介護保険料率": func(g CostGrowth) { rates.WithGrowth(g) },
		"後期高齢者医療":     func(g CostGrowth) { kouki.WithGrowth(g) },
		"国民健康保険税":     func(g CostGrowth) { kokuho.WithGrowth(g) },
		"介護保険料":       func(g CostGrowth) { nursingCare.WithGrowth(g) },
	} {
		for handed, c := range map[string]struct {
			growth  CostGrowth
			refused bool
		}{
			"全部答えた":         {CostGrowth{Medical: one, Care: one, CarePremium: one}, false},
			"伸ばさないと言った":     {NoCostGrowth(), false},
			"介護だけ答えた":       {CostGrowth{Care: one}, true},
			"介護保険料だけ答えていない": {CostGrowth{Medical: one, Care: one}, true},
			"医療だけ答えた":       {CostGrowth{Medical: one}, true},
			"どちらも答えていない":    {CostGrowth{}, true},
		} {
			t.Run(name+":"+handed, func(t *testing.T) {
				refused := panictest.Recovered(func() { withGrowth(c.growth) })

				if c.refused && refused == nil {
					t.Fatalf("%s が %+v を黙って受け取った", name, c.growth)
				}
				if !c.refused && refused != nil {
					t.Fatalf("%s が拒んだ: %v", name, refused)
				}
			})
		}
	}
}
