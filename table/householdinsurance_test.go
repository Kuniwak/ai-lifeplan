package table_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

func householdInsuranceOfTheBaseProject(t *testing.T) relation.Table[table.HouseholdInsuranceRow] {
	t.Helper()
	return buildHouseholdInsurance(t, keepTheProjectsWifeReceipts)
}

func householdInsuranceWithWifeReceipts(t *testing.T, forced money.Yen) relation.Table[table.HouseholdInsuranceRow] {
	t.Helper()
	return buildHouseholdInsurance(t, func(money.Yen) money.Yen { return forced })
}

func keepTheProjectsWifeReceipts(own money.Yen) money.Yen { return own }

func buildHouseholdInsurance(t *testing.T, wifeReceipts func(money.Yen) money.Yen) relation.Table[table.HouseholdInsuranceRow] {
	t.Helper()

	calendar := calendarOfTheBaseProject(t)
	lawFS := os.DirFS("../" + law.LawDirectory)

	kokuho, err := law.LoadKokuhoTable(os.DirFS("../"+law.LawDirectory), "世田谷区")
	if err != nil {
		t.Fatalf("law.LoadKokuhoTable: %v", err)
	}
	kouki, err := law.LoadKoukiRatesTable(os.DirFS("../"+law.LawDirectory), "東京都")
	if err != nil {
		t.Fatalf("law.LoadKoukiRatesTable: %v", err)
	}
	nursingCare, err := law.LoadNursingCarePremiumTable(os.DirFS("../"+law.LawDirectory), "世田谷区")
	if err != nil {
		t.Fatalf("law.LoadNursingCarePremiumTable: %v", err)
	}
	monthly := law.MustLoadNationalPensionPremiums(t, lawFS)

	income := make(map[table.PersonName]relation.Table[table.IncomeRow], 2)
	taxed := make(map[table.PersonName]relation.Table[law.ResidentTaxLiability], 2)
	for person, slot := range map[table.PersonName]tsv.Slot{"夫": input.IncomeHusbandSlot, "妻": input.IncomeWifeSlot} {
		built := incomeOfTheBaseProject(t, person, slot)
		if person == "妻" {
			rows := make([]relation.Row[table.IncomeRow], 0, built.Len())
			for _, row := range built.Rows() {
				row.Value.Total = wifeReceipts(row.Value.Total)
				rows = append(rows, row)
			}
			built = relation.New(rows)
		}
		income[person] = built
	}

	for person, spouse := range map[table.PersonName]table.PersonName{"夫": "妻", "妻": "夫"} {
		taxed[person] = liabilityOfTheBaseProject(t, person, spouse)
	}

	members, err := table.HouseholdMembersOf(calendar, income, taxed)
	if err != nil {
		t.Fatalf("table.HouseholdMembersOf: %v", err)
	}

	employee := make([]relation.Row[law.Cover], 0, calendar.Len())
	for _, row := range socialInsuranceOfTheBaseProject(t).Rows() {
		employee = append(employee, relation.Row[law.Cover]{Year: row.Year, Value: row.Value.Cover})
	}

	built, err := table.HouseholdInsuranceTable(table.HouseholdInsuranceInput{
		Calendar:        calendar,
		Members:         members,
		EmployeeCovers:  map[table.PersonName]relation.Table[law.Cover]{"夫": relation.New(employee)},
		Kokuho:          kokuho.WithGrowth(law.NoCostGrowth()),
		Kouki:           kouki.WithGrowth(law.NoCostGrowth()),
		NationalPension: law.NationalPensionPremiumTable{YearYenTable: monthly},

		NursingCare: nursingCare.WithGrowth(law.NoCostGrowth()),
	})
	if err != nil {
		t.Fatalf("table.HouseholdInsuranceTable: %v", err)
	}
	return built
}

func TestTheHouseholdInsuranceShouldRefuseAnybodyItCannotAnswerFor(t *testing.T) {
	kokuho, kouki, nursingCare := statutoryTables(t)

	calendar := relation.New([]relation.Row[table.CalendarRow]{{
		Year: 2060,
		Value: table.CalendarRow{Municipality: "世田谷区", Ages: []table.PersonYear{
			{Name: "本人", Age: 66},
			{Name: "連れ合い", Age: 64},
		}},
	}})
	years := calendar.Years()
	earning := relation.Constant(years, table.HouseholdMemberYear{
		Receipts: 1_200_000, PensionReceipts: 1_200_000, OldAgePensionBenefit: 1_200_000,
	})

	for name, c := range map[string]struct {
		insured table.PersonName
		members map[table.PersonName]relation.Table[table.HouseholdMemberYear]
		says    string
	}{
		"被保険者の行が無い": {
			"本人",
			map[table.PersonName]relation.Table[table.HouseholdMemberYear]{"連れ合い": earning},
			"収入が渡されていない",
		},
		"世帯の他の人の行が無い": {
			"本人",
			map[table.PersonName]relation.Table[table.HouseholdMemberYear]{"本人": earning},
			"収入が渡されていない",
		},

		"被保険者が暦にいない": {
			"だれか",
			map[table.PersonName]relation.Table[table.HouseholdMemberYear]{"本人": earning, "連れ合い": earning},
			"暦に被保険者",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := table.HouseholdInsuranceTable(table.HouseholdInsuranceInput{
				Calendar:        calendar,
				Members:         c.members,
				EmployeeCovers:  map[table.PersonName]relation.Table[law.Cover]{c.insured: relation.Constant(years, law.NationalHealthInsurance)},
				Kokuho:          kokuho.WithGrowth(law.NoCostGrowth()),
				Kouki:           kouki.WithGrowth(law.NoCostGrowth()),
				NationalPension: law.NationalPensionPremiumTable{},
				NursingCare:     nursingCare.WithGrowth(law.NoCostGrowth()),
			})

			if err == nil {
				t.Fatal("行の無い人がいるのに黙って通った")
			}
			if !strings.Contains(err.Error(), "2060") {
				t.Errorf("誤りの文に年が無い: %v", err)
			}

			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("誤りの文が %q を言っていない: %v", c.says, err)
			}
		})
	}
}
