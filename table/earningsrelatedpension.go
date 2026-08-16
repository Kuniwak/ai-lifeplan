package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type EarningsRelatedPensionInput struct {
	Recorded []law.Remuneration

	RecordedThrough law.Remuneration

	Pay    relation.Table[Pay]
	Grades law.StandardRemunerationTable

	Rates     law.PensionRevaluationRates
	Published date.Year

	DeferredMonths int

	RevaluedFrom date.Year

	Calendar relation.Table[CalendarRow]
	Person   PersonName
}

func EarningsRelatedPension(in EarningsRelatedPensionInput) (money.Yen, error) {
	months := make([]law.Remuneration, 0, len(in.Recorded)+256)
	months = append(months, in.Recorded...)

	var faceValue money.Yen

	after := func(y date.Year, m int) bool {
		if y != in.RecordedThrough.Year {
			return y > in.RecordedThrough.Year
		}
		return m > in.RecordedThrough.Month
	}

	bornOn, err := BornOnIn(in.Calendar, in.Person)
	if err != nil {
		return 0, fmt.Errorf("table.EarningsRelatedPension: %w", err)
	}

	for _, row := range in.Pay.Rows() {
		if EmployeeCoverIn(row.Year, bornOn, row.Value) != law.EmployeesHealthInsurance {
			continue
		}
		standard := in.Grades.Lookup(row.Value.Monthly())

		for month := 1; month <= date.MonthsAYear; month++ {
			if !after(row.Year, month) {
				continue
			}
			if row.Year >= in.RevaluedFrom {
				faceValue += standard
				continue
			}
			months = append(months, law.Remuneration{Year: row.Year, Month: month, Amount: standard})
		}

		const bonusMonth = 6
		if !after(row.Year, bonusMonth) {
			continue
		}
		for range row.Value.BonusesAYear {
			bonus := law.PensionStandardBonus(row.Value.BonusPayment())
			if row.Year >= in.RevaluedFrom {
				faceValue += bonus
				continue
			}
			months = append(months, law.Remuneration{Year: row.Year, Month: bonusMonth, Amount: bonus})
		}
	}

	revalued, err := in.Rates.EarningsRelatedPension(in.Published, months)
	if err != nil {
		return 0, fmt.Errorf("table.EarningsRelatedPension: %w", err)
	}
	increase, err := law.PensionStartAdjustment(in.DeferredMonths)
	if err != nil {
		return 0, fmt.Errorf("table.EarningsRelatedPension: %w", err)
	}
	before := revalued + faceValue.Mul(law.EarningsRelatedMultiplierAfterTotalRemuneration, money.HalfUp)
	return before.Mul(increase, money.HalfUp), nil
}
