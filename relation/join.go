package relation

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/sets"
)

func Join[A, B, C any](a Table[A], b Table[B], combine func(date.Year, A, B) C) Table[C] {
	assertSameYears("Join", a.Years(), b.Years())

	rows := make([]Row[C], 0, a.Len())
	for _, ra := range a.rows {
		vb, ok := b.At(ra.Year)
		if !ok {
			panic(fmt.Sprintf("relation.Join: year %d vanished between the check and the join", ra.Year))
		}
		rows = append(rows, Row[C]{Year: ra.Year, Value: combine(ra.Year, ra.Value, vb)})
	}

	return Table[C]{rows: rows}
}

func LeftJoin[A, B, C any](a Table[A], b Table[B], missing B, combine func(date.Year, A, B) C) Table[C] {
	if extra := sets.Difference(b.Years(), a.Years()); len(extra) > 0 {
		panic(fmt.Sprintf(
			"relation.LeftJoin: the right table covers %d year(s) the left one does not, so their values would be discarded: %v",
			len(extra), extra))
	}

	rows := make([]Row[C], 0, a.Len())
	for _, ra := range a.rows {
		vb, ok := b.At(ra.Year)
		if !ok {
			vb = missing
		}
		rows = append(rows, Row[C]{Year: ra.Year, Value: combine(ra.Year, ra.Value, vb)})
	}

	return Table[C]{rows: rows}
}

func assertSameYears(op string, a, b []date.Year) {
	onlyA := sets.Difference(a, b)
	onlyB := sets.Difference(b, a)
	if len(onlyA) == 0 && len(onlyB) == 0 {
		return
	}

	panic(fmt.Sprintf(
		"relation.%s: the two tables cover different years; only on the left: %v, only on the right: %v",
		op, onlyA, onlyB))
}
