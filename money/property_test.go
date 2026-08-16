package money

import (
	"testing"

	"pgregory.net/rapid"
)

func genYen() *rapid.Generator[Yen] {
	return rapid.Custom(func(t *rapid.T) Yen {
		return Yen(rapid.Int64Range(-1_000_000_000_000, 1_000_000_000_000).Draw(t, "yen"))
	})
}

func genUnit() *rapid.Generator[Yen] {
	return rapid.Custom(func(t *rapid.T) Yen {
		return Yen(rapid.Int64Range(1, 1_000_000).Draw(t, "unit"))
	})
}

func TestTruncateShouldNeverExceedTheOriginal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")
		unit := genUnit().Draw(t, "unit")

		got := y.Truncate(unit)

		if got > y {
			t.Fatalf("Yen(%d).Truncate(%d) = %d, which is larger than the original", y, unit, got)
		}
	})
}

func TestTruncateShouldStayWithinOneUnitOfTheOriginal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")
		unit := genUnit().Draw(t, "unit")

		got := y.Truncate(unit)

		if y-got >= unit {
			t.Fatalf("Yen(%d).Truncate(%d) = %d, which discards a whole unit or more", y, unit, got)
		}
	})
}

func TestTruncateShouldBeMonotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := genYen().Draw(t, "a")
		b := genYen().Draw(t, "b")
		unit := genUnit().Draw(t, "unit")
		if a > b {
			a, b = b, a
		}

		gotA, gotB := a.Truncate(unit), b.Truncate(unit)

		if gotA > gotB {
			t.Fatalf("Truncate is not monotonic: Yen(%d) <= Yen(%d) but %d > %d (unit %d)", a, b, gotA, gotB, unit)
		}
	})
}

func TestTruncateShouldBeIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")
		unit := genUnit().Draw(t, "unit")

		once := y.Truncate(unit)
		twice := once.Truncate(unit)

		if once != twice {
			t.Fatalf("Truncate is not idempotent: %d then %d (unit %d)", once, twice, unit)
		}
	})
}

func TestTruncateShouldProduceAMultipleOfTheUnit(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")
		unit := genUnit().Draw(t, "unit")

		got := y.Truncate(unit)

		if got%unit != 0 {
			t.Fatalf("Yen(%d).Truncate(%d) = %d, which is not a multiple of the unit", y, unit, got)
		}
	})
}

func TestParseYenShouldRoundTripThroughString(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")

		got, err := ParseYen(y.String())

		if err != nil {
			t.Fatalf("ParseYen(%q): %v", y.String(), err)
		}
		if got != y {
			t.Fatalf("round trip changed the amount: %d -> %q -> %d", y, y.String(), got)
		}
	})
}

func TestRoundingsShouldBracketEachOther(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		y := genYen().Draw(t, "y")
		num := rapid.Int64Range(0, 100_000).Draw(t, "num")
		r := NewRate(num, 100_000)

		down := y.Mul(r, Truncate)
		near := y.Mul(r, HalfUp)
		up := y.Mul(r, Ceil)

		if down > near || near > up {
			t.Fatalf("roundings out of order for Yen(%d) x %s: truncate=%d halfUp=%d ceil=%d", y, r, down, near, up)
		}
		if up-down > 1 {
			t.Fatalf("truncate and ceil differ by more than one yen for Yen(%d) x %s: %d vs %d", y, r, down, up)
		}
	})
}

func TestMulShouldBeMonotonicInTheAmountForANonNegativeRate(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := genYen().Draw(t, "a")
		b := genYen().Draw(t, "b")
		if a > b {
			a, b = b, a
		}
		num := rapid.Int64Range(0, 100_000).Draw(t, "num")
		r := NewRate(num, 100_000)

		gotA, gotB := a.Mul(r, Truncate), b.Mul(r, Truncate)

		if gotA > gotB {
			t.Fatalf("Mul is not monotonic: Yen(%d) <= Yen(%d) but %d > %d (rate %s)", a, b, gotA, gotB, r)
		}
	})
}
