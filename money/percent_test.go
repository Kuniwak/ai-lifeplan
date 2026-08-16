package money

import (
	"testing"

	"pgregory.net/rapid"
)

func TestPercentShouldRoundTripAtTheGranularityATableIsWrittenIn(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hundredths := rapid.Int64Range(-100_000, 100_000).Draw(t, "hundredths")
		want := NewRate(hundredths, 10_000)

		written := want.Percent()
		got, err := ParsePercent(written)
		if err != nil {
			t.Fatalf("%v を %q と書いたが読み直せない: %v", want, written, err)
		}
		if got.Num()*want.Den() != want.Num()*got.Den() {
			t.Fatalf("%v を %q と書いて読み直すと %v になる", want, written, got)
		}
	})
}

func TestPercentShouldDropTowardsNought(t *testing.T) {
	for _, c := range []struct {
		rate Rate
		want string
	}{
		{NewRate(2_025, 100_000), "2.02%"},
		{NewRate(-2_025, 100_000), "-2.02%"},
		{NewRate(1, 3), "33.33%"},
		{NewRate(-1, 3), "-33.33%"},
		{NewRate(4, 100), "4.00%"},
		{NewRate(0, 100), "0.00%"},
		{NewRate(-150, 10_000), "-1.50%"},
	} {
		if got := c.rate.Percent(); got != c.want {
			t.Errorf("%v.Percent() = %q, want %q", c.rate, got, c.want)
		}
	}
}
