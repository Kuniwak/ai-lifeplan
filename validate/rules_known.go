package validate

import (
	"fmt"
	"maps"
	"slices"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/wording"
)

const BalanceFollowsTheKnownRule RuleName = "balance-follows-known"

type KnownKind struct {
	Column tsv.ColumnName

	RecordedFrom string
}

type KnownKinds map[string]KnownKind

type LowerBoundYears struct {
	Needs []tsv.Slot

	Years func(tables map[tsv.Slot]*tsv.Table) []string
}

type PartialMark struct {
	Column tsv.ColumnName
	Yes    string
}

func (k KnownKind) recordedAt(year string) bool {
	return k.RecordedFrom == "" || year+"/12/31" >= k.RecordedFrom
}

func BalanceFollowsTheKnown(
	balanceSlot tsv.Slot, yearColumn tsv.ColumnName, partial PartialMark,
	knownSlot tsv.Slot, knownYearColumn, knownKindColumn, knownAmountColumn tsv.ColumnName,
	kinds KnownKinds, alsoLowerBound LowerBoundYears,
) Rule {
	return Rule{
		Name:  BalanceFollowsTheKnownRule,
		Needs: append([]tsv.Slot{balanceSlot, knownSlot}, alsoLowerBound.Needs...),
		Check: func(tables map[tsv.Slot]*tsv.Table) []Finding {
			balances, known := tables[balanceSlot], tables[knownSlot]

			var elsewhere []string
			if alsoLowerBound.Years != nil {
				elsewhere = slices.Sorted(slices.Values(alsoLowerBound.Years(tables)))
			}

			atBalance, found := columnsOf(balances, balanceSlot, yearColumn, partial.Column)
			atKnown, missing := columnsOf(known, knownSlot, knownYearColumn, knownKindColumn, knownAmountColumn)
			found = append(found, missing...)
			if len(found) > 0 {
				return found
			}

			recorded := make(map[string]map[string]money.Yen, len(known.Rows))
			at := make(map[string]map[string]int, len(known.Rows))
			for row, fields := range known.Rows {
				kind := fields[atKnown[knownKindColumn]]
				if _, assigned := kinds[kind]; !assigned {
					found = append(found, Finding{
						Slot: knownSlot,
						Message: fmt.Sprintf(
							"row %d: %q はどの列に入るか決まっていない。accounts.tsv に区分を書く",
							row+1, kind),
					})
					continue
				}
				amount, bad := yenAt(knownSlot, row, knownAmountColumn, fields[atKnown[knownAmountColumn]])
				if bad != nil {
					found = append(found, *bad)
					continue
				}

				year := fields[atKnown[knownYearColumn]]
				if _, seen := recorded[year]; !seen {
					recorded[year] = make(map[string]money.Yen, 1)
					at[year] = make(map[string]int, 1)
				}
				if before, twice := at[year][kind]; twice {
					found = append(found, Finding{
						Slot: knownSlot,
						Message: fmt.Sprintf("%d 行目と %d 行目: %s", before+1, row+1,
							wording.DuplicateKeyFinding(fmt.Sprintf("%s 年の種別", year), wording.Name(kind),
								wording.WhichAmountIsTheRecord)),
					})
					continue
				}
				recorded[year][kind] = amount
				at[year][kind] = row
			}
			if len(found) > 0 {
				return found
			}

			read := make(map[string]map[string]bool, len(recorded))
			for row, fields := range balances.Rows {
				year := fields[atBalance[yearColumn]]
				read[year] = make(map[string]bool, len(kinds))

				unrecorded := make(map[tsv.ColumnName][]string, len(kinds))
				mixed := make(map[tsv.ColumnName]bool, len(kinds))
				for kind, k := range kinds {
					if k.recordedAt(year) {
						mixed[k.Column] = true
						continue
					}
					unrecorded[k.Column] = append(unrecorded[k.Column], kind)
				}

				lowerBound := false
				for _, forColumn := range unrecorded {
					for _, kind := range forColumn {
						if _, ok := recorded[year][kind]; !ok {
							lowerBound = true
						}
					}
				}
				if _, known := slices.BinarySearch(elsewhere, year); known {
					lowerBound = true
				}
				if marked := fields[atBalance[partial.Column]] == partial.Yes; marked != lowerBound {
					says := "下限である理由があるので、印が立っていなければならない"
					if !lowerBound {
						says = "下限である理由が 1 つも無いので、印が立っていてはいけない"
					}
					found = append(found, Finding{
						Slot:    balanceSlot,
						Message: fmt.Sprintf("%s 年の %s: %s", year, partial.Column, says),
					})
				}

				for _, column := range slices.Sorted(maps.Keys(unrecorded)) {
					if mixed[column] {
						for _, kind := range unrecorded[column] {
							read[year][kind] = true
						}
						found = append(found, Finding{
							Slot: balanceSlot,
							Message: fmt.Sprintf(
								"%s 年の %s は書き出しの数と別に記録された数が混ざるので、合計しか無いこの表からはどちらがいくらか読めない。列を分ける",
								year, column),
						})
						continue
					}

					i, ok := balances.ColumnIndex(column)
					if !ok {
						found = append(found, Finding{
							Slot:    balanceSlot,
							Message: fmt.Sprintf("the column %q is missing", column),
						})
						continue
					}
					held, bad := yenAt(balanceSlot, row, column, fields[i])
					if bad != nil {
						found = append(found, *bad)
						continue
					}

					var want money.Yen
					for _, kind := range unrecorded[column] {
						want += recorded[year][kind]
						read[year][kind] = true
					}
					if held != want {
						found = append(found, boundColumnFinding(
							balanceSlot, year, column, held, "別に記録されている残高の合計", want))
					}
				}
			}

			for _, year := range slices.Sorted(maps.Keys(recorded)) {
				for _, kind := range slices.Sorted(maps.Keys(recorded[year])) {
					if read[year][kind] {
						continue
					}
					found = append(found, Finding{
						Slot: knownSlot,
						Message: fmt.Sprintf(
							"row %d: %s 年の %q は読まれない。書き出しがその年には既に記録しているか、balance.tsv にその年の行が無い",
							at[year][kind]+1, year, kind),
					})
				}
			}
			return found
		},
	}
}
