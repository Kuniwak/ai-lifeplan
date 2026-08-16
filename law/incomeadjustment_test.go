package law

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

func TestSalaryIncomeAdjustmentEligible(t *testing.T) {
	type testCase struct {
		Ages     []int
		Expected bool
	}

	testCases := map[string]testCase{
		"a newborn":                               {Ages: []int{0}, Expected: true},
		"just under the limit (boundary)":         {Ages: []int{22}, Expected: true},
		"exactly at the limit (boundary)":         {Ages: []int{23}, Expected: false},
		"one child qualifies, the other does not": {Ages: []int{25, 20}, Expected: true},
		"all of them are too old":                 {Ages: []int{23, 26}, Expected: false},
		"no dependants at all":                    {Ages: nil, Expected: false},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := SalaryIncomeAdjustmentEligible(tc.Ages)

			if got != tc.Expected {
				t.Errorf("所得金額調整控除の対象(%v) = %v, want %v", tc.Ages, got, tc.Expected)
			}
		})
	}
}

func TestSalaryIncomeAdjustmentShouldNeverExceedTheCap(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		salary := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "salary"))
		eligible := rapid.Bool().Draw(t, "eligible")
		year := date.Year(rapid.IntRange(2020, 2100).Draw(t, "year"))

		got := SalaryIncomeAdjustment(salary, eligible)

		if got < 0 || got > SalaryIncomeAdjustmentCap {
			t.Fatalf("給与収入 %d の調整控除が %d で、0 と上限 %d の外にある", salary, got, SalaryIncomeAdjustmentCap)
		}
		if SalaryIncomeAfterAdjustment(salary, eligible, year) < 0 {
			t.Fatalf("給与収入 %d で給与所得が負になった", salary)
		}
	})
}

func TestSalaryIncomeAfterAdjustmentShouldBeMonotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		lower := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "lower"))
		higher := money.Yen(rapid.Int64Range(0, 200_000_000).Draw(t, "higher"))
		if lower > higher {
			lower, higher = higher, lower
		}
		eligible := rapid.Bool().Draw(t, "eligible")
		year := date.Year(rapid.IntRange(2020, 2100).Draw(t, "year"))

		if SalaryIncomeAfterAdjustment(lower, eligible, year) > SalaryIncomeAfterAdjustment(higher, eligible, year) {
			t.Fatalf("給与収入 %d の所得が %d に増えたときに減った", lower, higher)
		}
	})
}
