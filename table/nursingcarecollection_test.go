package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func TestTheYearSomebodyTurnsSixtyFiveShouldNotBeSpeciallyCollected(t *testing.T) {
	kokuho, kouki, nursingCare := statutoryTables(t)

	const pension money.Yen = 1_200_000
	calendar := oneMemberCalendar("本人", 2060, 65, 2)
	years := calendar.Years()

	built, err := table.HouseholdInsuranceTable(table.HouseholdInsuranceInput{
		Calendar: calendar,
		Members: map[table.PersonName]relation.Table[table.HouseholdMemberYear]{
			"本人": relation.Constant(years, table.HouseholdMemberYear{
				Receipts: pension, PensionReceipts: pension, OldAgePensionBenefit: pension,
			}),
		},
		EmployeeCovers:  map[table.PersonName]relation.Table[law.Cover]{"本人": relation.Constant(years, law.NationalHealthInsurance)},
		Kokuho:          kokuho.WithGrowth(law.NoCostGrowth()),
		Kouki:           kouki.WithGrowth(law.NoCostGrowth()),
		NationalPension: law.NationalPensionPremiumTable{},
		NursingCare:     nursingCare.WithGrowth(law.NoCostGrowth()),
	})
	if err != nil {
		t.Fatalf("table.HouseholdInsuranceTable: %v", err)
	}

	for _, c := range []struct {
		year     date.Year
		withheld bool
		why      string
	}{
		{2060, false, "65 歳になった年度は特別徴収が始まらない"},
		{2061, true, "翌年からは年金から差し引かれる"},
	} {
		row, ok := built.At(c.year)
		if !ok {
			t.Fatalf("%d が無い", c.year)
		}
		if len(row.NursingCareOf) != 1 {
			t.Fatalf("%d: 介護保険料の内訳が %d 件。1 件のはず", c.year, len(row.NursingCareOf))
		}
		got := row.NursingCareOf[0]
		if got.Premium <= 0 {
			t.Fatalf("%d: 保険料が %d である。この検査が空回りしている", c.year, got.Premium)
		}
		if got.SpeciallyCollected != c.withheld {
			t.Errorf("%d: 特別徴収 %v、%v のはず（%s）", c.year, got.SpeciallyCollected, c.withheld, c.why)
		}
	}
}
