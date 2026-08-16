package law

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/panictest"
)

func TestMedicalCostGrowthShouldNotTouchTheYearsThatAreWritten(t *testing.T) {
	g := GrowingSteadilyBy(2018, 2090, money.NewRate(1, 100))

	for _, year := range []int{2020, 2025, 2026} {
		if got := g.Since(2026, date.Year(year)); got.Apply(1_000_000) != 1_000_000 {
			t.Errorf("%d 年が %v 倍されている（書かれている年は動かない）", year, got)
		}
	}
}

func TestTheZeroGrowthShouldLeaveEveryYearAlone(t *testing.T) {
	var g CostGrowthCurve

	if got := g.Since(2026, 2090); got.Apply(1_000_000) != 1_000_000 {
		t.Errorf("伸び率ゼロで %v 倍されている", got)
	}
}

func TestTheStatutoryTablesShouldSayWhereTheyStopSayingAnything(t *testing.T) {
	rates := loadedSocialInsuranceRates(t)
	kouki := loadedKouki(t, "東京都")
	kokuho := loadedKokuho(t, "世田谷区")

	for name, table := range map[string]interface{ LastWrittenYear() (date.Year, bool) }{
		"健康保険料率":  rates.Health,
		"介護保険料率":  rates.NursingCare,
		"後期高齢者医療": kouki,
		"国民健康保険":  kokuho,
	} {
		last, ok := table.LastWrittenYear()
		if !ok {
			t.Errorf("%s: 行が無い", name)
			continue
		}
		if last < 2020 {
			t.Errorf("%s: 最後の行が %d 年である。表が古すぎるか、読み方が違う", name, last)
		}
	}
}

func TestEveryMedicalSchemeShouldGrowWithTheOneRate(t *testing.T) {
	growth := GrowingEverythingBy(GrowingSteadilyBy(2018, 2090, money.NewRate(1, 100)))

	rates := loadedSocialInsuranceRates(t)
	kouki := loadedKouki(t, "東京都")
	kokuho := loadedKokuho(t, "世田谷区")

	household := KokuhoHousehold{Members: []KokuhoMember{
		{Base: 3_000_000, Months: date.WholeYear, NursingCareMonths: date.WholeYear},
		{Months: date.WholeYear},
	}}
	kokuhoOf := func(t *testing.T, table KokuhoTable, year date.Year) money.Yen {
		t.Helper()
		got, err := table.Premium(household, year)
		if err != nil {
			t.Fatalf("law.KokuhoTable.Premium: %v", err)
		}
		return got
	}

	cases := map[string]struct{ flat, grown money.Yen }{
		"健康保険": {
			flat:  rates.WithGrowth(NoCostGrowth()).HealthPremium(500_000, 2050),
			grown: rates.WithGrowth(growth).HealthPremium(500_000, 2050),
		},
		"介護保険": {
			flat:  rates.WithGrowth(NoCostGrowth()).NursingCarePremium(500_000, 2050),
			grown: rates.WithGrowth(growth).NursingCarePremium(500_000, 2050),
		},
		"後期高齢者医療": {
			flat:  kouki.WithGrowth(NoCostGrowth()).Premium(3_000_000, 2080),
			grown: kouki.WithGrowth(growth).Premium(3_000_000, 2080),
		},
		"国民健康保険": {
			flat:  kokuhoOf(t, kokuho.WithGrowth(NoCostGrowth()), 2060),
			grown: kokuhoOf(t, kokuho.WithGrowth(growth), 2060),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if c.flat <= 0 {
				t.Fatalf("据え置きの保険料が %v である。この検査が空回りしている", c.flat)
			}
			if c.grown <= c.flat {
				t.Errorf("伸ばしても %v のまま（据え置き %v）。医療費上昇率が届いていない", c.grown, c.flat)
			}
		})
	}
}

func TestTheGrowthShouldRefuseASpanItCannotCoverRatherThanGoFlat(t *testing.T) {
	const lastWritten = 2026

	covered := GrowingSteadilyBy(2018, 2090, money.NewRate(1, 100))
	if err := covered.AssertCovers("健康保険料率", lastWritten, 2090); err != nil {
		t.Errorf("覆えている率を拒んでいる: %v", err)
	}

	short := GrowingSteadilyBy(2030, 2090, money.NewRate(1, 100))
	err := short.AssertCovers("健康保険料率", lastWritten, 2090)
	if err == nil {
		t.Fatal("2027 年の行が無い率を受け入れた。伸びが黙って消える")
	}
	if !strings.Contains(err.Error(), "2027") {
		t.Errorf("どの年が無いのかを言っていない: %v", err)
	}

	if err := NoCostGrowthCurve().AssertCovers("健康保険料率", lastWritten, 2090); err != nil {
		t.Errorf("据え置きを拒んでいる: %v", err)
	}
}

func TestNursingCareShouldGrowWithItsOwnRateAndNotTheMedicalOne(t *testing.T) {
	rates := loadedSocialInsuranceRates(t)
	kokuho := loadedKokuho(t, "世田谷区")

	one := GrowingSteadilyBy(2018, 2090, money.NewRate(1, 100))
	careOnly := CostGrowth{Medical: NoCostGrowthCurve(), Care: one, CarePremium: one}
	medicalOnly := CostGrowth{Medical: one, Care: NoCostGrowthCurve(), CarePremium: NoCostGrowthCurve()}

	withCare := KokuhoHousehold{Members: []KokuhoMember{
		{Base: 3_000_000, Months: date.WholeYear, NursingCareMonths: date.WholeYear},
		{Months: date.WholeYear},
	}}
	withoutCare := KokuhoHousehold{Members: []KokuhoMember{
		{Base: 3_000_000, Months: date.WholeYear},
		{Months: date.WholeYear},
	}}
	kokuhoOf := func(t *testing.T, g CostGrowth, household KokuhoHousehold) money.Yen {
		t.Helper()
		got, err := kokuho.WithGrowth(g).Premium(household, 2060)
		if err != nil {
			t.Fatalf("law.KokuhoTable.Premium: %v", err)
		}
		return got
	}

	for _, c := range []struct {
		name   string
		flat   func(*testing.T) money.Yen
		grown  func(*testing.T, CostGrowth) money.Yen
		byCare bool
	}{
		{
			name:   "健康保険は医療費上昇率で動く",
			flat:   func(*testing.T) money.Yen { return rates.WithGrowth(NoCostGrowth()).HealthPremium(500_000, 2050) },
			grown:  func(_ *testing.T, g CostGrowth) money.Yen { return rates.WithGrowth(g).HealthPremium(500_000, 2050) },
			byCare: false,
		},
		{
			name: "介護保険は介護費上昇率で動く",
			flat: func(*testing.T) money.Yen {
				return rates.WithGrowth(NoCostGrowth()).NursingCarePremium(500_000, 2050)
			},
			grown: func(_ *testing.T, g CostGrowth) money.Yen {
				return rates.WithGrowth(g).NursingCarePremium(500_000, 2050)
			},
			byCare: true,
		},
		{
			name: "国民健康保険税の介護分は介護費上昇率で動く",
			flat: func(t *testing.T) money.Yen {
				return kokuhoOf(t, NoCostGrowth(), withCare) - kokuhoOf(t, NoCostGrowth(), withoutCare)
			},
			grown: func(t *testing.T, g CostGrowth) money.Yen {
				return kokuhoOf(t, g, withCare) - kokuhoOf(t, g, withoutCare)
			},
			byCare: true,
		},
		{
			name:   "国民健康保険税の介護分以外は医療費上昇率で動く",
			flat:   func(t *testing.T) money.Yen { return kokuhoOf(t, NoCostGrowth(), withoutCare) },
			grown:  func(t *testing.T, g CostGrowth) money.Yen { return kokuhoOf(t, g, withoutCare) },
			byCare: false,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			flat := c.flat(t)
			if flat <= 0 {
				t.Fatalf("据え置きの保険料が %v である。この検査が空回りしている", flat)
			}

			moves, holds := medicalOnly, careOnly
			movesBy, holdsBy := "医療費上昇率", "介護費上昇率"
			if c.byCare {
				moves, holds = careOnly, medicalOnly
				movesBy, holdsBy = "介護費上昇率", "医療費上昇率"
			}
			if got := c.grown(t, moves); got <= flat {
				t.Errorf("%s を与えたのに %v のまま（据え置き %v）", movesBy, got, flat)
			}
			if got := c.grown(t, holds); got != flat {
				t.Errorf("%s を与えただけで %v に動いた（据え置き %v）", holdsBy, got, flat)
			}
		})
	}
}

func TestGrowthNobodyStatedShouldBeRefused(t *testing.T) {
	one := GrowingSteadilyBy(2020, 2030, money.NewRate(1, 100))
	lastWritten := func() (date.Year, bool) { return 2020, true }

	for door, open := range map[string]func(*testing.T, CostGrowthCurve){
		"GrowPremium": func(t *testing.T, g CostGrowthCurve) {
			if got := g.GrowPremium(100_000, lastWritten, 2030); got <= 0 {
				t.Errorf("保険料が %d である。この検査が空回りしている", got)
			}
		},

		"AssertCovers": func(t *testing.T, g CostGrowthCurve) {
			if err := g.AssertCovers("健康保険料率", 2020, 2030); err != nil {
				t.Errorf("覆えている伸びを拒んでいる: %v", err)
			}
		},
	} {
		for name, c := range map[string]struct {
			growth  CostGrowthCurve
			refused bool
		}{
			"率を渡した":      {one, false},
			"伸ばさないと言った":  {NoCostGrowthCurve(), false},
			"誰も何も言っていない": {CostGrowthCurve{}, true},
		} {
			t.Run(door+":"+name, func(t *testing.T) {
				refused := panictest.Recovered(func() { open(t, c.growth) })

				if c.refused {
					if refused == nil {
						t.Fatal("誰も答えていない伸びが黙って通った")
					}

					if got := fmt.Sprint(refused); !strings.Contains(got, "law.NoCostGrowth()") {
						t.Errorf("どう直せばよいかを言っていない: %v", got)
					}
					return
				}
				if refused != nil {
					t.Fatalf("拒まれた: %v", refused)
				}
			})
		}
	}
}

func TestWithGrowthShouldRefuseAPairOnlyHalfAnswered(t *testing.T) {
	one := GrowingSteadilyBy(2018, 2090, money.NewRate(1, 100))

	for name, c := range map[string]struct {
		growth  CostGrowth
		refused bool

		names string
	}{
		"全部答えた":         {CostGrowth{Medical: one, Care: one, CarePremium: one}, false, ""},
		"両方とも伸ばさないと言った": {NoCostGrowth(), false, ""},
		"介護だけ答えた":       {CostGrowth{Care: one}, true, "Medical に law.NoCostGrowthCurve()"},
		"医療だけ答えた":       {CostGrowth{Medical: one}, true, "Care に law.NoCostGrowthCurve()"},
		"どちらも答えていない":    {CostGrowth{}, true, "law.NoCostGrowthCurve()"},
		"介護保険料だけ答えていない": {CostGrowth{Medical: one, Care: one}, true, "CarePremium に law.NoCostGrowthCurve()"},
	} {
		t.Run(name, func(t *testing.T) {
			message, refused := panictest.Message(func() { c.growth.AssertStated() })

			if c.refused && !refused {
				t.Fatalf("片方しか答えていない対が黙って通った: %+v", c.growth)
			}
			if !c.refused && refused {
				t.Fatalf("拒まれた: %v", message)
			}
			if c.names != "" && !strings.Contains(message, c.names) {
				t.Errorf("直し方が %q を指していない: %q", c.names, message)
			}
			if c.refused && strings.Contains(message, "law.NoCostGrowth()") {
				t.Errorf("対を返す law.NoCostGrowth() を直し方として挙げている。"+
					"答えた側の半分を捨てることになり、そもそも欄に入らない: %q", message)
			}
		})
	}
}
