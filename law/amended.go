package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
)

type Amended[V any] struct {
	table YearTable[V]

	name string
	from date.Year

	built bool
}

func NewAmended[V any](name string, rows ...YearRow[V]) Amended[V] {
	return NewAmendedFrom(name, 0, rows...)
}

func NewAmendedFrom[V any](name string, from date.Year, rows ...YearRow[V]) Amended[V] {
	if len(rows) == 0 {
		panic(fmt.Sprintf("law.NewAmendedFrom: %s に行が無い。答えられる年が一つも無い", name))
	}

	earliest := rows[0].FromYear
	for _, row := range rows {
		earliest = min(earliest, row.FromYear)
	}
	if earliest != from {
		panic(fmt.Sprintf(
			"law.NewAmendedFrom: %s の記録は %d 年からと言われているが、いちばん古い行は %d 年から始まる",
			name, from, earliest))
	}

	table, err := NewYearTable(rows)
	if err != nil {
		panic(fmt.Sprintf("law.NewAmendedFrom: %s: %v", name, err))
	}
	return Amended[V]{table: table, name: name, from: from, built: true}
}

func (a Amended[V]) FirstWrittenYear() (date.Year, bool) {
	return a.from, a.built
}

func (a Amended[V]) At(year date.Year) V {
	if !a.built {
		panic("law.Amended.At: this was not built by law.NewAmended, so it has no rows")
	}

	value, ok := a.table.At(year)
	if !ok {
		panic(fmt.Sprintf(
			"law: %s に %d 年の額が無い。この額が確かめられているのは %d 年からで、それより前は誰も調べていない",
			a.name, year, a.from))
	}
	return value
}
