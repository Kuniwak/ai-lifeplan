package money_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/panictest"
)

func TestRateDivShouldSpreadARateOverThatManyPeriods(t *testing.T) {
	annual, err := money.ParsePercent("2.00%")
	if err != nil {
		t.Fatalf("ParsePercent: %v", err)
	}

	monthly := annual.Div(12)

	if got, want := money.Yen(1_000_000).Mul(monthly, money.Truncate), money.Yen(1_666); got != want {
		t.Errorf("a month's interest = %d, want %d", got, want)
	}
}

func TestRateDivShouldRefuseNothing(t *testing.T) {
	if panictest.Recovered(func() { money.NewRate(2, 100).Div(0) }) == nil {
		t.Error("Div(0) did not panic; a rate spread over no periods is not a rate")
	}
}

func TestRateFloat64ShouldGiveTheRateAsAFraction(t *testing.T) {
	if got, want := money.NewRate(1, 4).Float64(), 0.25; got != want {
		t.Errorf("Float64() = %v, want %v", got, want)
	}
}
