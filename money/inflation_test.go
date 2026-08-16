package money_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/money"
)

func TestFactorShouldCompoundOneYearAtTheRate(t *testing.T) {
	factor := money.NoInflation().After(money.NewRate(2, 100))

	if got, want := factor.Apply(100), money.Yen(102); got != want {
		t.Errorf("100円を年 2%% で 1 年伸ばすと %v のはずが %v", want, got)
	}
}

func TestFactorShouldNotOverflowOverThePlansLength(t *testing.T) {
	const monthly, months, years = 250_000, 12, 68

	factor := money.NoInflation()
	for range years {
		factor = factor.After(money.NewRate(2, 100))
	}

	got := factor.Apply(monthly * months)
	if got < 11_000_000 || got > 12_000_000 {
		t.Errorf("月 25 万円の生活費を年 2%% で 68 年伸ばすと 1,100 万〜1,200 万円のはずが %v", got)
	}
}

func TestFactorComposeShouldAgreeWithCompoundingTheRates(t *testing.T) {
	two, one := money.NewRate(2, 100), money.NewRate(1, 100)

	prices, wages, together := money.NoInflation(), money.NoInflation(), money.NoInflation()
	for range 68 {
		prices = prices.After(two)
		wages = wages.After(one)
		together = together.After(two.Compound(one))
	}

	if got, want := prices.Compose(wages).String(), together.String(); got != want {
		t.Errorf("物価と賃金を別々に積んで掛けると %s。年ごとに合成すると %s", got, want)
	}
}

func TestComposingWithNoInflationShouldChangeNothing(t *testing.T) {
	prices := money.NoInflation().After(money.NewRate(2, 100))

	if got, want := prices.Compose(money.NoInflation()).String(), prices.String(); got != want {
		t.Errorf("動かない因数を掛けたら %s になった。%s のはず", got, want)
	}
}

func TestComposingWithTheZeroFactorShouldReadItAsOne(t *testing.T) {
	prices := money.NoInflation().After(money.NewRate(2, 100))

	var zero money.Factor
	if got, want := zero.Compose(prices).String(), prices.String(); got != want {
		t.Errorf("ゼロ値を掛けたら %s になった。%s のはず", got, want)
	}
}
