package validate

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	AmountAgreesWithItsStateRule RuleName = "amount-agrees-with-state"
	StateOnlyAtTheStartRule      RuleName = "state-only-at-the-start"
	StateOnlyAtTheEndRule        RuleName = "state-only-at-the-end"
)

type AmountStates struct {
	Written string

	Absent []string

	Unfetched []string
}

func AmountAgreesWithItsState(
	slot tsv.Slot, amountColumn, stateColumn tsv.ColumnName, states AmountStates,
) Rule {
	absent := slices.Sorted(slices.Values(states.Absent))
	unfetched := slices.Sorted(slices.Values(states.Unfetched))
	return Rule{
		Name:  AmountAgreesWithItsStateRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			at, found := columnsOf(table, slot, amountColumn, stateColumn)
			if len(found) > 0 {
				return found
			}

			for row, fields := range table.Rows {
				amount, state := fields[at[amountColumn]], fields[at[stateColumn]]
				_, accountsForBlank := slices.BinarySearch(absent, state)
				_, missing := slices.BinarySearch(unfetched, state)

				switch {
				case state == states.Written && isBlank(amount):
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf("row %d: %s が %q なのに %s が空である。空欄は合計に 0 として入るので、額はそのまま消える",
							row+1, stateColumn, state, amountColumn),
					})
				case accountsForBlank && !isBlank(amount):
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf("row %d: %s が %q なのに %s に %s と書いてある。どちらかが違う",
							row+1, stateColumn, state, amountColumn, amount),
					})
				case missing:
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf("row %d: %s が %q なので、この行の %s は分からない。合計はこの行のぶんだけ少なく出る",
							row+1, stateColumn, state, amountColumn),
					})
				case !accountsForBlank && state != states.Written:
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf("row %d: %s の %q は語彙の外にある。%s が何を意味するかが決まっていない",
							row+1, stateColumn, state, stateColumn),
					})
				}
			}
			return found
		},
	}
}

func StateOnlyAtTheStart(
	slot tsv.Slot, keyColumn, orderColumn, stateColumn tsv.ColumnName, only string,
) Rule {
	return stateOnlyAtOneEnd(StateOnlyAtTheStartRule, slot, keyColumn, orderColumn, stateColumn, only, atTheStart)
}

func StateOnlyAtTheEnd(
	slot tsv.Slot, keyColumn, orderColumn, stateColumn tsv.ColumnName, only string,
) Rule {
	return stateOnlyAtOneEnd(StateOnlyAtTheEndRule, slot, keyColumn, orderColumn, stateColumn, only, atTheEnd)
}

type stateEnd int

const (
	atTheStart stateEnd = iota
	atTheEnd
)

func stateOnlyAtOneEnd(
	name RuleName, slot tsv.Slot, keyColumn, orderColumn, stateColumn tsv.ColumnName,
	only string, end stateEnd,
) Rule {
	return Rule{
		Name:  name,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			at, found := columnsOf(table, slot, keyColumn, orderColumn, stateColumn)
			if len(found) > 0 {
				return found
			}

			type entry struct{ order, state string }
			byKey := make(map[string][]entry, len(table.Rows))
			for _, fields := range table.Rows {
				key := fields[at[keyColumn]]
				byKey[key] = append(byKey[key], entry{
					order: fields[at[orderColumn]],
					state: fields[at[stateColumn]],
				})
			}

			for _, key := range slices.Sorted(maps.Keys(byKey)) {
				rows := byKey[key]
				slices.SortFunc(rows, func(a, b entry) int {
					return strings.Compare(a.order, b.order)
				})
				if end == atTheEnd {
					slices.Reverse(rows)
				}

				var seenOther bool
				var boundary string
				for _, row := range rows {
					if row.state != only {
						seenOther, boundary = true, row.order
						continue
					}
					if !seenOther {
						continue
					}
					found = append(found, Finding{
						Slot:    slot,
						Message: stateWentBack(end, key, only, row.order, boundary),
					})
				}
			}
			return found
		},
	}
}

func stateWentBack(end stateEnd, key, only, order, boundary string) string {
	if end == atTheEnd {
		return fmt.Sprintf("%s の %s が %q ではない。%s に %q になったので、戻ることはない",
			boundary, key, only, order, only)
	}
	return fmt.Sprintf("%s の %s が %q。%s には %q ではなかったので、戻ることはない",
		order, key, only, boundary, only)
}
