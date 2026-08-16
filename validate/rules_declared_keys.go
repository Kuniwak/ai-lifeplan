package validate

import (
	"fmt"
	"slices"

	"github.com/Kuniwak/lifeplan/sets"
	"github.com/Kuniwak/lifeplan/tsv"
)

const KeysAreDeclaredRule RuleName = "keys-are-declared"

func KeysAreDeclared(partsSlot tsv.Slot, partsKeyColumn tsv.ColumnName, declaredSlot tsv.Slot, declaredKeyColumn tsv.ColumnName) Rule {
	needs := []tsv.Slot{partsSlot, declaredSlot}
	slices.Sort(needs)

	return Rule{
		Name:  KeysAreDeclaredRule,
		Needs: needs,
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			used, found := keysOf(tables[partsSlot], partsSlot, partsKeyColumn)
			if len(found) > 0 {
				return found
			}
			declared, found := keysOf(tables[declaredSlot], declaredSlot, declaredKeyColumn)
			if len(found) > 0 {
				return found
			}

			if len(declared) == 0 {
				return []Finding{{
					Slot: declaredSlot,
					Message: fmt.Sprintf(
						"%q に宣言が 1 つも無い。%s のどの鍵が揃うべきかを決められない",
						declaredSlot, partsSlot),
				}}
			}

			for _, key := range sets.Difference(declared, used) {
				found = append(found, Finding{
					Slot: partsSlot,
					Message: fmt.Sprintf(
						"%q は %s が宣言しているのに、%s のどの年にも一度も書かれていない。"+
							"行が無い鍵は「揃うべき鍵」として数えられないので、"+
							"どの検査からも見えないまま合計だけが小さくなる",
						key, declaredSlot, partsSlot),
				})
			}
			for _, key := range sets.Difference(used, declared) {
				found = append(found, Finding{
					Slot: partsSlot,
					Message: fmt.Sprintf(
						"%q が %s に書かれているが、%s が宣言していない。"+
							"名前を書き替えたのなら、宣言のほうも書き替えること",
						key, partsSlot, declaredSlot),
				})
			}
			return found
		},
	}
}

func keysOf(table *tsv.Table, slot tsv.Slot, column tsv.ColumnName) ([]string, []Finding) {
	at, found := columnsOf(table, slot, column)
	if len(found) > 0 {
		return nil, found
	}

	var keys []string
	for _, fields := range table.Rows {
		if key := fields[at[column]]; !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}
