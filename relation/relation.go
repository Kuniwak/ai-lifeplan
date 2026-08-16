package relation

import (
	"fmt"
	"github.com/Kuniwak/lifeplan/date"
	"slices"
)

type Row[T any] struct {
	Year  date.Year
	Value T
}

type Table[T any] struct {
	rows []Row[T]
}

func New[T any](rows []Row[T]) Table[T] {
	sorted := slices.Clone(rows)
	slices.SortFunc(sorted, func(a, b Row[T]) int { return int(a.Year - b.Year) })

	for i := 1; i < len(sorted); i++ {
		if sorted[i].Year == sorted[i-1].Year {
			panic(fmt.Sprintf("relation: year %d appears more than once", sorted[i].Year))
		}
	}

	return Table[T]{rows: sorted}
}

func Span(from, to date.Year) []date.Year {
	years := make([]date.Year, 0, max(int(to-from)+1, 0))
	for y := from; y <= to; y++ {
		years = append(years, y)
	}
	return years
}

func Constant[T any](years []date.Year, v T) Table[T] {
	return Over(years, func(date.Year) T { return v })
}

func Over[T any](years []date.Year, valueAt func(date.Year) T) Table[T] {
	ascending := slices.Clone(years)
	slices.Sort(ascending)

	rows := make([]Row[T], 0, len(ascending))
	for _, y := range ascending {
		rows = append(rows, Row[T]{Year: y, Value: valueAt(y)})
	}
	return New(rows)
}

func (t Table[T]) Len() int {
	return len(t.rows)
}

func (t Table[T]) Years() []date.Year {
	if len(t.rows) == 0 {
		return nil
	}

	years := make([]date.Year, 0, len(t.rows))
	for _, r := range t.rows {
		years = append(years, r.Year)
	}
	return years
}

func (t Table[T]) Rows() []Row[T] {
	if len(t.rows) == 0 {
		return nil
	}
	return slices.Clone(t.rows)
}

func (t Table[T]) At(y date.Year) (T, bool) {
	i, found := slices.BinarySearchFunc(t.rows, y, func(r Row[T], y date.Year) int { return int(r.Year - y) })
	if !found {
		var zero T
		return zero, false
	}
	return t.rows[i].Value, true
}

func SameEveryYear[T comparable](a, b Table[T]) bool {
	if a.Len() != b.Len() {
		return false
	}
	for _, row := range a.Rows() {
		other, ok := b.At(row.Year)
		if !ok || other != row.Value {
			return false
		}
	}
	return true
}
