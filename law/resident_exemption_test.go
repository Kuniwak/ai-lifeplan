package law

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/panictest"
	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/money"
)

const withTheAddition = 2021

func TestTheIncomeHalfIsNeverChargedWithoutThePerCapitaHalf(t *testing.T) {
	m := setagayaExemption(t)

	rapid.Check(t, func(t *rapid.T) {
		income := money.Yen(rapid.Int64Range(0, 5_000_000).Draw(t, "合計所得"))
		dependents := rapid.IntRange(0, 6).Draw(t, "扶養親族等の数")
		disabled := rapid.Bool().Draw(t, "障害者")

		got := ResidentTaxLiabilityOf(income, dependents, disabled, m, withTheAddition)
		if got.Income && !got.PerCapita {
			t.Fatalf("合計所得 %d・扶養 %d・障害者 %v で所得割だけが課されている", income, dependents, disabled)
		}
	})
}

func TestResidentTaxLiabilityOfShouldRefuseAYearBeforeTheRecord(t *testing.T) {
	exemption := ResidentExemption{PerPerson: 350_000, Addition: 210_000}

	cases := map[string]struct {
		taxYear     date.Year
		wantRefused bool
	}{
		"記録の最初の課税年度（境界値）": {taxYear: 2017},
		"その 1 年前（境界値）":    {taxYear: 2016, wantRefused: true},
		"はるか前":            {taxYear: 1900, wantRefused: true},
		"加算の入る課税年度（境界値）":  {taxYear: 2021},
		"計画が最初に引く課税年度":    {taxYear: 2018},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			refused := panictest.Recovered(func() {
				ResidentTaxLiabilityOf(1_000_000, 0, false, exemption, c.taxYear)
			})

			if c.wantRefused {
				if refused == nil {
					t.Fatalf("記録より前の課税年度 %d が黙って答えられた", c.taxYear)
				}
				if message, _ := refused.(string); !strings.Contains(message, "2017") {
					t.Errorf("拒否のメッセージが記録の始まりを言っていない: %v", refused)
				}
				return
			}
			if refused != nil {
				t.Fatalf("課税年度 %d が拒まれた: %v", c.taxYear, refused)
			}
		})
	}
}
