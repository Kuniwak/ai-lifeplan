package law

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

func TestReconstructionSurtax(t *testing.T) {
	type testCase struct {
		BaseTax  money.Yen
		Year     date.Year
		Expected money.Yen
	}

	testCases := map[string]testCase{
		"a representative year":                 {BaseTax: 1_000_000, Year: 2031, Expected: 21_000},
		"the last year it is levied (boundary)": {BaseTax: 1_000_000, Year: 2037, Expected: 21_000},
		"the year after (boundary value)":       {BaseTax: 1_000_000, Year: 2038, Expected: 0},
		"no base tax":                           {BaseTax: 0, Year: 2031, Expected: 0},
		"the fraction is truncated":             {BaseTax: 100, Year: 2031, Expected: 2},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := ReconstructionSurtax(tc.BaseTax, tc.Year)

			if got != tc.Expected {
				t.Errorf("ReconstructionSurtax(%d, %d) = %d, want %d", tc.BaseTax, tc.Year, got, tc.Expected)
			}
		})
	}
}

func TestIncomeTaxShouldRiseWithTheTaxableIncome(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "a"))
		b := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "b"))
		if a > b {
			a, b = b, a
		}

		gotA, gotB := IncomeTax(a), IncomeTax(b)

		if gotA > gotB {
			t.Fatalf("taxable %d is taxed %d but the larger %d only %d", a, gotA, b, gotB)
		}
	})
}

func TestCrossingABandShouldNeverLeaveLessInHand(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		taxable := money.Yen(rapid.Int64Range(0, 200_000).Draw(t, "thousands")) * 1_000
		more := taxable + money.Yen(rapid.Int64Range(1, 2_000).Draw(t, "moreThousands"))*1_000

		afterTax := taxable - IncomeTax(taxable)
		afterTaxMore := more - IncomeTax(more)

		if afterTaxMore < afterTax {
			t.Fatalf("taxable %d leaves %d, but the larger %d leaves only %d", taxable, afterTax, more, afterTaxMore)
		}
	})
}

func TestTheTruncationArtefactShouldStayWithinOneStep(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		taxable := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "taxable"))
		more := taxable + money.Yen(rapid.Int64Range(1, 2_000_000).Draw(t, "more"))
		mostLost := money.TaxableIncomeUnit.Mul(money.NewPercent(incomeTaxTopRate), money.Ceil)

		afterTax := taxable - IncomeTax(taxable)
		afterTaxMore := more - IncomeTax(more)

		if lost := afterTax - afterTaxMore; lost > mostLost {
			t.Fatalf("taxable %d leaves %d but the larger %d leaves %d, losing %d, more than the %d a truncation step can explain",
				taxable, afterTax, more, afterTaxMore, lost, mostLost)
		}
	})
}

func TestIncomeTaxShouldNeverExceedTheTaxableIncome(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		taxable := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "taxable"))

		got := IncomeTax(taxable)

		if got < 0 {
			t.Fatalf("taxable %d gives a negative tax %d", taxable, got)
		}
		if got > taxable {
			t.Fatalf("taxable %d is taxed %d, which is more than the income", taxable, got)
		}
	})
}

func TestABiggerBasicDeductionShouldNeverFollowABiggerIncome(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := money.Yen(rapid.Int64Range(0, 40_000_000).Draw(t, "a"))
		b := money.Yen(rapid.Int64Range(0, 40_000_000).Draw(t, "b"))
		if a > b {
			a, b = b, a
		}
		year := date.Year(rapid.IntRange(2020, 2100).Draw(t, "year"))

		if BasicDeduction(a, year) < BasicDeduction(b, year) {
			t.Fatalf("%d年: income %d deducts %d but the larger %d deducts %d",
				year, a, BasicDeduction(a, year), b, BasicDeduction(b, year))
		}
		if ResidentBasicDeduction(a, year) < ResidentBasicDeduction(b, year) {
			t.Fatalf("%d年: income %d deducts %d but the larger %d deducts %d",
				year, a, ResidentBasicDeduction(a, year), b, ResidentBasicDeduction(b, year))
		}
	})
}
