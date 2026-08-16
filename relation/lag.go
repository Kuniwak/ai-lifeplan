package relation

import "github.com/Kuniwak/lifeplan/date"

func Lag[A, B, C any](a Table[A], b Table[B], missing B, combine func(date.Year, A, B) C) Table[C] {
	rows := make([]Row[C], 0, a.Len())
	for _, ra := range a.rows {
		previous, ok := b.At(ra.Year - 1)
		if !ok {
			previous = missing
		}
		rows = append(rows, Row[C]{Year: ra.Year, Value: combine(ra.Year, ra.Value, previous)})
	}

	return Table[C]{rows: rows}
}
