package law

import (
	"os"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/money"
)

func depreciationTable(t *testing.T) DepreciationRateTable {
	t.Helper()

	parsed := MustLoadDepreciationRates(t, os.DirFS("../"+LawDirectory))
	return parsed
}

func TestDepreciationRate(t *testing.T) {
	type testCase struct {
		YearsElapsed int
		Expected     money.Rate
	}

	testCases := map[string]testCase{
		"brand new (boundary)":                   {YearsElapsed: 0, Expected: money.NewRate(1, 1)},
		"after one year (boundary)":              {YearsElapsed: 1, Expected: money.NewRate(8_000, 10_000)},
		"the three flat years end (boundary)":    {YearsElapsed: 3, Expected: money.NewRate(7_000, 10_000)},
		"the first year that slopes (boundary)":  {YearsElapsed: 4, Expected: money.NewRate(6_815, 10_000)},
		"halfway down":                           {YearsElapsed: 20, Expected: money.NewRate(3_852, 10_000)},
		"the floor is reached (boundary)":        {YearsElapsed: 30, Expected: money.NewRate(2_000, 10_000)},
		"past the table it stays at the floor":   {YearsElapsed: 76, Expected: money.NewRate(2_000, 10_000)},
		"a negative age is treated as brand new": {YearsElapsed: -1, Expected: money.NewRate(1, 1)},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := depreciationTable(t).Rate(tc.YearsElapsed)

			if got != tc.Expected {
				t.Errorf("経年減点補正率: 経過年数 %d は %v のはずだが %v になった", tc.YearsElapsed, tc.Expected, got)
			}
		})
	}
}

func TestDepreciationRateShouldNotMatchTheSpreadsheet(t *testing.T) {
	loaded := depreciationTable(t)

	if got := loaded.Rate(1); got == money.NewRate(9_579, 10_000) {
		t.Error("経過年数 1 が 0.9579 に戻っている。写しの値であって、評価基準のどの列でもない")
	}
	if got, want := loaded.Rate(1), money.NewRate(8_000, 10_000); got != want {
		t.Errorf("経過年数 1 の経年減点補正率 = %v, want %v（別表第13-2 の 1〜3 年は構造によらず一定）", got, want)
	}
}

func TestDepreciationRateShouldNeverRise(t *testing.T) {
	loaded := depreciationTable(t)

	rapid.Check(t, func(t *rapid.T) {
		earlier := rapid.IntRange(0, 200).Draw(t, "earlier")
		later := rapid.IntRange(0, 200).Draw(t, "later")
		if earlier > later {
			earlier, later = later, earlier
		}

		before := loaded.Rate(earlier)
		after := loaded.Rate(later)

		beforeValue := money.Yen(1_000_000).Mul(before, money.Truncate)
		afterValue := money.Yen(1_000_000).Mul(after, money.Truncate)
		if afterValue > beforeValue {
			t.Fatalf("a building worth %d after %d years is worth %d after %d", beforeValue, earlier, afterValue, later)
		}
		if floor := money.Yen(1_000_000).Mul(money.NewRate(2_000, 10_000), money.Truncate); afterValue < floor {
			t.Fatalf("after %d years the value %d has fallen below the floor %d", later, afterValue, floor)
		}
	})
}
