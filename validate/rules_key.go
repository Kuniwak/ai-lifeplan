package validate

import (
	"fmt"
	"strings"

	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/wording"
)

const UniqueKeyRule RuleName = "unique-key"

func UniqueKey(slot tsv.Slot, keyColumns []tsv.ColumnName) Rule {
	return Rule{
		Name:  UniqueKeyRule,
		Needs: []tsv.Slot{slot},
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			table := tables[slot]

			at, found := columnsOf(table, slot, keyColumns...)
			if len(found) > 0 {
				return found
			}

			seen := make(map[string]int, len(table.Rows))
			for row, fields := range table.Rows {
				parts := make([]string, 0, len(keyColumns))
				for _, column := range keyColumns {
					parts = append(parts, fields[at[column]])
				}
				key := strings.Join(parts, "\t")

				if before, twice := seen[key]; twice {
					found = append(found, Finding{
						Slot: slot,
						Message: fmt.Sprintf("%d 行目と %d 行目: %s", before+1, row+1,
							wording.DuplicateKeyFinding(keyKindOf(keyColumns), wording.Pair(parts...),
								wording.WhichRowAnswersForTheKey)),
					})
					continue
				}
				seen[key] = row
			}
			return found
		},
	}
}

func keyKindOf(keyColumns []tsv.ColumnName) string {
	names := make([]string, 0, len(keyColumns))
	for _, column := range keyColumns {
		names = append(names, string(column))
	}
	return strings.Join(names, " / ")
}
