package relation

import (
	"cmp"
	"fmt"
	"slices"
)

type Band[K cmp.Ordered, V any] struct {
	Lower K
	Value V
}

type Bands[K cmp.Ordered, V any] struct {
	bands []Band[K, V]
}

func NewBands[K cmp.Ordered, V any](bands []Band[K, V]) Bands[K, V] {
	sorted := slices.Clone(bands)
	slices.SortFunc(sorted, func(a, b Band[K, V]) int { return cmp.Compare(a.Lower, b.Lower) })

	for i := 1; i < len(sorted); i++ {
		if sorted[i].Lower == sorted[i-1].Lower {
			panic(fmt.Sprintf("relation: two bands start at %v, so neither can be said to apply", sorted[i].Lower))
		}
	}

	return Bands[K, V]{bands: sorted}
}

func (b Bands[K, V]) Len() int {
	return len(b.bands)
}

func (b Bands[K, V]) Min() (K, bool) {
	if len(b.bands) == 0 {
		var zero K
		return zero, false
	}
	return b.bands[0].Lower, true
}

func (b Bands[K, V]) Max() (K, bool) {
	if len(b.bands) == 0 {
		var zero K
		return zero, false
	}
	return b.bands[len(b.bands)-1].Lower, true
}

func (b Bands[K, V]) Lookup(k K) V {
	if len(b.bands) == 0 {
		panic(fmt.Sprintf("relation.Lookup: the table has no bands, so %v cannot be looked up", k))
	}

	i, found := slices.BinarySearchFunc(b.bands, k, func(band Band[K, V], k K) int {
		return cmp.Compare(band.Lower, k)
	})
	if found {
		return b.bands[i].Value
	}
	if i == 0 {
		lowest, _ := b.Min()
		panic(fmt.Sprintf("relation.Lookup: %v is below the lowest band, which starts at %v", k, lowest))
	}
	return b.bands[i-1].Value
}
