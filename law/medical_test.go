package law_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

func TestTheMedicalDeductionShouldTakeOffAHundredThousandFromAnOrdinaryIncome(t *testing.T) {
	if got, want := law.MedicalDeduction(600_000, 0, 7_000_000), money.Yen(500_000); got != want {
		t.Errorf("医療費控除 = %d, want %d", got, want)
	}
}

func TestTheMedicalDeductionShouldTakeOffFivePercentOfASmallIncome(t *testing.T) {
	if got, want := law.MedicalDeduction(600_000, 0, 1_000_000), money.Yen(550_000); got != want {
		t.Errorf("医療費控除 = %d, want %d", got, want)
	}
}

func TestTheMedicalDeductionShouldTakeTheRefundOffFirst(t *testing.T) {
	if got := law.MedicalDeduction(600_000, 600_000, 7_000_000); got != 0 {
		t.Errorf("医療費控除 = %d, want nothing when the whole bill came back", got)
	}
	if got, want := law.MedicalDeduction(600_000, 300_000, 7_000_000), money.Yen(200_000); got != want {
		t.Errorf("医療費控除 = %d, want %d", got, want)
	}
}

func TestTheMedicalDeductionShouldStopAtTwoMillion(t *testing.T) {
	if got, want := law.MedicalDeduction(9_000_000, 0, 7_000_000), law.MedicalDeductionCap; got != want {
		t.Errorf("医療費控除 = %d, want the ceiling %d", got, want)
	}
}

func TestASmallBillShouldEarnNoMedicalDeduction(t *testing.T) {
	if got := law.MedicalDeduction(40_000, 0, 7_000_000); got != 0 {
		t.Errorf("医療費控除 = %d, want nothing", got)
	}
}
