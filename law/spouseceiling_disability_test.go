package law

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

func TestASameLivelihoodSpouseHasNoTaxOfTheirOwnToDeductFrom(t *testing.T) {
	ceilings := spouseIncomeCeilings(t)

	checked := 0
	for year := date.Year(2018); year <= 2090; year++ {
		ceiling := ceilings.Ceiling(year)
		if ceiling <= 0 {
			t.Fatalf("%d 年の所得要件が %d である", year, ceiling)
		}
		checked++

		if left := max(ceiling-BasicDeduction(ceiling, year), 0); left != 0 {
			t.Errorf("%d 年 所得税: 同一生計配偶者の所得要件 %d に対し基礎控除は %d で、%d が残る。"+
				"**この年から、夫が妻の障害者控除を取りながら妻自身の課税所得も残る**",
				year, ceiling, BasicDeduction(ceiling, year), left)
		}

		exempt := residentDisabilityExemptionCeiling.At(year + 1)
		if ceiling > exempt {
			t.Errorf("%d 年 住民税: 同一生計配偶者の所得要件 %d が障害者等の非課税限度額 %d を超えた。"+
				"**この年から、障害者である同一生計配偶者に住民税が課され、自分の障害者控除がそれを減らす**",
				year, ceiling, exempt)
		}
	}
	if checked == 0 {
		t.Fatal("1 年も見ていない")
	}
}

func TestTheResidentBasicDeductionDoesNotCoverTheSameLivelihoodCeiling(t *testing.T) {
	ceilings := spouseIncomeCeilings(t)

	for _, year := range []int{2020, 2025, 2090} {
		ceiling := ceilings.Ceiling(date.Year(year))
		basic := ResidentBasicDeduction(ceiling, date.Year(year))
		if basic >= ceiling {
			t.Errorf("%d 年: 住民税の基礎控除 %d が所得要件 %d 以上になった。"+
				"TestASameLivelihoodSpouseHasNoTaxOfTheirOwnToDeductFrom の住民税の説明"+
				"（非課税限度額が効いている）を書き直すこと", year, basic, ceiling)
		}
	}
}

func TestTheDisabilityDeductionIsWorthWhatTheTableSays(t *testing.T) {
	table := MustLoadDisabilityDeductions(t, os.DirFS("../"+LawDirectory))

	hers, err := table.Lookup(OrdinaryDisability)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if want := money.Yen(270_000); hers.IncomeTax != want {
		t.Errorf("一般の障害者の障害者控除（所得税）= %d, want %d", hers.IncomeTax, want)
	}
}

func spouseIncomeCeilings(t *testing.T) SpouseIncomeCeilingTable {
	t.Helper()

	return MustLoadSpouseIncomeCeilings(t, os.DirFS("../"+LawDirectory))
}
