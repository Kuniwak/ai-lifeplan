package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/relation"
)

type partedTable[N comparable, V any] struct {
	parts map[N]YearTable[V]

	order []N
}

func newPartedTable[N comparable, V any](what string, byPart map[N][]relation.Period[date.Year, V], order []N) (partedTable[N, V], error) {
	if len(byPart) == 0 {
		return partedTable[N, V]{}, fmt.Errorf("%s: the table has no parts, so every lookup would miss", what)
	}

	parts := make(map[N]YearTable[V], len(byPart))
	for part, rows := range byPart {
		banded, err := NewYearTableOfPeriods(rows)
		if err != nil {
			return partedTable[N, V]{}, fmt.Errorf("%s: %v: %w", what, part, err)
		}
		parts[part] = banded
	}
	return partedTable[N, V]{parts: parts, order: order}, nil
}

func (t partedTable[N, V]) Parts() []N {
	parts := make([]N, 0, len(t.parts))
	for _, name := range t.order {
		if _, ok := t.parts[name]; ok {
			parts = append(parts, name)
		}
	}
	return parts
}

func (t partedTable[N, V]) At(name N, year date.Year) (V, bool) { return t.parts[name].At(year) }

func (t partedTable[N, V]) yearsOf(name N) YearTable[V] { return t.parts[name] }

func (t partedTable[N, V]) FirstWrittenYear() (date.Year, bool) {
	return t.earliest(func(y YearTable[V]) (date.Year, bool) { return y.FirstWrittenYear() })
}

func (t partedTable[N, V]) LastWrittenYear() (date.Year, bool) {
	return t.earliest(func(y YearTable[V]) (date.Year, bool) { return y.LastWrittenYear() })
}

func (t partedTable[N, V]) earliest(of func(YearTable[V]) (date.Year, bool)) (date.Year, bool) {
	var earliest date.Year
	found := false
	for _, name := range t.Parts() {
		year, ok := of(t.parts[name])
		if !ok {
			continue
		}
		if !found || year < earliest {
			earliest, found = year, true
		}
	}
	return earliest, found
}
