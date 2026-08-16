package table_test

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func statutoryTables(t *testing.T) (law.KokuhoTable, law.KoukiRatesTable, law.NursingCarePremiumTable) {
	t.Helper()

	fsys := os.DirFS("../" + law.LawDirectory)

	kokuho, err := law.LoadKokuhoTable(fsys, "世田谷区")
	if err != nil {
		t.Fatalf("law.LoadKokuhoTable: %v", err)
	}
	kouki, err := law.LoadKoukiRatesTable(fsys, "東京都")
	if err != nil {
		t.Fatalf("law.LoadKoukiRatesTable: %v", err)
	}
	nursingCare, err := law.LoadNursingCarePremiumTable(fsys, "世田谷区")
	if err != nil {
		t.Fatalf("law.LoadNursingCarePremiumTable: %v", err)
	}
	return kokuho, kouki, nursingCare
}

func oneMemberCalendar(name table.PersonName, from date.Year, firstAge, years int) relation.Table[table.CalendarRow] {
	rows := make([]relation.Row[table.CalendarRow], 0, years)
	for i := range years {
		rows = append(rows, relation.Row[table.CalendarRow]{
			Year: from + date.Year(i),
			Value: table.CalendarRow{
				Municipality: "世田谷区",
				Ages:         []table.PersonYear{{Name: name, Age: firstAge + i}},
			},
		})
	}
	return relation.New(rows)
}
