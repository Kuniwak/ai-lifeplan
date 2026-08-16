package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type ChildAllowanceRow struct {
	PaidFor []PersonPremium

	Dependents int

	Limits *law.ChildAllowanceLimits

	Total money.Yen
}

type ChildAllowanceInput struct {
	Calendar relation.Table[CalendarRow]

	HigherEarnerIncome relation.Table[money.Yen]

	Table law.ChildAllowanceTable
}

func ChildAllowanceTable(in ChildAllowanceInput) (relation.Table[ChildAllowanceRow], error) {
	var empty relation.Table[ChildAllowanceRow]

	years := in.Calendar.Years()
	rows := make([]relation.Row[ChildAllowanceRow], 0, len(years))

	for _, y := range years {
		calendar, _ := in.Calendar.At(y)

		judged, judgedCalendar := y-1, calendar
		if previous, ok := in.Calendar.At(judged); ok {
			judgedCalendar = previous
		}

		income, ok := in.HigherEarnerIncome.At(judged)
		if !ok {
			if income, ok = in.HigherEarnerIncome.At(y); !ok {
				return empty, fmt.Errorf("table.ChildAllowanceTable: no income to test the thresholds against in %d", y)
			}
		}

		var row ChildAllowanceRow
		for _, person := range judgedCalendar.Ages {
			if person.IsChild() && judgedCalendar.InHousehold(person.Name) {
				row.Dependents++
			}
		}

		if limits, tested := in.Table.Limits(y, row.Dependents); tested {
			row.Limits = &limits
		}

		born := 0
		for _, person := range calendar.Ages {
			if !person.IsChild() || !calendar.InHousehold(person.Name) {
				continue
			}
			if !law.ChildAllowanceCountsTowardsThirdOrLater(y, person.BornOn) {
				continue
			}
			born++

			paid := in.Table.Yearly(y, person.BornOn, born >= 3, income, row.Dependents)
			row.PaidFor = append(row.PaidFor, PersonPremium{Name: person.Name, Premium: paid})
			row.Total += paid
		}

		rows = append(rows, relation.Row[ChildAllowanceRow]{Year: y, Value: row})
	}

	return relation.New(rows), nil
}
