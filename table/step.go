package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/stepfn"
	"github.com/Kuniwak/lifeplan/tsv"
)

type parser[T any] func(tsv.Reader, int, tsv.ColumnName) (T, error)

func readStep[T any](
	table *tsv.Table, slot tsv.Slot, column tsv.ColumnName, from, to date.Year, value parser[T],
) (relation.Table[T], error) {
	var empty relation.Table[T]

	r, err := tsv.NewReader(table, slot, input.YearColumn, column)
	if err != nil {
		return empty, err
	}

	written := make([]relation.Row[T], 0, r.Rows())
	for row := range r.Rows() {
		year, err := r.Year(row, input.YearColumn)
		if err != nil {
			return empty, err
		}
		v, err := value(r, row, column)
		if err != nil {
			return empty, err
		}
		written = append(written, relation.Row[T]{Year: year, Value: v})
	}

	expanded, err := stepfn.Expand(written, from, to)
	if err != nil {
		return empty, fmt.Errorf("table: %s: %w", slot, err)
	}
	return expanded, nil
}

func ReadYenStep(table *tsv.Table, slot tsv.Slot, column tsv.ColumnName, from, to date.Year) (relation.Table[money.Yen], error) {
	return readStep(table, slot, column, from, to, tsv.Reader.Yen)
}

func readRateStep(table *tsv.Table, slot tsv.Slot, column tsv.ColumnName, from, to date.Year) (relation.Table[money.Rate], error) {
	return readStep(table, slot, column, from, to, tsv.Reader.Percent)
}
