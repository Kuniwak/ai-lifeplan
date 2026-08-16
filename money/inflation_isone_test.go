package money

import "testing"

func TestIsOneShouldNotBeFooledByAFactorTooSmallToRound(t *testing.T) {
	if !NoInflation().IsOne() {
		t.Error("NoInflation() が 1 でないと言っている")
	}
	var zero Factor
	if !zero.IsOne() {
		t.Error("ゼロ値の Factor が 1 でないと言っている")
	}

	tiny := NoInflation().After(NewRate(4, 10_000_000))
	if tiny.IsOne() {
		t.Error("1.0000004 を 1 だと言っている")
	}
	if tiny.Apply(1) != 1 || tiny.Apply(1_000_000) != 1_000_000 {
		t.Fatal("この検査の前提が崩れている。1 円と 100 万円は動かないはず")
	}
	if got := tiny.Apply(8_915_000); got == 8_915_000 {
		t.Error("頭金が動かない。この検査の前提が崩れている")
	}
}
