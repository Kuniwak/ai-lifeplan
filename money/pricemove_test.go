package money_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/money"
)

func TestParsePriceMoveShouldTellARatioFromADifference(t *testing.T) {
	for _, c := range []struct {
		field        string
		isDifference bool
		want         string
	}{
		{"119.5%", false, "119.50%"},
		{"0.0%", false, "0.00%"},
		{"+0.541pt", true, "+0.54pt"},
		{"-0.131pt", true, "-0.13pt"},
	} {
		t.Run(c.field, func(t *testing.T) {
			got, err := money.ParsePriceMove(c.field)
			if err != nil {
				t.Fatalf("money.ParsePriceMove(%q): %v", c.field, err)
			}
			if got.IsDifference() != c.isDifference {
				t.Errorf("%q の差かどうかが %v。%v のはず", c.field, got.IsDifference(), c.isDifference)
			}
			if got.String() != c.want {
				t.Errorf("%q が %q に戻った。%q のはず", c.field, got.String(), c.want)
			}
		})
	}
}

func TestParsePriceMoveShouldRefuseADifferenceWithNoSign(t *testing.T) {
	for _, field := range []string{"0.541pt", "0.0pt", "pt"} {
		if _, err := money.ParsePriceMove(field); err == nil {
			t.Errorf("%q を受け付けている", field)
		}
	}

	for _, field := range []string{"+-0.5pt", "++0.5pt", "-+0.5pt", "+ 0.541pt", "+0.5 pt"} {
		if _, err := money.ParsePriceMove(field); err == nil {
			t.Errorf("%q を受け付けている", field)
		}
	}
	for _, field := range []string{"119.5", "abc", "", "+0.5%"} {
		if _, err := money.ParsePriceMove(field); err == nil {
			t.Errorf("%q を受け付けている", field)
		}
	}
}

func TestAppliedShouldScaleARatioWithTheIndexAndNotADifference(t *testing.T) {
	slow := money.NewRate(41, 10_000)
	fast := money.NewRate(200, 10_000)

	ratio := money.RatioMove(money.NewRate(2613, 1000))
	difference := money.DifferenceMove(money.NewRate(67, 10_000))

	if got, want := ratio.Applied(slow).Percent(), "1.07%"; got != want {
		t.Errorf("比 261.3%% を 0.41%% に当てて %s。%s のはず", got, want)
	}
	if got, want := ratio.Applied(fast).Percent(), "5.22%"; got != want {
		t.Errorf("比 261.3%% を 2.00%% に当てて %s。%s のはず", got, want)
	}

	if got, want := difference.Applied(slow).Percent(), "1.08%"; got != want {
		t.Errorf("差 +0.67pt を 0.41%% に当てて %s。%s のはず", got, want)
	}
	if got, want := difference.Applied(fast).Percent(), "2.67%"; got != want {
		t.Errorf("差 +0.67pt を 2.00%% に当てて %s。%s のはず", got, want)
	}
}

func TestPlusShouldAddRates(t *testing.T) {
	for _, c := range []struct{ a, b, want string }{
		{"2.00%", "0.54%", "2.54%"},
		{"2.00%", "-0.13%", "1.87%"},
		{"0.00%", "0.54%", "0.54%"},
	} {
		a, err := money.ParsePercent(c.a)
		if err != nil {
			t.Fatal(err)
		}
		b, err := money.ParsePercent(c.b)
		if err != nil {
			t.Fatal(err)
		}
		if got := a.Plus(b).Percent(); got != c.want {
			t.Errorf("%s + %s = %s, want %s", c.a, c.b, got, c.want)
		}
	}
}

func TestParsePriceMoveShouldRefuseADifferenceThatMakesPricesNegative(t *testing.T) {
	for _, field := range []string{"-200.0pt", "-100.0pt", "-150.5pt"} {
		if _, err := money.ParsePriceMove(field); err == nil {
			t.Errorf("%q を受け付けている", field)
		}
	}
	if _, err := money.ParsePriceMove("-99.99pt"); err != nil {
		t.Errorf("-99.99pt を拒んでいる: %v", err)
	}
}
