package law

import (
	"github.com/Kuniwak/lifeplan/money"

	"github.com/Kuniwak/lifeplan/date"
)

const (
	LifeInsuranceCategoryCapIncomeTax money.Yen = 40_000
	LifeInsuranceCategoryCapResident  money.Yen = 28_000

	LifeInsuranceTotalCapIncomeTax money.Yen = 120_000
	LifeInsuranceTotalCapResident  money.Yen = 70_000
)

func LifeInsuranceDeduction(premium money.Yen) Deduction {
	return Deduction{
		IncomeTax: lifeInsuranceIncomeTax(premium),
		Resident:  lifeInsuranceResident(premium),
	}
}

func GeneralLifeInsuranceDeduction(premium money.Yen, year date.Year, hasYoungDependant bool) Deduction {
	d := LifeInsuranceDeduction(premium)
	if hasYoungDependant && childRearingLifeInsuranceInForce.At(year) {
		d.IncomeTax = childRearingLifeInsurance(premium)
	}
	return d
}

var childRearingLifeInsuranceInForce = NewAmended("子育て世帯の一般生命保険料控除",
	YearRow[bool]{FromYear: 0, Value: false},
	YearRow[bool]{FromYear: 2026, Value: true},
	YearRow[bool]{FromYear: 2028, Value: false},
)

func childRearingLifeInsurance(premium money.Yen) money.Yen {
	switch {
	case premium <= 0:
		return 0
	case premium <= 30_000:
		return premium
	case premium <= 60_000:
		return premium.Mul(money.NewRate(1, 2), money.Truncate) + 15_000
	case premium <= 120_000:
		return premium.Mul(money.NewRate(1, 4), money.Truncate) + 30_000
	default:
		return ChildRearingLifeInsuranceCap
	}
}

const ChildRearingLifeInsuranceCap money.Yen = 60_000

func lifeInsuranceIncomeTax(premium money.Yen) money.Yen {
	switch {
	case premium <= 0:
		return 0
	case premium <= 20_000:
		return premium
	case premium <= 40_000:
		return premium.Mul(money.NewRate(1, 2), money.Truncate) + 10_000
	case premium <= 80_000:
		return premium.Mul(money.NewRate(1, 4), money.Truncate) + 20_000
	default:
		return LifeInsuranceCategoryCapIncomeTax
	}
}

func lifeInsuranceResident(premium money.Yen) money.Yen {
	switch {
	case premium <= 0:
		return 0
	case premium <= 12_000:
		return premium
	case premium <= 32_000:
		return premium.Mul(money.NewRate(1, 2), money.Truncate) + 6_000
	case premium <= 56_000:
		return premium.Mul(money.NewRate(1, 4), money.Truncate) + 14_000
	default:
		return LifeInsuranceCategoryCapResident
	}
}

func LifeInsuranceDeductionTotal(general, medical, annuity money.Yen, year date.Year, hasYoungDependant bool) Deduction {
	sum := func(pick func(Deduction) money.Yen) money.Yen {
		return pick(GeneralLifeInsuranceDeduction(general, year, hasYoungDependant)) +
			pick(LifeInsuranceDeduction(medical)) +
			pick(LifeInsuranceDeduction(annuity))
	}

	return Deduction{
		IncomeTax: min(sum(func(d Deduction) money.Yen { return d.IncomeTax }), LifeInsuranceTotalCapIncomeTax),
		Resident:  min(sum(func(d Deduction) money.Yen { return d.Resident }), LifeInsuranceTotalCapResident),
	}
}

const (
	EarthquakeCapIncomeTax money.Yen = 50_000
	EarthquakeCapResident  money.Yen = 25_000
)

func EarthquakeInsuranceDeduction(premium money.Yen) Deduction {
	if premium <= 0 {
		return Deduction{}
	}

	return Deduction{
		IncomeTax: min(premium, EarthquakeCapIncomeTax),
		Resident:  min(premium.Mul(money.NewRate(1, 2), money.Truncate), EarthquakeCapResident),
	}
}
