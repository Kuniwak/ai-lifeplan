package relation

import (
	"cmp"
	"fmt"
	"slices"
)

type Bound[K cmp.Ordered] struct {
	key   K
	given bool
}

func From[K cmp.Ordered](k K) Bound[K] { return Bound[K]{key: k, given: true} }

func To[K cmp.Ordered](k K) Bound[K] { return Bound[K]{key: k, given: true} }

func Unbounded[K cmp.Ordered]() Bound[K] { return Bound[K]{} }

func (b Bound[K]) Key() (K, bool) { return b.key, b.given }

func (b Bound[K]) String() string {
	if !b.given {
		return "…"
	}
	return fmt.Sprint(b.key)
}

type Period[K cmp.Ordered, V any] struct {
	from  Bound[K]
	to    Bound[K]
	value V
}

func NewPeriod[K cmp.Ordered, V any](from, to Bound[K], value V) Period[K, V] {
	return Period[K, V]{from: from, to: to, value: value}
}

func (s Period[K, V]) backwards() bool {
	lower, hasLower := s.from.Key()
	upper, hasUpper := s.to.Key()
	return hasLower && hasUpper && upper < lower
}

func (s Period[K, V]) Value() V { return s.value }

func (s Period[K, V]) Bounds() (from, to Bound[K]) { return s.from, s.to }

func (s Period[K, V]) Covers(k K) bool {
	if lower, ok := s.from.Key(); ok && k < lower {
		return false
	}
	if upper, ok := s.to.Key(); ok && k > upper {
		return false
	}
	return true
}

func (s Period[K, V]) String() string {
	return fmt.Sprintf("%v〜%v: %v", s.from, s.to, s.value)
}

func Overlap[K cmp.Ordered, V any](a, b Period[K, V]) (K, bool) {
	witness, ok := highest(a.from, b.from)
	if !ok {
		witness, ok = lowest(a.to, b.to)
		if !ok {
			var everywhere K
			return everywhere, true
		}
	}
	if a.Covers(witness) && b.Covers(witness) {
		return witness, true
	}
	var none K
	return none, false
}

func highest[K cmp.Ordered](a, b Bound[K]) (K, bool) {
	x, hasX := a.Key()
	y, hasY := b.Key()
	switch {
	case hasX && hasY:
		return max(x, y), true
	case hasX:
		return x, true
	case hasY:
		return y, true
	}
	var none K
	return none, false
}

func lowest[K cmp.Ordered](a, b Bound[K]) (K, bool) {
	x, hasX := a.Key()
	y, hasY := b.Key()
	switch {
	case hasX && hasY:
		return min(x, y), true
	case hasX:
		return x, true
	case hasY:
		return y, true
	}
	var none K
	return none, false
}

type Periods[K cmp.Ordered, V any] struct {
	periods []Period[K, V]
}

func NewPeriods[K cmp.Ordered, V any](periods []Period[K, V]) (Periods[K, V], error) {
	for _, s := range periods {
		if s.backwards() {
			return Periods[K, V]{}, fmt.Errorf(
				"relation.NewPeriods: %v は終わりが始まりより前で、どの鍵も覆わない", s)
		}
	}
	for i, a := range periods {
		for _, b := range periods[i+1:] {
			if k, over := Overlap(a, b); over {
				return Periods[K, V]{}, fmt.Errorf(
					"relation.NewPeriods: %v は %v と %v の両方が覆っており、どちらが効くのか決まらない", k, a, b)
			}
		}
	}
	return Periods[K, V]{periods: slices.Clone(periods)}, nil
}

func (s Periods[K, V]) Len() int { return len(s.periods) }

func (s Periods[K, V]) All() []Period[K, V] { return slices.Clone(s.periods) }

func (s Periods[K, V]) Lookup(k K) (V, bool) {
	for _, period := range s.periods {
		if period.Covers(k) {
			return period.value, true
		}
	}
	var none V
	return none, false
}

func BandsOfPeriods[K interface{ ~int }, V any](periods []Period[K, V], step K) ([]Band[K, V], error) {
	if len(periods) == 0 {
		return nil, fmt.Errorf("relation.BandsOfPeriods: no periods, so every lookup would miss")
	}

	sorted := slices.Clone(periods)
	slices.SortFunc(sorted, func(a, b Period[K, V]) int {
		x, hasX := a.from.Key()
		y, hasY := b.from.Key()
		switch {
		case !hasX && !hasY:
			return 0
		case !hasX:
			return -1
		case !hasY:
			return 1
		}
		return cmp.Compare(x, y)
	})

	for i, s := range sorted[:len(sorted)-1] {
		next := sorted[i+1]
		end, bounded := s.to.Key()
		if !bounded {
			return nil, fmt.Errorf(
				"relation.BandsOfPeriods: %v が終わらないのに、そのあとに %v がある。終わらない帯は最後にしか置けない", s, next)
		}
		start, begins := next.from.Key()
		if !begins || start != end+step {
			return nil, fmt.Errorf(
				"relation.BandsOfPeriods: %v の次が %v で、%v を誰も書いていないか、二度書いている", s, next, end+step)
		}
	}
	if last := sorted[len(sorted)-1]; last.backwards() {
		return nil, fmt.Errorf("relation.BandsOfPeriods: %v は終わりが始まりより前である", last)
	} else if _, bounded := last.to.Key(); bounded {
		return nil, fmt.Errorf(
			"relation.BandsOfPeriods: 最後の %v に終わりがある。帯は最後の行を定義域の上まで延ばすので、"+
				"その行が覆わないと言っている年にも答えてしまう", last)
	}

	bands := make([]Band[K, V], 0, len(sorted))
	for _, s := range sorted {
		lower, _ := s.from.Key()
		bands = append(bands, Band[K, V]{Lower: lower, Value: s.value})
	}
	return bands, nil
}
