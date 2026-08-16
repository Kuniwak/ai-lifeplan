package law

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

const RetirementDeductionFloor money.Yen = 800_000

const RetirementDeductionShortYears = 20

func RetirementIncomeDeduction(serviceYears int) money.Yen {
	if serviceYears <= RetirementDeductionShortYears {
		return max(money.Yen(serviceYears)*400_000, RetirementDeductionFloor)
	}
	return 8_000_000 + money.Yen(serviceYears-RetirementDeductionShortYears)*700_000
}

func RetirementIncome(payment money.Yen, serviceYears int) money.Yen {
	return max(payment-RetirementIncomeDeduction(serviceYears), 0) / 2
}

func RetirementResidentRate(m ResidentRates) (municipal, prefectural money.Rate) {
	if m.DesignatedCity {
		return money.NewRate(8, 100), money.NewRate(2, 100)
	}
	return money.NewRate(6, 100), money.NewRate(4, 100)
}

func RetirementIncomeTax(payment money.Yen, serviceYears int, year date.Year, m ResidentRates) (incomeTax, resident money.Yen) {
	income := RetirementIncome(payment, serviceYears).TruncateTaxableIncome()
	if income <= 0 {
		return 0, 0
	}

	base := IncomeTax(income)
	incomeTax = base + ReconstructionSurtax(base, year)

	municipalRate, prefecturalRate := RetirementResidentRate(m)
	resident = income.Mul(municipalRate, money.Truncate).TruncateIncomeTax() +
		income.Mul(prefecturalRate, money.Truncate).TruncateIncomeTax()
	return incomeTax, resident
}

const PensionDrawableAge = 60
