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

const (
	theLandValue  money.Yen = 12_805_000
	theHouseBase  money.Yen = 8_885_000
	theBuiltIn    date.Year = 2022
	theAssessedIn date.Year = 2025
)

func propertyTaxOfTheBaseProject(t *testing.T) relation.Table[table.PropertyTaxRow] {
	t.Helper()

	depreciation := law.MustLoadDepreciationRates(t, os.DirFS("../"+law.LawDirectory))
	rates, err := law.LoadPropertyTaxTable(os.DirFS("../"+law.LawDirectory), "世田谷区")
	if err != nil {
		t.Fatalf("law.LoadPropertyTaxTable: %v", err)
	}

	built, err := table.PropertyTaxTable(table.PropertyTaxInput{
		Calendar:     calendarOfTheBaseProject(t),
		BuiltIn:      theBuiltIn,
		LandValue:    theLandValue,
		HouseBaseAt:  theHouseBase,
		AssessedIn:   theAssessedIn,
		Depreciation: depreciation,
		Table:        rates,
	})
	if err != nil {
		t.Fatalf("table.PropertyTaxTable: %v", err)
	}
	return built
}

func TestNoPropertyTaxShouldBeChargedBeforeTheHouseIsBought(t *testing.T) {
	built := propertyTaxOfTheBaseProject(t)

	for _, year := range []date.Year{2018, 2021} {
		row, _ := built.At(year)
		if row.Total != 0 {
			t.Errorf("%d: %d charged although nothing was owned", year, row.Total)
		}
	}
	if row, _ := built.At(theBuiltIn); row.Total <= 0 {
		t.Errorf("%d: nothing charged in the year it was bought", theBuiltIn)
	}
}

func TestThePropertyTaxShouldRiseWhenTheNewHouseReliefEnds(t *testing.T) {
	built := propertyTaxOfTheBaseProject(t)

	last, _ := built.At(date.Year(2026))
	after, _ := built.At(date.Year(2027))

	if last.NewHouseRelief <= 0 {
		t.Error("2026 has no 新築住宅の減額 although it is the fifth year")
	}
	if after.NewHouseRelief != 0 {
		t.Errorf("2027 still has 新築住宅の減額 of %d", after.NewHouseRelief)
	}
	if after.PropertyTax <= last.PropertyTax {
		t.Errorf("固定資産税 did not rise when the relief ended: %d then %d",
			last.PropertyTax, after.PropertyTax)
	}
}
