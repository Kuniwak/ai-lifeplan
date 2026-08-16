package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

const StandardRemunerationPensionTableName = "national/standard-remuneration-pension"

const StandardRemunerationHealthTableName = "national/standard-remuneration-health"

const (
	StandardRemunerationLowerColumn tsv.ColumnName = "報酬月額下限[円]"
	StandardRemunerationValueColumn tsv.ColumnName = "標準報酬月額[円]"
)

type StandardRemunerationBand struct {
	LowerBound money.Yen

	Standard money.Yen
}

type StandardRemunerationTable struct {
	bands relation.Bands[money.Yen, money.Yen]
	rows  []StandardRemunerationBand
}

func ParseStandardRemunerationTable(table *tsv.Table) (StandardRemunerationTable, error) {
	lower, ok := table.ColumnIndex(StandardRemunerationLowerColumn)
	if !ok {
		return StandardRemunerationTable{}, fmt.Errorf("law.ParseStandardRemunerationTable: the column %q is missing", StandardRemunerationLowerColumn)
	}
	value, ok := table.ColumnIndex(StandardRemunerationValueColumn)
	if !ok {
		return StandardRemunerationTable{}, fmt.Errorf("law.ParseStandardRemunerationTable: the column %q is missing", StandardRemunerationValueColumn)
	}

	rows := make([]StandardRemunerationBand, 0, len(table.Rows))
	bands := make([]relation.Band[money.Yen, money.Yen], 0, len(table.Rows))
	seen := make(map[money.Yen]int, len(table.Rows))
	for i, fields := range table.Rows {

		bound, err := money.ParseYen(fields[lower])
		if err != nil {
			return StandardRemunerationTable{}, fmt.Errorf("law.ParseStandardRemunerationTable: row %d, %q: %w", i+1, StandardRemunerationLowerColumn, err)
		}
		standard, err := money.ParseYen(fields[value])
		if err != nil {
			return StandardRemunerationTable{}, fmt.Errorf("law.ParseStandardRemunerationTable: row %d, %q: %w", i+1, StandardRemunerationValueColumn, err)
		}

		if first, repeated := seen[bound]; repeated {
			return StandardRemunerationTable{}, fmt.Errorf(
				"law.ParseStandardRemunerationTable: row %d starts at %d, which row %d already starts at. "+
					"この表は年ではなく報酬月額で引くので、二つの版が混ざると両方の等級から答えることになる",
				i+1, bound, first)
		}
		seen[bound] = i + 1

		rows = append(rows, StandardRemunerationBand{LowerBound: bound, Standard: standard})
		bands = append(bands, relation.Band[money.Yen, money.Yen]{Lower: bound, Value: standard})
	}

	if len(bands) == 0 {
		return StandardRemunerationTable{}, fmt.Errorf("law.ParseStandardRemunerationTable: the table has no grades, so every lookup would miss")
	}

	looked := relation.NewBands(bands)
	if lowest, _ := looked.Min(); lowest != 0 {
		return StandardRemunerationTable{}, fmt.Errorf(
			"law.ParseStandardRemunerationTable: the lowest grade starts at %d rather than 0, so a 報酬月額 below it would fall through the table", lowest)
	}

	return StandardRemunerationTable{bands: looked, rows: rows}, nil
}

func (t StandardRemunerationTable) Lookup(monthlyPay money.Yen) money.Yen {
	return t.bands.Lookup(monthlyPay)
}

func (t StandardRemunerationTable) Bands() []StandardRemunerationBand {
	return t.rows
}

var pensionStandardRemunerationCeiling = relation.NewBands([]relation.Band[int, money.Yen]{
	{Lower: relation.MonthsSince(0, 1), Value: 620_000},
	{Lower: relation.MonthsSince(2020, 9), Value: 650_000},
})

func PensionStandardRemunerationCeiling(year date.Year, month int) money.Yen {
	return pensionStandardRemunerationCeiling.Lookup(relation.MonthsSince(year, month))
}

func PensionStandardRemunerationCeilingOnPayslip(year date.Year, month int) money.Yen {
	return PensionStandardRemunerationCeiling(MonthCovered(year, month))
}

func MonthCovered(payslipYear date.Year, payslipMonth int) (date.Year, int) {
	return relation.YearMonthOf(relation.MonthsSince(payslipYear, payslipMonth) - 1)
}

const RegularDecisionMonth = 9

func RegularDecisionYearOnPayslip(payslipYear date.Year, payslipMonth int) date.Year {
	covered, month := MonthCovered(payslipYear, payslipMonth)
	if month >= RegularDecisionMonth {
		return covered
	}
	return covered - 1
}

func RateFiscalYearOnPayslip(payslipYear date.Year, payslipMonth int) date.Year {
	covered, month := MonthCovered(payslipYear, payslipMonth)
	if month >= 3 {
		return covered
	}
	return covered - 1
}
