package relation_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/relation"
)

func TestAPeriodShouldCoverBothItsEndsAndNothingOutsideThem(t *testing.T) {
	type testCase struct {
		Period relation.Period[int, string]
		Year   int
		Want   bool
	}

	closed := relation.NewPeriod(relation.From(2022), relation.To(2025), "closed")
	openStart := relation.NewPeriod(relation.Unbounded[int](), relation.To(2021), "不明から")
	openEnd := relation.NewPeriod(relation.From(2026), relation.Unbounded[int](), "無期限まで")
	openBoth := relation.NewPeriod(relation.Unbounded[int](), relation.Unbounded[int](), "どちらも")

	testCases := map[string]testCase{
		"始まりの年（境界値）":     {Period: closed, Year: 2022, Want: true},
		"始まりの 1 年前（境界値）": {Period: closed, Year: 2021, Want: false},
		"終わりの年（境界値）":     {Period: closed, Year: 2025, Want: true},
		"終わりの 1 年後（境界値）": {Period: closed, Year: 2026, Want: false},
		"始まりが開いていれば下は無い": {Period: openStart, Year: -3000, Want: true},
		"始まりが開いていても上は効く": {Period: openStart, Year: 2022, Want: false},
		"終わりが開いていれば上は無い": {Period: openEnd, Year: 9999, Want: true},
		"終わりが開いていても下は効く": {Period: openEnd, Year: 2025, Want: false},
		"どちらも開いていれば何でも":  {Period: openBoth, Year: 0, Want: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := tc.Period.Covers(tc.Year)

			if got != tc.Want {
				t.Errorf("%v.Covers(%d) = %t, want %t", tc.Period, tc.Year, got, tc.Want)
			}
		})
	}
}

func TestOverlapShouldNameAYearBothPeriodsCover(t *testing.T) {
	type testCase struct {
		A, B relation.Period[int, string]
		Want bool
	}

	testCases := map[string]testCase{
		"隣り合っていて重ならない（境界値）": {
			A:    relation.NewPeriod(relation.From(2000), relation.To(2021), "前"),
			B:    relation.NewPeriod(relation.From(2022), relation.Unbounded[int](), "後"),
			Want: false,
		},
		"1 年だけ重なる（境界値）": {
			A:    relation.NewPeriod(relation.From(2000), relation.To(2022), "前"),
			B:    relation.NewPeriod(relation.From(2022), relation.Unbounded[int](), "後"),
			Want: true,
		},
		"どちらも両端が開いている": {
			A:    relation.NewPeriod(relation.Unbounded[int](), relation.Unbounded[int](), "甲"),
			B:    relation.NewPeriod(relation.Unbounded[int](), relation.Unbounded[int](), "乙"),
			Want: true,
		},
		"片方が丸ごと入っている": {
			A:    relation.NewPeriod(relation.From(2000), relation.To(2030), "外"),
			B:    relation.NewPeriod(relation.From(2010), relation.To(2011), "内"),
			Want: true,
		},
		"下が開いた帯と上が開いた帯がすれ違う": {
			A:    relation.NewPeriod(relation.Unbounded[int](), relation.To(2000), "下"),
			B:    relation.NewPeriod(relation.From(2001), relation.Unbounded[int](), "上"),
			Want: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			year, got := relation.Overlap(tc.A, tc.B)

			if got != tc.Want {
				t.Fatalf("relation.Overlap(%v, %v) = %d, %t, want %t", tc.A, tc.B, year, got, tc.Want)
			}
			if got && !(tc.A.Covers(year) && tc.B.Covers(year)) {
				t.Errorf("%d 年が重なっていると言われたが、どちらかがその年を覆っていない", year)
			}
		})
	}
}

func TestPeriodsShouldMissTheYearsNobodyWrote(t *testing.T) {
	type testCase struct {
		Year  int
		Want  string
		Found bool
	}

	testCases := map[string]testCase{
		"下が開いた帯は、はるか前も覆う": {Year: -3000, Want: "前", Found: true},
		"その帯の終わりの年（境界値）":  {Year: 2021, Want: "前", Found: true},
		"穴の 1 年目（境界値）":    {Year: 2022, Found: false},
		"穴の最後の年（境界値）":     {Year: 2025, Found: false},
		"次の帯の始まりの年（境界値）":  {Year: 2026, Want: "後", Found: true},
		"上が開いた帯は、はるか先も覆う": {Year: 9999, Want: "後", Found: true},
	}

	periods, err := relation.NewPeriods([]relation.Period[int, string]{
		relation.NewPeriod(relation.From(2026), relation.Unbounded[int](), "後"),
		relation.NewPeriod(relation.Unbounded[int](), relation.To(2021), "前"),
	})
	if err != nil {
		t.Fatalf("relation.NewPeriods: %v", err)
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, found := periods.Lookup(tc.Year)

			if found != tc.Found || got != tc.Want {
				t.Errorf("Lookup(%d) = %q, %t, want %q, %t", tc.Year, got, found, tc.Want, tc.Found)
			}
		})
	}
}

func TestNewPeriodsShouldRefuseATableThatCannotBeReadOffAtAll(t *testing.T) {
	testCases := map[string][]relation.Period[int, string]{
		"2 つの行が同じ年を名乗る": {
			relation.NewPeriod(relation.From(2000), relation.To(2022), "甲"),
			relation.NewPeriod(relation.From(2022), relation.To(2030), "乙"),
		},
		"終わりが始まりより前": {
			relation.NewPeriod(relation.From(2022), relation.To(2021), "甲"),
		},
	}

	for name, periods := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := relation.NewPeriods(periods)

			if err == nil {
				t.Error("組み立てが通ってしまった")
			}
		})
	}
}

func TestNewPeriodsShouldNotShareTheCallersSlice(t *testing.T) {
	given := []relation.Period[int, string]{
		relation.NewPeriod(relation.Unbounded[int](), relation.To(2021), "前"),
		relation.NewPeriod(relation.From(2026), relation.Unbounded[int](), "後"),
	}
	periods, err := relation.NewPeriods(given)
	if err != nil {
		t.Fatalf("relation.NewPeriods: %v", err)
	}

	given[0] = relation.NewPeriod(relation.Unbounded[int](), relation.Unbounded[int](), "すり替え")
	periods.All()[1] = relation.NewPeriod(relation.Unbounded[int](), relation.Unbounded[int](), "すり替え")

	if got, _ := periods.Lookup(2021); got != "前" {
		t.Errorf("組み立てたあとの書き換えが表に届いている: %q", got)
	}
	if got, found := periods.Lookup(2023); found {
		t.Errorf("誰も書いていない 2023 年が %q になっている", got)
	}
}

func TestBandsOfPeriodsShouldRefuseWhatBandsCannotSay(t *testing.T) {
	type testCase struct {
		Periods []relation.Period[int, string]
		Wants   bool
	}

	testCases := map[string]testCase{
		"隙間なく連なり、最後が開いている": {
			Periods: []relation.Period[int, string]{
				relation.NewPeriod(relation.Unbounded[int](), relation.To(2021), "前"),
				relation.NewPeriod(relation.From(2022), relation.Unbounded[int](), "後"),
			},
			Wants: true,
		},
		"順不同でも同じ": {
			Periods: []relation.Period[int, string]{
				relation.NewPeriod(relation.From(2022), relation.Unbounded[int](), "後"),
				relation.NewPeriod(relation.Unbounded[int](), relation.To(2021), "前"),
			},
			Wants: true,
		},
		"穴がある": {
			Periods: []relation.Period[int, string]{
				relation.NewPeriod(relation.Unbounded[int](), relation.To(2021), "前"),
				relation.NewPeriod(relation.From(2023), relation.Unbounded[int](), "後"),
			},
		},
		"重なっている": {
			Periods: []relation.Period[int, string]{
				relation.NewPeriod(relation.Unbounded[int](), relation.To(2023), "前"),
				relation.NewPeriod(relation.From(2022), relation.Unbounded[int](), "後"),
			},
		},
		"最後が終わっている": {
			Periods: []relation.Period[int, string]{
				relation.NewPeriod(relation.Unbounded[int](), relation.To(2021), "前"),
				relation.NewPeriod(relation.From(2022), relation.To(2025), "後"),
			},
		},
		"1 つしかなく、それが終わっている": {
			Periods: []relation.Period[int, string]{
				relation.NewPeriod(relation.Unbounded[int](), relation.To(2021), "だけ"),
			},
		},
		"1 つも無い": {},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			bands, err := relation.BandsOfPeriods(tc.Periods, 1)

			switch {
			case tc.Wants && err != nil:
				t.Fatalf("帯にできるはずが %v", err)
			case !tc.Wants && err == nil:
				t.Fatal("断るはずが帯になってしまった")
			case tc.Wants && len(bands) != len(tc.Periods):
				t.Errorf("%d 本の帯になった。行は %d である", len(bands), len(tc.Periods))
			}
		})
	}
}
