package validate

import (
	"fmt"
	"slices"

	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/wording"
)

const EveryChoiceMadeRule = "every-choice-made"

type Answer func(string) bool

func OneOf(answers ...string) Answer {
	allowed := slices.Sorted(slices.Values(answers))
	return func(answer string) bool {
		_, ok := slices.BinarySearch(allowed, answer)
		return ok
	}
}

func AnyAnswer(string) bool { return true }

func EveryChoiceMade(slot tsv.Slot, itemColumn, answerColumn tsv.ColumnName, items []string, allowed Answer, silent string) Rule {
	wanted := slices.Sorted(slices.Values(items))

	return Rule{
		Name:  EveryChoiceMadeRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			at, found := columnsOf(table, slot, itemColumn, answerColumn)
			if len(found) > 0 {
				return found
			}

			answered := make(map[string]int, len(table.Rows))
			for row, fields := range table.Rows {
				item := fields[at[itemColumn]]
				answer := fields[at[answerColumn]]

				if _, ok := slices.BinarySearch(wanted, item); !ok {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf("%d 行目: %q は計画に場所の無い%sである。あるのは %v",
							row+1, item, itemColumn, wanted),
					})
					continue
				}
				if before, twice := answered[item]; twice {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf("%d 行目と %d 行目: %s", before+1, row+1,
							wording.DuplicateKeyFinding(string(itemColumn), wording.Name(item), wording.WhichAnswerIsTaken)),
					})
				} else {
					answered[item] = row
				}

				if !allowed(answer) {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf("%d 行目: %q の %s が %q である。この列が受け付けない答えである",
							row+1, item, answerColumn, answer),
					})
				}
			}

			for _, item := range wanted {
				if _, ok := answered[item]; ok {
					continue
				}
				found = append(found, Finding{
					Slot: slot,
					Message: fmt.Sprintf("%q について誰も答えていない。行が無いことは答えが無いことではなく、黙って %q と答えたことになる",
						item, silent),
				})
			}
			return found
		},
	}
}
