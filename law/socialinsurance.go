package law

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const HealthInsuranceRateTableName = "national/health-insurance-rate"

const StandardBonusUnit money.Yen = 1000

func StandardBonus(gross money.Yen) money.Yen {
	if gross <= 0 {
		return 0
	}
	return gross.Truncate(StandardBonusUnit)
}

const (
	PensionStandardBonusCap money.Yen = 1_500_000

	HealthStandardBonusCapFiscalYear money.Yen = 5_730_000
)

func PensionStandardBonus(gross money.Yen) money.Yen {
	return min(StandardBonus(gross), PensionStandardBonusCap)
}

func HealthStandardBonus(paidSoFarInFiscalYear, gross money.Yen) money.Yen {
	remaining := max(HealthStandardBonusCapFiscalYear-paidSoFarInFiscalYear, 0)
	return min(StandardBonus(gross), remaining)
}

const (
	HealthInsuranceRateColumn tsv.ColumnName = "健康保険料率"
	NursingCareRateColumn     tsv.ColumnName = "介護保険料率"
)

var PensionRateInsured = money.NewRate(915, 10_000)

type SocialInsuranceRates struct {
	Health YearRateTable

	NursingCare YearRateTable

	growth CostGrowth
}

func (r SocialInsuranceRates) WithGrowth(g CostGrowth) SocialInsuranceRates {
	g.AssertStated()
	r.growth = g
	return r
}

func ParseSocialInsuranceRates(table *tsv.Table) (SocialInsuranceRates, error) {
	health, err := ParseYearRateTable(table, HealthInsuranceRateTableName, HealthInsuranceRateColumn)
	if err != nil {
		return SocialInsuranceRates{}, err
	}
	nursing, err := ParseYearRateTable(table, HealthInsuranceRateTableName, NursingCareRateColumn)
	if err != nil {
		return SocialInsuranceRates{}, err
	}
	return SocialInsuranceRates{Health: health, NursingCare: nursing}, nil
}

func (r SocialInsuranceRates) HealthPremium(standardAmount money.Yen, year date.Year) money.Yen {
	return r.growth.Medical.GrowPremium(insuredPremium(standardAmount, r.Health.Rate(year)), r.Health.LastWrittenYear, year)
}

func (r SocialInsuranceRates) NursingCarePremium(standardAmount money.Yen, year date.Year) money.Yen {
	return r.growth.Care.GrowPremium(insuredPremium(standardAmount, r.NursingCare.Rate(year)), r.NursingCare.LastWrittenYear, year)
}

func PensionInsurancePremium(standardAmount money.Yen) money.Yen {
	return insuredPremium(standardAmount, PensionRateInsured)
}

const (
	NursingCareAgeMin = 40
	NursingCareAgeMax = 64
)

func NursingCareInsured(age int) bool {
	return NursingCareAgeMin <= age && age <= NursingCareAgeMax
}

func insuredPremium(standardAmount money.Yen, rate money.Rate) money.Yen {
	return standardAmount.Mul(rate, money.Truncate)
}
