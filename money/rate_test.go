package money

import (
	"fmt"
	"testing"

	"github.com/Kuniwak/lifeplan/panictest"
	"github.com/google/go-cmp/cmp"
)

func TestParsePercentNG(t *testing.T) {
	testCases := map[string]string{
		"empty":                "",
		"missing percent sign": "8",
		"not a number":         "abc%",
		"percent sign only":    "%",
		"double percent sign":  "8%%",
	}

	for name, input := range testCases {
		t.Run(name, func(t *testing.T) {

			_, err := ParsePercent(input)

			if err == nil {
				t.Errorf("ParsePercent(%q): want error, got none", input)
			}
		})
	}
}

func TestYenMulShouldPanicWhenNoRoundingIsGiven(t *testing.T) {
	r, err := ParsePercent("8%")
	if err != nil {
		t.Fatalf("ParsePercent: %v", err)
	}
	refused := panictest.Recovered(func() { Yen(1000).Mul(r, nil) })

	if refused == nil {
		t.Error("want panic when the rounding is nil, got none")
	}
}

func TestShareOfShouldNotOverflow(t *testing.T) {
	const huge Yen = 3_000_000_000
	if got, want := ShareOf(huge, huge/4, huge), huge/4; got != want {
		t.Errorf("30 億円の 1/4 が %v になった（%v のはず）", got, want)
	}
}

func TestShareOfAnEmptyWhole(t *testing.T) {
	if got := ShareOf(1_000_000, 0, 0); got != 0 {
		t.Errorf("全体がゼロなのに %v を返した", got)
	}
}

func TestTimesShouldScaleARateAndNotFollowIt(t *testing.T) {
	index := NewRate(2, 100)
	ratio := NewRate(1188, 1000)

	scaled := index.Times(ratio)

	if want := NewRate(297, 12500); scaled.Cmp(want) != 0 {
		t.Errorf("2%% の 118.8%% が %s、%s のはず", scaled, want)
	}
	if got := scaled.Num(); got != 297 {
		t.Errorf("分子が %d である。約分されていない", got)
	}
	if followed := index.Compound(ratio); followed.Cmp(scaled) == 0 {
		t.Errorf("Compound と Times が同じ答え %s を出している", followed)
	}
}

func TestNewPercentShouldApplyLikeTheSameRateReadFromATable(t *testing.T) {
	type testCase struct {
		Percent int64
		Amount  Yen
	}

	testCases := map[string]testCase{
		"零 (boundary value)":    {Percent: 0, Amount: 1234567},
		"3% (消費税)":              {Percent: 3, Amount: 1234567},
		"8% (給与所得控除)":           {Percent: 8, Amount: 1234567},
		"40% (育児休業給付の後半)":       {Percent: 40, Amount: 1234567},
		"100% (boundary value)": {Percent: 100, Amount: 1234567},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			written := fmt.Sprintf("%d%%", tc.Percent)
			read, err := ParsePercent(written)
			if err != nil {
				t.Fatalf("ParsePercent(%q): want no error, got %v", written, err)
			}

			got := tc.Amount.Mul(NewPercent(tc.Percent), Truncate)

			if diff := cmp.Diff(tc.Amount.Mul(read, Truncate), got); diff != "" {
				t.Errorf("NewPercent(%d) applied to %d, against %q (-want +got):\n%s",
					tc.Percent, tc.Amount, written, diff)
			}
		})
	}
}
