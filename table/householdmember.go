package table

import (
	"fmt"
	"maps"
	"slices"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type HouseholdMemberYear struct {
	Receipts, Income money.Yen

	PensionReceipts, PensionIncome money.Yen

	OldAgePensionBenefit money.Yen

	Taxed law.ResidentTaxLiability
}

func HouseholdMembersOf(
	calendar relation.Table[CalendarRow],
	income map[PersonName]relation.Table[IncomeRow],
	taxed map[PersonName]relation.Table[law.ResidentTaxLiability],
) (map[PersonName]relation.Table[HouseholdMemberYear], error) {
	members := make(map[PersonName]relation.Table[HouseholdMemberYear], len(income))
	for _, person := range slices.Sorted(maps.Keys(income)) {
		earned := income[person]

		liable, ok := taxed[person]
		if !ok {
			return nil, fmt.Errorf(
				"table.HouseholdMembersOf: %q の収入はあるが、市民税を課されるかどうかが無い", person)
		}

		rows := make([]relation.Row[HouseholdMemberYear], 0, earned.Len())
		for _, row := range earned.Rows() {
			levied, ok := liable.At(row.Year)
			if !ok {
				return nil, fmt.Errorf(
					"table.HouseholdMembersOf: %q の %d 年の収入はあるが、市民税を課されるかどうかが無い",
					person, row.Year)
			}
			rows = append(rows, relation.Row[HouseholdMemberYear]{
				Year: row.Year,
				Value: HouseholdMemberYear{
					Receipts:             row.Value.Total,
					Income:               row.Value.TotalIncome,
					PensionReceipts:      row.Value.PensionReceived,
					PensionIncome:        row.Value.PensionIncome,
					OldAgePensionBenefit: row.Value.OldAgePensionBenefit,
					Taxed:                levied,
				},
			})
		}
		members[person] = relation.New(rows)
	}

	for _, person := range slices.Sorted(maps.Keys(taxed)) {
		if _, ok := income[person]; !ok {
			return nil, fmt.Errorf(
				"table.HouseholdMembersOf: %q は市民税を課されるかどうかだけがあって収入が無い", person)
		}
	}

	children := make(map[PersonName][]relation.Row[HouseholdMemberYear], 2)
	for _, row := range calendar.Rows() {
		for _, person := range row.Value.Ages {
			if person.IsChild() {
				if _, ok := members[person.Name]; !ok {
					children[person.Name] = append(children[person.Name],
						relation.Row[HouseholdMemberYear]{Year: row.Year})
				}
			}
		}
	}
	for _, person := range slices.Sorted(maps.Keys(children)) {
		members[person] = relation.New(children[person])
	}

	for _, row := range calendar.Rows() {
		for _, person := range row.Value.Ages {
			if _, ok := members[person.Name].At(row.Year); ok {
				continue
			}
			if !person.IsChild() {
				return nil, fmt.Errorf(
					"table.HouseholdMembersOf: %d 年の %q は就学段階を持たないのに収入の表がその年を覆っていない。"+
						"稼ぎが無いのか、表が足りないのかが分からない", row.Year, person.Name)
			}
			return nil, fmt.Errorf(
				"table.HouseholdMembersOf: %d 年の %q の行が作られていない", row.Year, person.Name)
		}
	}
	return members, nil
}
