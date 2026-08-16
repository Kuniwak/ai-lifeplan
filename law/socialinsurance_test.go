package law

import (
	"github.com/Kuniwak/lifeplan/date"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/money"
)

func kyokaiRates(t *testing.T) SocialInsuranceRates {
	t.Helper()

	return loadedSocialInsuranceRates(t).WithGrowth(NoCostGrowth())
}

func standardRemunerationFromPremium(premium money.Yen, table StandardRemunerationTable, charge func(money.Yen) money.Yen) (money.Yen, bool) {
	for _, band := range table.Bands() {
		if charge(band.Standard) == premium {
			return band.Standard, true
		}
	}
	return 0, false
}

func TestNursingCareInsured(t *testing.T) {
	type testCase struct {
		Age      int
		Expected bool
	}

	testCases := map[string]testCase{
		"just below the lower bound (boundary)": {Age: 39, Expected: false},
		"the lower bound (boundary)":            {Age: 40, Expected: true},
		"the upper bound (boundary)":            {Age: 64, Expected: true},
		"just above the upper bound (boundary)": {Age: 65, Expected: false},
		"a child":                               {Age: 0, Expected: false},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := NursingCareInsured(tc.Age)

			if got != tc.Expected {
				t.Errorf("NursingCareInsured(%d) = %v, want %v", tc.Age, got, tc.Expected)
			}
		})
	}
}

func TestStandardBonusShouldLoseLessThanAThousandYen(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gross := money.Yen(rapid.Int64Range(0, 100_000_000).Draw(t, "gross"))

		standard := StandardBonus(gross)

		if standard > gross {
			t.Fatalf("StandardBonus(%d) = %d, more than the bonus itself", gross, standard)
		}
		if gross-standard >= StandardBonusUnit {
			t.Fatalf("StandardBonus(%d) = %d, losing %d yen, which is a whole unit or more",
				gross, standard, gross-standard)
		}
		if standard%StandardBonusUnit != 0 {
			t.Fatalf("StandardBonus(%d) = %d, which is not a multiple of %d",
				gross, standard, StandardBonusUnit)
		}
	})
}

func TestPremiumsShouldBeMonotonicInTheStandardAmount(t *testing.T) {
	rates := kyokaiRates(t)

	rapid.Check(t, func(t *rapid.T) {
		year := date.Year(rapid.IntRange(2016, 2094).Draw(t, "year"))
		lower := money.Yen(rapid.Int64Range(0, 1_500_000).Draw(t, "lower"))
		higher := money.Yen(rapid.Int64Range(0, 1_500_000).Draw(t, "higher"))
		if lower > higher {
			lower, higher = higher, lower
		}

		if rates.HealthPremium(lower, year) > rates.HealthPremium(higher, year) {
			t.Fatalf("健康保険料 falls from %d to %d as the standard amount rises from %d to %d",
				rates.HealthPremium(lower, year), rates.HealthPremium(higher, year), lower, higher)
		}
		if PensionInsurancePremium(lower) > PensionInsurancePremium(higher) {
			t.Fatalf("厚生年金保険料 falls from %d to %d as the standard amount rises from %d to %d",
				PensionInsurancePremium(lower), PensionInsurancePremium(higher), lower, higher)
		}
		if rates.NursingCarePremium(lower, year) > rates.NursingCarePremium(higher, year) {
			t.Fatalf("介護保険料 falls from %d to %d as the standard amount rises from %d to %d",
				rates.NursingCarePremium(lower, year), rates.NursingCarePremium(higher, year), lower, higher)
		}
	})
}

func TestTruncatingTheInsuredHalfShouldMatchTheStatutoryRounding(t *testing.T) {
	rates := kyokaiRates(t)

	rapid.Check(t, func(t *rapid.T) {
		standard := money.Yen(rapid.Int64Range(0, 1_500).Draw(t, "thousands")) * StandardBonusUnit

		year := date.Year(rapid.IntRange(2016, 2094).Draw(t, "year"))

		for _, rate := range []money.Rate{rates.Health.Rate(year), PensionRateInsured, rates.NursingCare.Rate(year)} {
			got := insuredPremium(standard, rate)

			doubled := standard * 2
			if got*2 != doubled.Mul(rate, money.Truncate) && got*2 != doubled.Mul(rate, money.Truncate)-1 {
				t.Fatalf("insuredPremium(%d, %v) = %d, which is not the truncated product", standard, rate, got)
			}
		}
	})
}

func TestHealthStandardBonusShouldNeverPassTheFiscalYearCap(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bonuses := rapid.SliceOfN(rapid.Int64Range(0, 10_000_000), 0, 8).Draw(t, "bonuses")

		var charged money.Yen
		for _, gross := range bonuses {
			charged += HealthStandardBonus(charged, money.Yen(gross))
		}

		if charged > HealthStandardBonusCapFiscalYear {
			t.Fatalf("bonuses %v charge %d in standard bonus, past the cap of %d",
				bonuses, charged, HealthStandardBonusCapFiscalYear)
		}
	})
}

func TestThePensionRateShouldBeTheRateTheActFixes(t *testing.T) {
	if want := money.NewRate(183, 1_000).Div(2); PensionRateInsured.Cmp(want) != 0 {
		t.Errorf("厚生年金保険料率の被保険者負担が %v である（%v のはず）", PensionRateInsured, want)
	}

	var _ money.Rate = PensionRateInsured
}
