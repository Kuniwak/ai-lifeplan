package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

const DepreciationRateTableName = "national/depreciation-rate"

const (
	DepreciationYearsColumn tsv.ColumnName = "経過年数"
	DepreciationRateColumn  tsv.ColumnName = "経年減点補正率"
)

type DepreciationRateTable struct {
	bands relation.Bands[int, money.Rate]
}

func ParseDepreciationRateTable(table *tsv.Table) (DepreciationRateTable, error) {
	r, err := newReader(table, DepreciationRateTableName, DepreciationYearsColumn, DepreciationRateColumn)
	if err != nil {
		return DepreciationRateTable{}, fmt.Errorf("law.ParseDepreciationRateTable: %w", err)
	}

	bands := make([]relation.Band[int, money.Rate], 0, r.Rows())
	for row := range r.Rows() {
		years, err := r.Count(row, DepreciationYearsColumn)
		if err != nil {
			return DepreciationRateTable{}, fmt.Errorf("law.ParseDepreciationRateTable: %w", err)
		}
		rate, err := r.Percent(row, DepreciationRateColumn)
		if err != nil {
			return DepreciationRateTable{}, fmt.Errorf("law.ParseDepreciationRateTable: %w", err)
		}
		bands = append(bands, relation.Band[int, money.Rate]{Lower: years, Value: rate})
	}

	if len(bands) == 0 {
		return DepreciationRateTable{}, fmt.Errorf("law.ParseDepreciationRateTable: the table has no rows, so every lookup would miss")
	}

	looked := relation.NewBands(bands)
	if first, _ := looked.Min(); first != 1 {
		return DepreciationRateTable{}, fmt.Errorf(
			"law.ParseDepreciationRateTable: the table starts at %d years elapsed rather than 1, leaving the first year unaccounted for", first)
	}
	return DepreciationRateTable{bands: looked}, nil
}

func (t DepreciationRateTable) Rate(yearsElapsed int) money.Rate {
	if yearsElapsed <= 0 {
		return money.NewRate(1, 1)
	}
	return t.bands.Lookup(yearsElapsed)
}
