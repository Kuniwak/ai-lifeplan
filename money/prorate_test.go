package money_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/panictest"
)

func TestForMonthsShouldCutAnAnnualAmountToTheMonthsItIsOwedFor(t *testing.T) {
	for _, tc := range []struct {
		name   string
		annual money.Yen
		months int
		want   money.Yen
		why    string
	}{
		{name: "通年", annual: 120_000, months: 12, want: 120_000, why: "12 か月は年額そのもの"},
		{name: "半年", annual: 120_000, months: 6, want: 60_000},
		{name: "0 か月", annual: 120_000, months: 0, want: 0, why: "資格の無い年は 0 円"},
		{
			name: "端数はゼロへ落ちる", annual: 100, months: 7, want: 58,
			why: "100 × 7 ÷ 12 = 58.33。**保険料なら少なく見る側**で、最大 11 円",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.annual.ForMonths(tc.months); got != tc.want {
				t.Errorf("Yen(%d).ForMonths(%d) = %d, want %d（%s）",
					tc.annual, tc.months, got, tc.want, tc.why)
			}
		})
	}
}

func TestForMonthsShouldPanicOnAMonthCountThatIsNotAYearsWorth(t *testing.T) {
	for _, months := range []int{-1, 13} {
		t.Run(string(rune('a'+months+1)), func(t *testing.T) {
			if panictest.Recovered(func() { money.Yen(120_000).ForMonths(months) }) == nil {
				t.Errorf("ForMonths(%d) が通った", months)
			}
		})
	}
}
