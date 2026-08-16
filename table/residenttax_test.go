package table_test

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func residentTaxOfTheBaseProject(t *testing.T, taxpayer, spouse table.PersonName) relation.Table[table.ResidentTaxRow] {
	t.Helper()

	built, err := table.ResidentTaxTable(table.ResidentTaxInput{
		Calendar:  calendarOfTheBaseProject(t),
		IncomeTax: incomeTaxOfTheBaseProject(t, taxpayer, spouse),
		Tables:    law.MustLoadResidentLevies(t, os.DirFS("../"+law.LawDirectory), "世田谷区"),
		Liable:    liabilityOfTheBaseProject(t, taxpayer, spouse),
	})
	if err != nil {
		t.Fatalf("table.ResidentTaxTable: %v", err)
	}
	return built
}

func TestThePerCapitaAmountShouldBeChargedOnAnIncomeTooSmallToTax(t *testing.T) {
	const (
		year     = date.Year(2030)
		earner   = "夫"
		received = money.Yen(3_000_000)
	)

	built, err := table.ResidentTaxTable(table.ResidentTaxInput{
		Calendar: relation.New([]relation.Row[table.CalendarRow]{
			{Year: year - 1, Value: table.CalendarRow{Municipality: "世田谷区"}},
			{Year: year, Value: table.CalendarRow{Municipality: "世田谷区"}},
		}),
		IncomeTax: relation.New([]relation.Row[table.IncomeTaxRow]{
			{Year: year - 1, Value: table.IncomeTaxRow{
				TotalIncome: received,
				Deductions:  table.Deductions{SocialInsurance: received},
			}},
		}),
		Tables: law.MustLoadResidentLevies(t, os.DirFS("../"+law.LawDirectory), "世田谷区"),

		Liable: relation.New([]relation.Row[law.ResidentTaxLiability]{
			{Year: year, Value: law.ResidentTaxLiability{PerCapita: true}},
		}),
	})
	if err != nil {
		t.Fatalf("table.ResidentTaxTable: %v", err)
	}

	row, ok := built.At(year)
	if !ok {
		t.Fatalf("%d is missing", year)
	}
	if row.Taxable != 0 {
		t.Fatalf("この検査は課税所得金額 0 の年を作るためのもの。%d になっている", row.Taxable)
	}
	if row.PerCapita <= 0 {
		t.Errorf("均等割 = %d、合計所得 %d は非課税限度額の上なので課されるはず", row.PerCapita, received)
	}
	if row.ForestEnvironmentTax <= 0 {
		t.Errorf("森林環境税 = %d、均等割が課される年なので課されるはず", row.ForestEnvironmentTax)
	}
	if row.PrefecturalIncome+row.MunicipalIncome != 0 {
		t.Errorf("課税所得金額が 0 なのに所得割が %d ある", row.PrefecturalIncome+row.MunicipalIncome)
	}
}
