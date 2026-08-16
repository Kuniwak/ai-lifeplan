package law

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const NationalPensionPremiumTableName = "national/national-pension-premium"

const NationalPensionPremiumColumn tsv.ColumnName = "保険料月額[円]"

type NationalPensionPremiumTable struct {
	YearYenTable
}

func (t NationalPensionPremiumTable) Monthly(fiscalYear date.Year) money.Yen {
	return t.Amount(fiscalYear)
}

const (
	NationalPensionFromAge = 20
	NationalPensionToAge   = 59
)

type NationalPensionCategory int

const (
	NotNationalPensionInsured NationalPensionCategory = iota

	FirstCategoryInsured

	SecondCategoryInsured

	ThirdCategoryInsured
)

func NationalPensionCategoryOf(age int, cover Cover, isInsured, isSpouseOfInsured bool) NationalPensionCategory {
	if cover == EmployeesHealthInsurance && isInsured {
		return SecondCategoryInsured
	}
	if age < NationalPensionFromAge || age > NationalPensionToAge {
		return NotNationalPensionInsured
	}
	if cover == EmployeesHealthInsurance {
		if isSpouseOfInsured {
			return ThirdCategoryInsured
		}
		return FirstCategoryInsured
	}
	if cover == NationalHealthInsurance {
		return FirstCategoryInsured
	}
	return NotNationalPensionInsured
}

func NationalPensionFirstCategoryFrom(born date.Date) date.Date {
	return born.ReachesAge(NationalPensionFromAge)
}

func NationalPensionFirstCategoryThrough(born date.Date) date.Date {
	lost := born.Anniversary(NationalPensionToAge + 1)
	return date.Date{Year: lost.Year, Month: lost.Month, Day: 1}.DayBefore()
}

func NationalPensionMonthsIn(year date.Year, born date.Date, cover Cover, isInsured, isSpouseOfInsured bool) date.Months {
	switch cover {
	case NationalHealthInsurance:
	case EmployeesHealthInsurance:
		if isInsured || isSpouseOfInsured {
			return date.NoMonths
		}
	default:
		return date.NoMonths
	}
	return date.MonthsOfYearIn(year,
		NationalPensionFirstCategoryFrom(born), NationalPensionFirstCategoryThrough(born))
}
