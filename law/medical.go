package law

import "github.com/Kuniwak/lifeplan/money"

const (
	MedicalDeductionFloor money.Yen = 100_000

	MedicalDeductionCap money.Yen = 2_000_000
)

var MedicalDeductionRate = money.NewRate(5, 100)

func MedicalDeduction(paid, refunded, totalIncome money.Yen) money.Yen {
	outOfPocket := paid - refunded
	if outOfPocket <= 0 {
		return 0
	}

	floor := min(max(totalIncome, 0).Mul(MedicalDeductionRate, money.Truncate), MedicalDeductionFloor)

	return min(max(outOfPocket-floor, 0), MedicalDeductionCap)
}
