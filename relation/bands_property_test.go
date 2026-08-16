package relation

import (
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/panictest"
	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"
)

func genBands() *rapid.Generator[Bands[int, int]] {
	return rapid.Custom(func(t *rapid.T) Bands[int, int] {
		lowers := rapid.SliceOfNDistinct(
			rapid.IntRange(1, 10_000_000),
			0, 12,
			func(v int) int { return v },
		).Draw(t, "lowers")

		bands := []Band[int, int]{{Lower: 0, Value: 0}}
		for _, l := range lowers {
			bands = append(bands, Band[int, int]{Lower: l, Value: l})
		}
		return NewBands(bands)
	})
}

func TestLookupShouldAlwaysHitATableThatStartsAtTheBottom(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bands := genBands().Draw(t, "bands")
		key := rapid.IntRange(0, 20_000_000).Draw(t, "key")

		if msg, refused := panictest.Message(func() { bands.Lookup(key) }); refused {
			t.Fatalf("Lookup(%d) missed a table covering everything from 0: %s", key, msg)
		}
	})
}

func TestLookupShouldReturnTheBandTheKeyFallsIn(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bands := genBands().Draw(t, "bands")
		key := rapid.IntRange(0, 20_000_000).Draw(t, "key")

		got := bands.Lookup(key)

		want := 0
		for _, b := range bands.bands {
			if b.Lower <= key && b.Lower > want {
				want = b.Lower
			}
		}
		if got != want {
			t.Fatalf("Lookup(%d) = %d, want the band starting at %d", key, got, want)
		}
	})
}

func TestLookupShouldBeMonotonicWhenTheValuesAre(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bands := genBands().Draw(t, "bands")
		a := rapid.IntRange(0, 20_000_000).Draw(t, "a")
		b := rapid.IntRange(0, 20_000_000).Draw(t, "b")
		if a > b {
			a, b = b, a
		}

		gotA, gotB := bands.Lookup(a), bands.Lookup(b)

		if gotA > gotB {
			t.Fatalf("a rising table gave a falling answer: Lookup(%d)=%d but Lookup(%d)=%d", a, gotA, b, gotB)
		}
	})
}

func TestLagShouldKeepExactlyTheLeftYears(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		left := genYears().Draw(t, "left")
		right := genYears().Draw(t, "right")

		got := Lag(tableOf(left), tableOf(right), -1, func(y date.Year, l, prev int) int { return prev })

		if diff := cmp.Diff(tableOf(left).Years(), got.Years()); diff != "" {
			t.Fatalf("Lag changed the year set (-want +got):\n%s", diff)
		}
	})
}

func TestLagShouldStandInExactlyWhenThePreviousYearIsAbsent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		left := genYears().Draw(t, "left")
		right := genYears().Draw(t, "right")
		const standIn = -1

		got := Lag(tableOf(left), tableOf(right), standIn, func(y date.Year, l, prev int) int { return prev })

		for _, row := range got.Rows() {
			hasPrevious := slices.Contains(right, row.Year-1)
			if !hasPrevious && row.Value != standIn {
				t.Fatalf("year %d has no previous year on the right but the stand-in was not used: got %d", row.Year, row.Value)
			}
			if hasPrevious && row.Value == standIn {
				t.Fatalf("year %d has a previous year on the right but the stand-in was used", row.Year)
			}
		}
	})
}
