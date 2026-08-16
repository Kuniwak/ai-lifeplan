package law_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

func TestPensionStartAdjustmentShouldReduceByFourTenthsOfAPercentAMonthWhenDrawnEarly(t *testing.T) {
	for _, tt := range []struct {
		months int
		want   money.Rate
	}{
		{-60, money.NewRate(760, 1_000)},
		{-12, money.NewRate(952, 1_000)},
		{0, money.NewRate(1_000, 1_000)},
		{60, money.NewRate(1_420, 1_000)},
		{120, money.NewRate(1_840, 1_000)},
	} {
		got, err := law.PensionStartAdjustment(tt.months)
		if err != nil {
			t.Errorf("%d か月: %v", tt.months, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%d か月の率が %v、欲しいのは %v", tt.months, got, tt.want)
		}
	}
}

func TestPensionStartAdjustmentShouldRefuseBeforeSixty(t *testing.T) {
	if _, err := law.PensionStartAdjustment(-61); err == nil {
		t.Error("-61 か月（59 歳 11 か月）が通った")
	}
	if _, err := law.PensionStartAdjustment(121); err == nil {
		t.Error("121 か月（75 歳 1 か月）が通った")
	}
}
