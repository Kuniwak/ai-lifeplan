package law

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

const (
	SpecialTaxCreditIncomeYear = 2024

	SpecialTaxCreditPerPerson money.Yen = 30_000

	SpecialTaxCreditIncomeCeiling money.Yen = 18_050_000
)

func SpecialTaxCredit(incomeYear date.Year, totalIncome money.Yen, sameLivelihoodDependants int) money.Yen {
	if incomeYear != SpecialTaxCreditIncomeYear {
		return 0
	}
	if totalIncome > SpecialTaxCreditIncomeCeiling {
		return 0
	}
	return SpecialTaxCreditPerPerson * money.Yen(1+sameLivelihoodDependants)
}

const (
	ResidentSpecialTaxCreditLevyYear = 2024

	ResidentSpecialTaxCreditSpouseLevyYear = 2025

	ResidentSpecialTaxCreditPerPerson money.Yen = 10_000
)

func ResidentSpecialTaxCredit(levyYear date.Year, priorYearTotalIncome money.Yen, deductibleDependants int) money.Yen {
	if levyYear != ResidentSpecialTaxCreditLevyYear {
		return 0
	}
	if priorYearTotalIncome > SpecialTaxCreditIncomeCeiling {
		return 0
	}
	return ResidentSpecialTaxCreditPerPerson * money.Yen(1+deductibleDependants)
}

func ResidentSpecialTaxCreditForSpouse(levyYear date.Year, priorYearTotalIncome money.Yen, hasSpouseOutsideTheDeduction bool) money.Yen {
	if levyYear != ResidentSpecialTaxCreditSpouseLevyYear {
		return 0
	}
	if !hasSpouseOutsideTheDeduction {
		return 0
	}
	if priorYearTotalIncome > SpecialTaxCreditIncomeCeiling {
		return 0
	}
	return ResidentSpecialTaxCreditPerPerson
}

func TaxpayerMayClaimASpouse(taxpayerIncome money.Yen, incomeYear date.Year) bool {
	return taxpayerTier(taxpayerIncome, incomeYear) >= 0
}

func SplitResidentSpecialTaxCredit(credit, prefecturalLevy, municipalLevy money.Yen) ResidentTaxCredits {
	levy := prefecturalLevy + municipalLevy
	if credit >= levy {
		return ResidentTaxCredits{Prefectural: prefecturalLevy, Municipal: municipalLevy}
	}
	prefectural := (credit*prefecturalLevy + levy - 1) / levy
	return ResidentTaxCredits{Prefectural: prefectural, Municipal: credit - prefectural}
}
