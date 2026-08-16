package money

import (
	"math/big"
	"testing"

	"pgregory.net/rapid"
)

func genInflation() *rapid.Generator[Rate] {
	return rapid.Custom(func(t *rapid.T) Rate {
		return NewRate(rapid.Int64Range(0, 1_000).Draw(t, "num"), 10_000)
	})
}

func genPlanYears() *rapid.Generator[int] {
	return rapid.IntRange(0, 80)
}

func grown(r Rate, periods int) Factor {
	f := NoInflation()
	for range periods {
		f = f.After(r)
	}
	return f
}

func TestFactorOverNoYearsShouldChangeNothing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")
		r := genInflation().Draw(t, "r")

		if got := grown(r, 0).Apply(y); got != y {
			t.Fatalf("Factor(%v, 0 年).Apply(%d) = %d, 0 年で額が変わった", r, y, got)
		}
	})
}

func TestFactorAtNoInflationShouldChangeNothing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")
		years := genPlanYears().Draw(t, "years")

		if got := grown(NewRate(0, 100), years).Apply(y); got != y {
			t.Fatalf("Factor(0%%, %d 年).Apply(%d) = %d, 率 0 で額が変わった", years, y, got)
		}
	})
}

func TestFactorShouldBeMonotoneInTheYears(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")
		r := genInflation().Draw(t, "r")
		years := genPlanYears().Draw(t, "years")

		earlier := grown(r, years).Apply(y)
		later := grown(r, years+1).Apply(y)

		switch {
		case y > 0 && later < earlier:
			t.Fatalf("%d 円を %v で %d 年伸ばした %d が、%d 年の %d より小さい", y, r, years+1, later, years, earlier)
		case y < 0 && later > earlier:
			t.Fatalf("%d 円を %v で %d 年伸ばした %d が、%d 年の %d より大きい", y, r, years+1, later, years, earlier)
		case y == 0 && later != 0:
			t.Fatalf("0 円を %v で %d 年伸ばしたら %d になった", r, years+1, later)
		}
	})
}

func TestFactorShouldStayWithinHalfAYenOfTheExactValue(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")
		r := genInflation().Draw(t, "r")
		years := genPlanYears().Draw(t, "years")

		got := grown(r, years).Apply(y)

		exact := exactGrowth(y, r, years)
		diff := new(big.Rat).Sub(new(big.Rat).SetInt64(int64(got)), exact)
		if diff.Abs(diff).Cmp(big.NewRat(1, 2)) > 0 {
			t.Fatalf("%d 円を %v で %d 年伸ばすと %d, 厳密値 %v から半円を超えて離れている",
				y, r, years, got, exact.FloatString(4))
		}
	})
}

func exactGrowth(y Yen, r Rate, years int) *big.Rat {
	factor := new(big.Rat).SetFrac64(r.num+r.den, r.den)
	moved := new(big.Rat).SetInt64(int64(y))
	for range years {
		moved.Mul(moved, factor)
	}
	return moved
}

func TestFactorShouldNotDependOnTheOrderOfTheRates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")
		rates := rapid.SliceOfN(genInflation(), 0, 20).Draw(t, "rates")

		forwards, backwards := NoInflation(), NoInflation()
		for i := range rates {
			forwards = forwards.After(rates[i])
			backwards = backwards.After(rates[len(rates)-1-i])
		}

		if forwards.Apply(y) != backwards.Apply(y) {
			t.Fatalf("%d 円に率を順に、逆順にかけると %d と %d で食い違う",
				y, forwards.Apply(y), backwards.Apply(y))
		}
	})
}

func TestHalfUpBigShouldAgreeWithHalfUp(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.Int64Range(-1_000_000, 1_000_000).Draw(t, "n")
		d := rapid.Int64Range(1, 1_000_000).Draw(t, "d")

		want := HalfUp(n, d)
		got := halfUpBig(new(big.Rat).SetFrac64(n, d)).Int64()

		if got != want {
			t.Fatalf("halfUpBig(%d/%d) = %d, HalfUp は %d", n, d, got, want)
		}
	})
}

func TestDeflateShouldUndoTheGrowthToWithinAYen(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")
		r := genInflation().Draw(t, "r")
		years := genPlanYears().Draw(t, "years")

		f := grown(r, years)
		back := f.Deflate(f.Apply(y))

		if diff := back - y; diff > 1 || diff < -1 {
			t.Fatalf("%d 円を %v で %d 年伸ばして戻すと %d になった", y, r, years, back)
		}
	})
}

func TestDeflateAtNoInflationShouldChangeNothing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")

		if got := NoInflation().Deflate(y); got != y {
			t.Fatalf("物価が動いていないのに %d が %d になった", y, got)
		}
	})
}
