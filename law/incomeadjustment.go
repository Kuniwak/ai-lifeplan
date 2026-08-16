package law

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

const (
	SalaryIncomeAdjustmentThreshold money.Yen = 8_500_000

	SalaryIncomeAdjustmentCeiling money.Yen = 10_000_000

	SalaryIncomeAdjustmentCap money.Yen = 150_000

	SalaryIncomeAdjustmentDependantAge = 23
)

func SalaryIncomeAdjustmentEligible(dependantAges []int) bool {
	for _, age := range dependantAges {
		if age < SalaryIncomeAdjustmentDependantAge {
			return true
		}
	}
	return false
}

func SalaryIncomeAdjustment(salary money.Yen, eligible bool) money.Yen {
	if !eligible || salary <= SalaryIncomeAdjustmentThreshold {
		return 0
	}

	counted := min(salary, SalaryIncomeAdjustmentCeiling)
	return (counted - SalaryIncomeAdjustmentThreshold).Mul(money.NewRate(10, 100), money.Ceil)
}

func SalaryIncomeAfterAdjustment(salary money.Yen, eligible bool, year date.Year) money.Yen {
	return SalaryIncome(salary, year) - SalaryIncomeAdjustment(salary, eligible)
}

const PensionAndSalaryAdjustmentCap money.Yen = 100_000

func PensionAndSalaryAdjustment(salaryIncome, pensionIncome money.Yen) money.Yen {
	counted := min(salaryIncome, PensionAndSalaryAdjustmentCap) +
		min(pensionIncome, PensionAndSalaryAdjustmentCap)
	return max(counted-PensionAndSalaryAdjustmentCap, 0)
}

func TotalIncomeAdjustment(salary, pensionIncome money.Yen, eligible bool, year date.Year) (first, second money.Yen) {
	first = SalaryIncomeAdjustment(salary, eligible)
	second = PensionAndSalaryAdjustment(SalaryIncome(salary, year)-first, pensionIncome)
	return first, second
}
