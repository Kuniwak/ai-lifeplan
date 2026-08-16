package table_test

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func theHealthGradesAndRates(t *testing.T) (law.StandardRemunerationTable, law.SocialInsuranceRates) {
	t.Helper()

	lawFS := os.DirFS("../" + law.LawDirectory)
	return law.MustLoadStandardRemunerations(t, lawFS, law.StandardRemunerationHealthTableName),
		law.MustLoadSocialInsuranceRates(t, lawFS).WithGrowth(law.NoCostGrowth())
}

func thePayOfTheBaseProject(t *testing.T) relation.Table[table.Pay] {
	t.Helper()

	income, err := table.IncomeInputFor(tablesOfTheBaseProject(t), calendarOfTheBaseProject(t), "夫", input.IncomeHusbandSlot)
	if err != nil {
		t.Fatalf("table.IncomeInputFor: %v", err)
	}
	return income.Pay
}

func socialInsuranceOfTheBaseProject(t *testing.T) relation.Table[table.SocialInsuranceRow] {
	t.Helper()

	return socialInsuranceOfTheBaseProjectReading(t,
		actuals.PremiumsRecordedByYear(thePayslips(t), "夫"))
}

func socialInsuranceDerivedOfTheBaseProject(t *testing.T) relation.Table[table.SocialInsuranceRow] {
	t.Helper()
	return socialInsuranceOfTheBaseProjectReading(t, relation.Table[actuals.PremiumsDeducted]{})
}

func socialInsuranceOfTheBaseProjectReading(t *testing.T, recorded relation.Table[actuals.PremiumsDeducted]) relation.Table[table.SocialInsuranceRow] {
	t.Helper()

	lawFS := os.DirFS("../" + law.LawDirectory)
	load := func(name string) law.StandardRemunerationTable {
		t.Helper()
		return law.MustLoadStandardRemunerations(t, lawFS, name)
	}

	grades, rates := theHealthGradesAndRates(t)
	employment := law.EmploymentInsuranceTable{
		YearRateTable: law.MustLoadEmploymentInsuranceRates(t, lawFS),
	}

	built, err := table.SocialInsuranceTable(table.SocialInsuranceInput{
		Pay:                 thePayOfTheBaseProject(t),
		Calendar:            calendarOfTheBaseProject(t),
		Person:              "夫",
		HealthGrades:        grades,
		PensionGrades:       load(law.StandardRemunerationPensionTableName),
		Rates:               rates,
		EmploymentInsurance: employment,
		Recorded:            recorded,
	})
	if err != nil {
		t.Fatalf("table.SocialInsuranceTable: %v", err)
	}
	return built
}

const theSocialInsuranceRounding money.Yen = 5_000

const theGradeGap = 3

func awayFromTheSheetsCopy(want money.Yen) money.Yen {
	return theSocialInsuranceRounding + want*theGradeGap/100
}

var theYearsTheRulesHaveMovedSince = yearsFrom(2026)

const (
	planStart date.Year = 2018
	planEnd   date.Year = 2090
)

func yearsFrom(first date.Year) map[date.Year]bool {
	years := make(map[date.Year]bool, int(planEnd-first)+1)
	for y := first; y <= planEnd; y++ {
		years[y] = true
	}
	return years
}
