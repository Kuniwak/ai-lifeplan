package law

import (
	"testing"

	"github.com/Kuniwak/lifeplan/money"
)

func TestRetirementIncomeDeduction(t *testing.T) {
	type testCase struct {
		ServiceYears int
		Want         money.Yen
	}

	testCases := map[string]testCase{
		"five years":                    {ServiceYears: 5, Want: 2_000_000},
		"twenty years (boundary value)": {ServiceYears: 20, Want: 8_000_000},
		"twenty one years":              {ServiceYears: 21, Want: 8_700_000},
		"twenty six years, this iDeCo":  {ServiceYears: 26, Want: 12_200_000},
		"one year is the floor of 800万": {ServiceYears: 1, Want: 800_000},

		"two years (boundary value)": {ServiceYears: 2, Want: 800_000},
		"three years":                {ServiceYears: 3, Want: 1_200_000},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if got := RetirementIncomeDeduction(tc.ServiceYears); got != tc.Want {
				t.Errorf("RetirementIncomeDeduction(%d) = %d, want %d", tc.ServiceYears, got, tc.Want)
			}
		})
	}
}

func TestRetirementIncomeTaxOfThisHouseholdsIDeCoIsNothing(t *testing.T) {
	m := setagaya(t)

	incomeTax, resident := RetirementIncomeTax(8_455_000, 26, 2049, m)
	if incomeTax != 0 || resident != 0 {
		t.Errorf("iDeCo 8,455,000 を勤続 26 年で受け取ると 所得税 %d・住民税 %d、want 0・0", incomeTax, resident)
	}
}

func TestRetirementIncomeTaxOfAPaymentOverTheDeduction(t *testing.T) {
	m := setagaya(t)

	incomeTax, resident := RetirementIncomeTax(20_000_000, 26, 2049, m)
	if want := money.Yen(352_000); incomeTax != want {
		t.Errorf("所得税 = %d, want %d", incomeTax, want)
	}
	if want := money.Yen(390_000); resident != want {
		t.Errorf("住民税 = %d, want %d", resident, want)
	}
}
