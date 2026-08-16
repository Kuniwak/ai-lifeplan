package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/stepfn"
	"github.com/Kuniwak/lifeplan/tsv"
)

const DependantAgeLimit = law.SalaryIncomeAdjustmentDependantAge

func IncomeInputFor(
	tables map[tsv.Slot]*tsv.Table,
	calendar relation.Table[CalendarRow],
	person PersonName, paySlot tsv.Slot,
) (IncomeInput, error) {
	var in IncomeInput

	years := calendar.Years()
	if len(years) == 0 {
		return in, fmt.Errorf("table.IncomeInputFor: the calendar is empty, so there are no years to work over")
	}
	from, to := years[0], years[len(years)-1]

	pay, err := readPay(tables[paySlot], paySlot, from, to)
	if err != nil {
		return in, err
	}
	in.Pay = pay

	if in.Pension, err = readPension(tables[input.PensionSlot], person); err != nil {
		return in, err
	}

	if in.PriceLevels, err = PriceLevelsFrom(tables, from, to); err != nil {
		return in, err
	}
	if in.WageLevels, err = WageLevelsFrom(tables, from, to); err != nil {
		return in, err
	}

	ages := make([]relation.Row[int], 0, len(years))
	young := make([]relation.Row[bool], 0, len(years))
	for _, y := range years {
		row, _ := calendar.At(y)

		age, ok := row.AgeOf(person)
		if !ok {
			return in, fmt.Errorf("table.IncomeInputFor: %q is not in the household", person)
		}
		ages = append(ages, relation.Row[int]{Year: y, Value: age})
		young = append(young, relation.Row[bool]{Year: y, Value: hasYoungDependant(row)})
	}
	in.Age = relation.New(ages)
	in.HasYoungDependant = relation.New(young)

	return in, nil
}

func hasYoungDependant(row CalendarRow) bool {
	for _, p := range row.Ages {
		if !p.IsChild() {
			continue
		}
		if p.Age >= 0 && p.Age < DependantAgeLimit {
			return true
		}
	}
	return false
}

func readPay(table *tsv.Table, slot tsv.Slot, from, to date.Year) (relation.Table[Pay], error) {
	var empty relation.Table[Pay]

	columns := []tsv.ColumnName{input.YearColumn, input.AnnualSalaryColumn, input.BonusColumn, input.BonusesAYearColumn, input.LeaveMonthsColumn}

	_, hasBusiness := table.ColumnIndex(input.BusinessReceiptsColumn)
	if hasBusiness {
		columns = append(columns,
			input.BusinessReceiptsColumn, input.BusinessExpensesColumn, input.BlueFormRecordKeepingColumn)
	}
	_, hasMiscellaneous := table.ColumnIndex(input.MiscellaneousReceiptsColumn)
	if hasMiscellaneous {
		columns = append(columns, input.MiscellaneousReceiptsColumn)
	}
	_, hasHours := table.ColumnIndex(input.WeeklyHoursColumn)
	if hasHours {
		columns = append(columns,
			input.WeeklyHoursColumn, input.NormalWeeklyHoursColumn, input.SpecifiedWorkplaceColumn)
	}
	_, hasExempt := table.ColumnIndex(input.ExemptMonthsColumn)
	if hasExempt {
		columns = append(columns, input.ExemptMonthsColumn)
	}

	r, err := tsv.NewReader(table, slot, columns...)
	if err != nil {
		return empty, err
	}

	written := make([]relation.Row[Pay], 0, r.Rows())
	for row := range r.Rows() {
		year, err := r.Year(row, input.YearColumn)
		if err != nil {
			return empty, err
		}
		salary, err := r.Yen(row, input.AnnualSalaryColumn)
		if err != nil {
			return empty, err
		}
		bonus, err := r.Yen(row, input.BonusColumn)
		if err != nil {
			return empty, err
		}
		bonuses, err := r.Count(row, input.BonusesAYearColumn)
		if err != nil {
			return empty, err
		}
		leave, err := r.Count(row, input.LeaveMonthsColumn)
		if err != nil {
			return empty, err
		}

		value := Pay{
			Salary:       salary,
			Bonus:        bonus,
			BonusesAYear: bonuses,
			LeaveMonths:  leave,
		}
		if hasBusiness {
			if value.BusinessReceipts, err = r.Yen(row, input.BusinessReceiptsColumn); err != nil {
				return empty, err
			}
			if value.BusinessExpenses, err = r.Yen(row, input.BusinessExpensesColumn); err != nil {
				return empty, err
			}
			value.BlueFormRecordKeeping = law.BlueFormRecordKeeping(r.Field(row, input.BlueFormRecordKeepingColumn))
		}
		if hasMiscellaneous {
			if value.MiscellaneousReceipts, err = r.Yen(row, input.MiscellaneousReceiptsColumn); err != nil {
				return empty, err
			}
		}
		if hasHours {
			if value.WeeklyHours, err = r.Count(row, input.WeeklyHoursColumn); err != nil {
				return empty, err
			}
			if value.Workplace.NormalWeeklyHours, err = r.Count(row, input.NormalWeeklyHoursColumn); err != nil {
				return empty, err
			}
			value.Workplace.Specified = r.Field(row, input.SpecifiedWorkplaceColumn) == input.SpecifiedWorkplace
		}
		if hasExempt {
			if value.ExemptMonths, err = r.Months(row, input.ExemptMonthsColumn); err != nil {
				return empty, err
			}
		}
		written = append(written, relation.Row[Pay]{Year: year, Value: value})
	}

	expanded, err := stepfn.Expand(written, from, to)
	if err != nil {
		return empty, fmt.Errorf("table: %s: %w", slot, err)
	}
	return expanded, nil
}

func readPension(table *tsv.Table, person PersonName) (Pension, error) {
	r, err := tsv.NewReader(table, input.PensionSlot,
		input.PersonColumn, input.PensionStartColumn, input.PensionExpectedColumn)
	if err != nil {
		return Pension{}, err
	}

	for row := range r.Rows() {
		if PersonName(r.Field(row, input.PersonColumn)) != person {
			continue
		}

		start, err := r.Year(row, input.PensionStartColumn)
		if err != nil {
			return Pension{}, err
		}
		expected, err := r.Percent(row, input.PensionExpectedColumn)
		if err != nil {
			return Pension{}, err
		}

		return Pension{StartYear: start, Expected: expected}, nil
	}

	return Pension{}, fmt.Errorf("table: %s: nothing is written about %q's pension", input.PensionSlot, person)
}
