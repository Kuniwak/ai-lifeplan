package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type BasicPensionInput struct {
	Full money.Yen

	PaidMonths int

	DeferredMonths int
}

func BasicPension(in BasicPensionInput) (money.Yen, error) {
	before, err := law.OldAgeBasicPension(in.Full, in.PaidMonths)
	if err != nil {
		return 0, fmt.Errorf("table.BasicPension: %w", err)
	}
	increase, err := law.PensionStartAdjustment(in.DeferredMonths)
	if err != nil {
		return 0, fmt.Errorf("table.BasicPension: %w", err)
	}
	return before.Mul(increase, money.HalfUp), nil
}

type NationalPensionPaidMonthsInput struct {
	PaidInTheRecord int
	RecordedThrough date.Date

	Monthly []actuals.Remuneration

	Calendar relation.Table[CalendarRow]
	Person   PersonName
}

func NationalPensionPaidMonths(in NationalPensionPaidMonthsInput) (int, error) {
	born, err := BornOnIn(in.Calendar, in.Person)
	if err != nil {
		return 0, fmt.Errorf("table.NationalPensionPaidMonths: %w", err)
	}

	twenty := date.FirstOfMonth(law.NationalPensionFirstCategoryFrom(born))
	seam := date.FirstOfMonth(in.RecordedThrough)

	recorded := in.PaidInTheRecord
	for _, one := range in.Monthly {
		if one.Person != string(in.Person) || one.Bonus || !one.Known {
			continue
		}
		month := date.Date{Year: one.Year, Month: one.Month, Day: 1}
		if !month.Before(twenty) && month.Before(seam) {
			recorded++
		}
	}

	start := seam
	if start.Before(twenty) {
		start = twenty
	}
	through := law.NationalPensionFirstCategoryThrough(born)

	projected := max(date.MonthsBetween(start, through), 0)

	return min(recorded+projected, law.BasicPensionFullMonths), nil
}

type EmployeePensionMonthsInput struct {
	Recorded        int
	RecordedThrough date.Date

	Pay      relation.Table[Pay]
	Calendar relation.Table[CalendarRow]
	Person   PersonName
}

func EmployeePensionMonths(in EmployeePensionMonthsInput) (int, error) {
	bornOn, err := BornOnIn(in.Calendar, in.Person)
	if err != nil {
		return 0, fmt.Errorf("table.EmployeePensionMonths: %w", err)
	}

	months := in.Recorded
	for _, row := range in.Pay.Rows() {
		if row.Year < in.RecordedThrough.Year {
			continue
		}
		if EmployeeCoverIn(row.Year, bornOn, row.Value) != law.EmployeesHealthInsurance {
			continue
		}
		if row.Year == in.RecordedThrough.Year {
			months += date.MonthsAYear - in.RecordedThrough.Month + 1
			continue
		}
		months += date.MonthsAYear
	}
	return months, nil
}
