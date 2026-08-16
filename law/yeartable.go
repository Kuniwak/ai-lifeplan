package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

type YearRateTable struct {
	name  string
	bands relation.Bands[date.Year, money.Rate]
}

func ParseYearRateTable(table *tsv.Table, name string, rateColumn tsv.ColumnName) (YearRateTable, error) {
	r, err := newReader(table, name, LawStartYearColumn, rateColumn, LawEndYearColumn)
	if err != nil {
		return YearRateTable{}, fmt.Errorf("law.ParseYearRateTable: %w", err)
	}

	bands, err := readYearBands(r, func(row int) (money.Rate, error) {
		return r.Percent(row, rateColumn)
	})
	if err != nil {
		return YearRateTable{}, fmt.Errorf("law.ParseYearRateTable: %w", err)
	}
	return YearRateTable{name: name, bands: relation.NewBands(bands)}, nil
}

func (t YearRateTable) Rate(year date.Year) money.Rate {
	assertWithinTheRecord(t.name, t.bands, year)
	return t.bands.Lookup(year)
}

type YearYenTable struct {
	name  string
	bands relation.Bands[date.Year, money.Yen]
}

func ParseYearYenTable(table *tsv.Table, name string, amountColumn tsv.ColumnName) (YearYenTable, error) {
	r, err := newReader(table, name, LawStartYearColumn, amountColumn, LawEndYearColumn)
	if err != nil {
		return YearYenTable{}, fmt.Errorf("law.ParseYearYenTable: %w", err)
	}

	bands, err := readYearBands(r, func(row int) (money.Yen, error) {
		return r.Yen(row, amountColumn)
	})
	if err != nil {
		return YearYenTable{}, fmt.Errorf("law.ParseYearYenTable: %w", err)
	}
	return YearYenTable{name: name, bands: relation.NewBands(bands)}, nil
}

func readYearBands[V any](r reader, value func(row int) (V, error)) ([]relation.Band[date.Year, V], error) {
	periods, err := readYearPeriods(r, value)
	if err != nil {
		return nil, err
	}
	return relation.BandsOfPeriods(periods, 1)
}

func readYearPeriods[V any](r reader, value func(row int) (V, error)) ([]relation.Period[date.Year, V], error) {
	periods := make([]relation.Period[date.Year, V], 0, r.Rows())
	for row := range r.Rows() {
		from, err := r.startBound(row)
		if err != nil {
			return nil, err
		}
		to, err := r.endBound(row)
		if err != nil {
			return nil, err
		}
		v, err := value(row)
		if err != nil {
			return nil, err
		}
		periods = append(periods, relation.NewPeriod(from, to, v))
	}
	return periods, nil
}

func (t YearYenTable) Amount(year date.Year) money.Yen {
	assertWithinTheRecord(t.name, t.bands, year)
	return t.bands.Lookup(year)
}

func (r reader) startYear(row int) (date.Year, error) {
	if validate.LawYearWord(r.Field(row, LawStartYearColumn)) == validate.Unknown {
		return 0, nil
	}
	return r.Year(row, LawStartYearColumn)
}

func (r reader) startBound(row int) (relation.Bound[date.Year], error) {
	if validate.LawYearWord(r.Field(row, LawStartYearColumn)) == validate.Unknown {
		return relation.Unbounded[date.Year](), nil
	}
	year, err := r.Year(row, LawStartYearColumn)
	if err != nil {
		return relation.Unbounded[date.Year](), err
	}
	return relation.From(year), nil
}

func (r reader) endBound(row int) (relation.Bound[date.Year], error) {
	if validate.LawYearWord(r.Field(row, LawEndYearColumn)) == validate.Indefinite {
		return relation.Unbounded[date.Year](), nil
	}
	year, err := r.Year(row, LawEndYearColumn)
	if err != nil {
		return relation.Unbounded[date.Year](), err
	}
	return relation.To(year), nil
}

func assertWithinTheRecord[V any](name string, bands relation.Bands[date.Year, V], year date.Year) {
	first, ok := bands.Min()
	if !ok || year >= first {
		return
	}
	panic(fmt.Sprintf(
		"law: %s に %d 年の行が無い。この表は %d 年からしか書かれておらず、それより前は誰も値を書いていない。"+
			"その表の最初の行の適用開始年を確かめること",
		name, year, first))
}

type YearRow[V any] struct {
	FromYear date.Year
	Value    V
}

type YearTable[V any] struct {
	bands relation.Bands[date.Year, V]
}

func NewYearTableOfPeriods[V any](periods []relation.Period[date.Year, V]) (YearTable[V], error) {
	bands, err := relation.BandsOfPeriods(periods, 1)
	if err != nil {
		return YearTable[V]{}, err
	}
	return YearTable[V]{bands: relation.NewBands(bands)}, nil
}

func NewYearTable[V any](rows []YearRow[V]) (YearTable[V], error) {
	if len(rows) == 0 {
		return YearTable[V]{}, fmt.Errorf("law.NewYearTable: no rows, so every lookup would miss")
	}

	bands := make([]relation.Band[date.Year, V], 0, len(rows))
	for _, row := range rows {
		bands = append(bands, relation.Band[date.Year, V]{Lower: row.FromYear, Value: row.Value})
	}

	return YearTable[V]{bands: relation.NewBands(bands)}, nil
}

func (t YearTable[V]) At(year date.Year) (V, bool) {
	var zero V
	first, ok := t.bands.Min()
	if !ok || year < first {
		return zero, false
	}
	return t.bands.Lookup(year), true
}
