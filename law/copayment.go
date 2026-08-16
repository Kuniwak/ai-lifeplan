package law

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

const LateElderlyAge = 75

const (
	CopaymentSchoolAge = 6

	CopaymentElderlyAge = 70
)

const (
	CopaymentActiveEarnerTaxableIncome money.Yen = 1_450_000

	CopaymentActiveEarnerRevenue      money.Yen = 5_200_000
	CopaymentActiveEarnerRevenueAlone money.Yen = 3_830_000

	CopaymentTwoTenthsTaxableIncome money.Yen = 280_000

	CopaymentTwoTenthsIncome      money.Yen = 3_200_000
	CopaymentTwoTenthsIncomeAlone money.Yen = 2_000_000
)

type CopaymentIncome struct {
	HighestTaxableIncome money.Yen

	PensionAndOtherIncome money.Yen

	Revenue money.Yen

	AloneInNationalHealth bool
	AloneInLateElderly    bool
}

func MedicalCopaymentShare(age int, income CopaymentIncome) money.Rate {
	if age < CopaymentSchoolAge {
		return money.NewRate(2, 10)
	}
	if age < CopaymentElderlyAge {
		return money.NewRate(3, 10)
	}
	alone := income.AloneInNationalHealth
	if age >= LateElderlyAge {
		alone = income.AloneInLateElderly
	}
	if income.isActiveEarner(alone) {
		return money.NewRate(3, 10)
	}
	if age < LateElderlyAge {
		return money.NewRate(2, 10)
	}
	if income.reachesTwoTenths() {
		return money.NewRate(2, 10)
	}
	return money.NewRate(1, 10)
}

func (i CopaymentIncome) isActiveEarner(alone bool) bool {
	if i.HighestTaxableIncome < CopaymentActiveEarnerTaxableIncome {
		return false
	}
	floor := CopaymentActiveEarnerRevenue
	if alone {
		floor = CopaymentActiveEarnerRevenueAlone
	}
	return i.Revenue >= floor
}

func (i CopaymentIncome) reachesTwoTenths() bool {
	if i.HighestTaxableIncome < CopaymentTwoTenthsTaxableIncome {
		return false
	}
	floor := CopaymentTwoTenthsIncome
	if i.AloneInLateElderly {
		floor = CopaymentTwoTenthsIncomeAlone
	}
	return i.PensionAndOtherIncome >= floor
}

func CopaymentShareInMonth(year date.Year, month int, born date.Date, income CopaymentIncome) money.Rate {
	return MedicalCopaymentShare(copaymentAgeIn(year, month, born), income)
}

func copaymentAgeIn(year date.Year, month int, born date.Date) int {
	age := int(year - born.Year)
	if reached := born.ReachesAge(age); reached.Year > year || (reached.Year == year && reached.Month >= month) {
		age--
	}
	if age >= LateElderlyAge-1 {
		if birthday := born.Anniversary(age + 1); birthday.Year == year && birthday.Month <= month {
			age++
		}
	}
	return age
}
