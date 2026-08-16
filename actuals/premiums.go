package actuals

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type PremiumsDeducted struct {
	HealthOnPay, HealthOnBonus money.Yen

	PensionOnPay, PensionOnBonus money.Yen

	Employment money.Yen
}

func (p PremiumsDeducted) Health() money.Yen  { return p.HealthOnPay + p.HealthOnBonus }
func (p PremiumsDeducted) Pension() money.Yen { return p.PensionOnPay + p.PensionOnBonus }
func (p PremiumsDeducted) Total() money.Yen   { return p.Health() + p.Pension() + p.Employment }

func PremiumsRecordedByYear(slips []Payslip, person string) relation.Table[PremiumsDeducted] {
	theirs := make([]Payslip, 0, len(slips))
	for _, slip := range slips {
		if slip.Person == person {
			theirs = append(theirs, slip)
		}
	}

	deducted := PremiumsDeductedByYear(theirs)
	rows := make([]relation.Row[PremiumsDeducted], 0, len(deducted))
	for year := range YearsWhollyCovered(theirs) {
		rows = append(rows, relation.Row[PremiumsDeducted]{Year: year, Value: deducted[year]})
	}
	return relation.New(rows)
}

func PremiumsDeductedByYear(slips []Payslip) map[date.Year]PremiumsDeducted {
	deducted := make(map[date.Year]PremiumsDeducted, 8)
	for _, slip := range slips {
		sum := deducted[slip.Year]
		switch slip.Kind {
		case PayslipSalary:
			sum.HealthOnPay += slip.Health
			sum.PensionOnPay += slip.Pension
		case PayslipBonus:
			sum.HealthOnBonus += slip.Health
			sum.PensionOnBonus += slip.Pension
		default:
			continue
		}
		sum.Employment += slip.Employment
		deducted[slip.Year] = sum
	}
	return deducted
}

func YearsWhollyCovered(slips []Payslip) map[date.Year]bool {
	paid := make(map[date.Year]date.Months, 8)
	for _, slip := range slips {
		if slip.Kind != PayslipSalary {
			continue
		}
		paid[slip.Year] = paid[slip.Year].Union(date.MonthOnly(slip.Month))
	}

	covered := make(map[date.Year]bool, len(paid))
	for year, months := range paid {
		if months == date.WholeYear {
			covered[year] = true
		}
	}
	return covered
}
