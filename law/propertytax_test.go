package law_test

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

func propertyTaxTable(t *testing.T) law.PropertyTaxTable {
	t.Helper()

	table, err := law.LoadPropertyTaxTable(os.DirFS("../"+law.LawDirectory), setagaya)
	if err != nil {
		t.Fatalf("law.LoadPropertyTaxTable: %v", err)
	}
	return table
}

func TestTheNewHouseReliefShouldRunOutAfterItsYears(t *testing.T) {
	table := propertyTaxTable(t)
	const (
		landValue money.Yen = 14_000_000
		houseBase money.Yen = 9_000_000
	)

	last, err := table.Bill(landValue, houseBase, 2, 2024)
	if err != nil {
		t.Fatalf("Bill: %v", err)
	}
	after, err := table.Bill(landValue, houseBase, 3, 2025)
	if err != nil {
		t.Fatalf("Bill: %v", err)
	}

	if last.NewHouseRelief <= 0 {
		t.Errorf("the third year has no relief")
	}
	if after.NewHouseRelief != 0 {
		t.Errorf("the fourth year still has relief of %d", after.NewHouseRelief)
	}
	if after.PropertyTax <= last.PropertyTax {
		t.Errorf("the tax did not rise when the relief ran out: %d then %d", last.PropertyTax, after.PropertyTax)
	}
}

func TestTheReliefShouldNotTouchTheCityPlanningTax(t *testing.T) {
	table := propertyTaxTable(t)
	const (
		landValue money.Yen = 12_805_000
		houseBase money.Yen = 8_885_000
	)

	with, err := table.Bill(landValue, houseBase, 1, 2023)
	if err != nil {
		t.Fatalf("Bill: %v", err)
	}
	without, err := table.Bill(landValue, houseBase, 9, 2031)
	if err != nil {
		t.Fatalf("Bill: %v", err)
	}

	if with.CityPlanningTax != without.CityPlanningTax {
		t.Errorf("都市計画税 moved with the relief: %d then %d", with.CityPlanningTax, without.CityPlanningTax)
	}
}
