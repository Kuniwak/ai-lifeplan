package relation

import "github.com/Kuniwak/lifeplan/date"

func Map[A, B any](t Table[A], f func(date.Year, A) B) Table[B] {
	rows := make([]Row[B], 0, t.Len())
	for _, r := range t.rows {
		rows = append(rows, Row[B]{Year: r.Year, Value: f(r.Year, r.Value)})
	}
	return Table[B]{rows: rows}
}

func MapEach[K comparable, A, B any](tables map[K]Table[A], f func(date.Year, A) B) map[K]Table[B] {
	out := make(map[K]Table[B], len(tables))
	for name, t := range tables {
		out[name] = Map(t, f)
	}
	return out
}

func Aggregate[A, B any](t Table[A], initial B, accumulate func(B, date.Year, A) B) B {
	acc := initial
	for _, r := range t.rows {
		acc = accumulate(acc, r.Year, r.Value)
	}
	return acc
}
