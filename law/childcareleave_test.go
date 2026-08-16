package law_test

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

func childcareLeaveTable(t *testing.T) law.ChildcareLeaveBenefitTable {
	t.Helper()

	table := law.MustLoadChildcareLeaveBenefits(t, os.DirFS("../"+law.LawDirectory))
	return table
}

func TestTheChildcareLeaveBenefitShouldBeTwoThirdsOfThePayForTheFirstSixMonths(t *testing.T) {
	table := childcareLeaveTable(t)

	got, err := table.Benefit(200_000, 2, 2022)
	if err != nil {
		t.Fatalf("Benefit: %v", err)
	}
	if want := money.Yen(268_000); got != want {
		t.Errorf("two months of leave = %d, want %d", got, want)
	}
}

func TestTheChildcareLeaveBenefitShouldStopAtTheCeiling(t *testing.T) {
	table := childcareLeaveTable(t)

	got, err := table.Benefit(810_000, 2, 2022)
	if err != nil {
		t.Fatalf("Benefit: %v", err)
	}
	if want := money.Yen(301_902 * 2); got != want {
		t.Errorf("two months of leave = %d, want %d", got, want)
	}
}

func TestTheChildcareLeaveBenefitShouldDropToHalfAfterSixMonths(t *testing.T) {
	table := childcareLeaveTable(t)

	got, err := table.Benefit(200_000, 8, 2022)
	if err != nil {
		t.Fatalf("Benefit: %v", err)
	}
	if want := money.Yen(6*134_000 + 2*100_000); got != want {
		t.Errorf("eight months of leave = %d, want %d", got, want)
	}
}

func TestTheChildcareLeaveBenefitShouldRiseWithTheCeiling(t *testing.T) {
	table := childcareLeaveTable(t)

	before, err := table.Benefit(810_000, 1, 2025)
	if err != nil {
		t.Fatalf("Benefit: %v", err)
	}
	after, err := table.Benefit(810_000, 1, 2026)
	if err != nil {
		t.Fatalf("Benefit: %v", err)
	}

	if before != 315_369 || after != 323_811 {
		t.Errorf("the ceiling went %d -> %d, want %d -> %d", before, after, 315_369, 323_811)
	}
}

func TestNoLeaveShouldEarnNoBenefit(t *testing.T) {
	table := childcareLeaveTable(t)

	got, err := table.Benefit(810_000, 0, 2022)
	if err != nil {
		t.Fatalf("Benefit: %v", err)
	}
	if got != 0 {
		t.Errorf("no leave earned %d", got)
	}
}
