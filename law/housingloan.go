package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

const HousingLoanCreditTableName = "national/housing-loan-credit"

const (
	HousingLoanRateColumn        tsv.ColumnName = "控除率"
	HousingLoanYearsColumn       tsv.ColumnName = "控除期間[年]"
	HousingLoanCeilingColumn     tsv.ColumnName = "借入限度額[円]"
	HousingLoanIncomeLimitColumn tsv.ColumnName = "合計所得金額上限[円]"
	HousingLoanFromYearColumn    tsv.ColumnName = "居住開始年"
)

const HousingLoanCreditUnit money.Yen = 100

type HousingLoanTerms struct {
	Rate        money.Rate
	Years       int
	Ceiling     money.Yen
	IncomeLimit money.Yen
}

type HousingLoanCreditTable struct {
	terms relation.Periods[date.Year, HousingLoanTerms]
}

func ParseHousingLoanCreditTable(table *tsv.Table) (HousingLoanCreditTable, error) {
	r, err := newReader(table, HousingLoanCreditTableName, HousingLoanFromYearColumn, HousingLoanRateColumn,
		HousingLoanYearsColumn, HousingLoanCeilingColumn, HousingLoanIncomeLimitColumn, LawEndYearColumn)
	if err != nil {
		return HousingLoanCreditTable{}, fmt.Errorf("law.ParseHousingLoanCreditTable: %w", err)
	}

	rows := make([]relation.Period[date.Year, HousingLoanTerms], 0, r.Rows())
	for row := range r.Rows() {
		from, err := r.Year(row, HousingLoanFromYearColumn)
		if err != nil {
			return HousingLoanCreditTable{}, fmt.Errorf("law.ParseHousingLoanCreditTable: %w", err)
		}
		rate, err := r.Percent(row, HousingLoanRateColumn)
		if err != nil {
			return HousingLoanCreditTable{}, fmt.Errorf("law.ParseHousingLoanCreditTable: %w", err)
		}
		years, err := r.Count(row, HousingLoanYearsColumn)
		if err != nil {
			return HousingLoanCreditTable{}, fmt.Errorf("law.ParseHousingLoanCreditTable: %w", err)
		}
		ceiling, err := r.Yen(row, HousingLoanCeilingColumn)
		if err != nil {
			return HousingLoanCreditTable{}, fmt.Errorf("law.ParseHousingLoanCreditTable: %w", err)
		}
		limit, err := r.Yen(row, HousingLoanIncomeLimitColumn)
		if err != nil {
			return HousingLoanCreditTable{}, fmt.Errorf("law.ParseHousingLoanCreditTable: %w", err)
		}

		through, err := r.endBound(row)
		if err != nil {
			return HousingLoanCreditTable{}, fmt.Errorf("law.ParseHousingLoanCreditTable: %w", err)
		}

		rows = append(rows, relation.NewPeriod(relation.From(from), through, HousingLoanTerms{
			Rate: rate, Years: years, Ceiling: ceiling, IncomeLimit: limit,
		}))
	}

	if len(rows) == 0 {
		return HousingLoanCreditTable{}, fmt.Errorf("law.ParseHousingLoanCreditTable: the table has no rows, so every lookup would miss")
	}
	terms, err := relation.NewPeriods(rows)
	if err != nil {
		return HousingLoanCreditTable{}, fmt.Errorf("law.ParseHousingLoanCreditTable: %w", err)
	}
	return HousingLoanCreditTable{terms: terms}, nil
}

func (t HousingLoanCreditTable) Terms(movedIn date.Year) (HousingLoanTerms, bool) {
	return t.terms.Lookup(movedIn)
}

func (t HousingLoanCreditTable) Credit(balance, totalIncome money.Yen, movedIn date.Year, year date.Year) money.Yen {
	terms, ok := t.Terms(movedIn)
	if !ok {
		return 0
	}
	if year < movedIn || year >= movedIn+date.Year(terms.Years) {
		return 0
	}
	if totalIncome > terms.IncomeLimit {
		return 0
	}

	counted := min(max(balance, 0), terms.Ceiling)
	return counted.Mul(terms.Rate, money.Truncate).Truncate(HousingLoanCreditUnit)
}
