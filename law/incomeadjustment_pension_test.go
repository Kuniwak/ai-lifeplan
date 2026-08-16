package law

import (
	"testing"

	"github.com/Kuniwak/lifeplan/money"
)

func TestPensionAndSalaryAdjustmentShouldFollowTheFormula(t *testing.T) {
	type testCase struct {
		SalaryIncome, PensionIncome, Expected money.Yen
	}

	for name, tc := range map[string]testCase{
		"合計が 10 万円ちょうどでは効かない（境界）":      {50_000, 50_000, 0},
		"合計が 10 万円を 1 円超えると 1 円出る（境界）": {50_000, 50_001, 1},

		"どちらも 10 万円を超えると上限の 10 万円": {5_000_000, 3_000_000, 100_000},
		"給与だけ 10 万円を超える":           {5_000_000, 60_000, 60_000},
		"年金だけ 10 万円を超える":           {60_000, 5_000_000, 60_000},
		"給与がちょうど 10 万円（境界）":        {100_000, 5_000_000, 100_000},

		"年金が無い":  {5_000_000, 0, 0},
		"給与が無い":  {0, 5_000_000, 0},
		"どちらも無い": {0, 0, 0},
	} {
		t.Run(name, func(t *testing.T) {
			got := PensionAndSalaryAdjustment(tc.SalaryIncome, tc.PensionIncome)
			if got != tc.Expected {
				t.Errorf("所得金額調整控除(給与所得 %d, 公的年金等の雑所得 %d) = %d, want %d",
					tc.SalaryIncome, tc.PensionIncome, got, tc.Expected)
			}
		})
	}
}

func TestTheTwoAdjustmentsShouldBeAppliedInTheStatedOrder(t *testing.T) {
	const (
		salary  money.Yen = 9_000_000
		pension money.Yen = 5_000_000
		year              = 2024
	)

	first, second := TotalIncomeAdjustment(salary, pension, true, year)
	got := first + second

	if want := money.Yen(50_000 + 100_000); got != want {
		t.Errorf("二つ合わせた所得金額調整控除 = %d, want %d", got, want)
	}
}
