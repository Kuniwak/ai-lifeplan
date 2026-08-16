package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const ChildAllowanceLimitsTableName = "national/child-allowance-limits"

type ChildAllowanceLimits struct {
	IncomeLimit money.Yen

	IncomeCeiling money.Yen
}

const (
	ChildAllowanceUnderThree     money.Yen = 15_000
	ChildAllowancePrimary        money.Yen = 10_000
	ChildAllowancePrimaryThird   money.Yen = 15_000
	ChildAllowanceLowerSecondary money.Yen = 10_000

	ChildAllowanceReduced money.Yen = 5_000

	ChildAllowanceThirdOrLater money.Yen = 30_000
)

var ChildAllowanceReformCommencesIn = date.Date{Year: 2024, Month: 10, Day: 1}

const ChildAllowanceReformedFrom = 2025

const (
	childAllowanceUnderThreeMaxAge     = 2
	childAllowancePrimaryMaxAge        = 12
	childAllowanceLowerSecondaryMaxAge = 15
	childAllowanceUpperSecondaryMaxAge = 18
)

const ChildAllowanceSiblingMaxAge = 22

func ChildAllowanceCountsTowardsThirdOrLater(year date.Year, born date.Date) bool {
	maxAge := childAllowanceUpperSecondaryMaxAge
	if year >= ChildAllowanceReformedFrom {
		maxAge = ChildAllowanceSiblingMaxAge
	}
	return year <= born.ReachesAge(maxAge).SchoolYearEnd().Year
}

const (
	ChildAllowanceDependentsColumn tsv.ColumnName = "扶養親族等の数"
	ChildAllowanceLimitColumn      tsv.ColumnName = "所得制限限度額[円]"
	ChildAllowanceCeilingColumn    tsv.ColumnName = "所得上限限度額[円]"
)

const childAllowanceLimitStep money.Yen = 380_000

type ChildAllowanceTable struct {
	limits []ChildAllowanceLimits

	from date.Year
}

func ParseChildAllowanceTable(table *tsv.Table) (ChildAllowanceTable, error) {
	r, err := newReader(table, ChildAllowanceLimitsTableName,
		ChildAllowanceDependentsColumn, ChildAllowanceLimitColumn, ChildAllowanceCeilingColumn,
		LawStartYearColumn)
	if err != nil {
		return ChildAllowanceTable{}, fmt.Errorf("law.ParseChildAllowanceTable: %w", err)
	}

	limits := make([]ChildAllowanceLimits, 0, r.Rows())
	var from date.Year
	for row := range r.Rows() {
		start, err := r.startYear(row)
		if err != nil {
			return ChildAllowanceTable{}, fmt.Errorf("law.ParseChildAllowanceTable: %w", err)
		}
		if row == 0 {
			from = start
		} else if start != from {
			return ChildAllowanceTable{}, fmt.Errorf(
				"law.ParseChildAllowanceTable: row %d starts in %d but row 1 starts in %d; every band of one wording starts together",
				row+1, start, from)
		}

		dependents, err := r.Count(row, ChildAllowanceDependentsColumn)
		if err != nil {
			return ChildAllowanceTable{}, fmt.Errorf("law.ParseChildAllowanceTable: %w", err)
		}
		if dependents != row {
			return ChildAllowanceTable{}, fmt.Errorf(
				"law.ParseChildAllowanceTable: row %d is for %d dependents; the rows have to run from 0 upwards without a gap", row+1, dependents)
		}

		limit, err := r.Yen(row, ChildAllowanceLimitColumn)
		if err != nil {
			return ChildAllowanceTable{}, fmt.Errorf("law.ParseChildAllowanceTable: %w", err)
		}
		ceiling, err := r.Yen(row, ChildAllowanceCeilingColumn)
		if err != nil {
			return ChildAllowanceTable{}, fmt.Errorf("law.ParseChildAllowanceTable: %w", err)
		}
		if ceiling < limit {
			return ChildAllowanceTable{}, fmt.Errorf(
				"law.ParseChildAllowanceTable: row %d has a ceiling of %d below its limit of %d, which leaves 特例給付 no band to be paid in", row+1, ceiling, limit)
		}
		limits = append(limits, ChildAllowanceLimits{IncomeLimit: limit, IncomeCeiling: ceiling})
	}

	if len(limits) == 0 {
		return ChildAllowanceTable{}, fmt.Errorf("law.ParseChildAllowanceTable: the table has no rows, so every lookup would miss")
	}
	return ChildAllowanceTable{limits: limits, from: from}, nil
}

func (t ChildAllowanceTable) Limits(year date.Year, dependents int) (ChildAllowanceLimits, bool) {
	if year < t.from || year >= ChildAllowanceReformedFrom {
		return ChildAllowanceLimits{}, false
	}
	return t.limitsFor(dependents), true
}

func (t ChildAllowanceTable) limitsFor(dependents int) ChildAllowanceLimits {
	if dependents < 0 {
		dependents = 0
	}
	if dependents < len(t.limits) {
		return t.limits[dependents]
	}

	last := t.limits[len(t.limits)-1]
	extra := money.Yen(dependents-(len(t.limits)-1)) * childAllowanceLimitStep
	return ChildAllowanceLimits{
		IncomeLimit:   last.IncomeLimit + extra,
		IncomeCeiling: last.IncomeCeiling + extra,
	}
}

func ChildAllowanceMonthlyAfterReform(underThree, thirdOrLater bool) money.Yen {
	switch {
	case thirdOrLater:
		return ChildAllowanceThirdOrLater
	case underThree:
		return ChildAllowanceUnderThree
	default:
		return ChildAllowancePrimary
	}
}

func (t ChildAllowanceTable) MonthlyBeforeReform(age int, thirdOrLater bool, income money.Yen, dependents int) money.Yen {
	if age > childAllowanceLowerSecondaryMaxAge {
		return 0
	}

	limits := t.limitsFor(dependents)
	switch {
	case income >= limits.IncomeCeiling:
		return 0
	case income >= limits.IncomeLimit:
		return ChildAllowanceReduced
	}

	switch {
	case age <= childAllowanceUnderThreeMaxAge:
		return ChildAllowanceUnderThree
	case age <= childAllowancePrimaryMaxAge && thirdOrLater:
		return ChildAllowancePrimaryThird
	case age <= childAllowancePrimaryMaxAge:
		return ChildAllowancePrimary
	default:
		return ChildAllowanceLowerSecondary
	}
}

type ChildAllowanceMonths struct {
	UnderThree int
	Older      int
}

func (m ChildAllowanceMonths) Total() int { return m.UnderThree + m.Older }

func ChildAllowanceMonthsIn(year date.Year, born date.Date) ChildAllowanceMonths {
	return childAllowanceMonthsFrom(year, born, 1)
}

func childAllowanceMonthsFrom(year date.Year, born date.Date, firstMonth int) ChildAllowanceMonths {
	last := lastPaidMonth(born)
	if year > last.Year || year < born.Year {
		return ChildAllowanceMonths{}
	}

	from := max(1, firstMonth)
	if year == born.Year {
		from = max(from, born.Month+1)
	}
	to := 12
	if year == last.Year {
		to = last.Month
	}

	stepsDown := born.Anniversary(3)
	var months ChildAllowanceMonths
	for m := from; m <= to; m++ {
		if year < stepsDown.Year || (year == stepsDown.Year && m <= stepsDown.Month) {
			months.UnderThree++
			continue
		}
		months.Older++
	}
	return months
}

func lastPaidMonth(born date.Date) date.Date {
	return born.ReachesAge(childAllowanceUpperSecondaryMaxAge).SchoolYearEnd()
}

func (t ChildAllowanceTable) Yearly(year date.Year, born date.Date, thirdOrLater bool, income money.Yen, dependents int) money.Yen {
	age := int(year - born.Year)
	before := func(months int) money.Yen {
		return t.MonthlyBeforeReform(age, thirdOrLater, income, dependents) * money.Yen(months)
	}
	after := func(from int) money.Yen {
		months := childAllowanceMonthsFrom(year, born, from)
		return ChildAllowanceMonthlyAfterReform(true, thirdOrLater)*money.Yen(months.UnderThree) +
			ChildAllowanceMonthlyAfterReform(false, thirdOrLater)*money.Yen(months.Older)
	}

	switch {
	case year < ChildAllowanceReformCommencesIn.Year:
		return before(date.MonthsAYear)
	case year > ChildAllowanceReformCommencesIn.Year:
		return after(1)
	}

	return before(ChildAllowanceReformCommencesIn.Month-1) + after(ChildAllowanceReformCommencesIn.Month)
}
