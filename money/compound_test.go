package money

import (
	"testing"

	"pgregory.net/rapid"
)

func TestCompoundShouldMultiplyRatherThanAdd(t *testing.T) {
	real, prices := NewRate(4, 100), NewRate(2, 100)

	if got := real.Compound(prices).Percent(); got != "6.08%" {
		t.Errorf("実質 4%% と物価 2%% を合わせると %q, 6.08%% のはず", got)
	}
}

func TestCompoundWithNothingShouldChangeNothing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		num := rapid.Int64Range(-9_000, 100_000).Draw(t, "num")
		r := NewRate(num, 10_000)

		got := r.Compound(NewRate(0, 100))
		if got.Num()*r.Den() != r.Num()*got.Den() {
			t.Fatalf("%v に 0%% を合わせると %v になった", r, got)
		}
	})
}

func TestCompoundShouldNotDependOnTheOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := NewRate(rapid.Int64Range(-9_000, 100_000).Draw(t, "a"), 10_000)
		b := NewRate(rapid.Int64Range(-9_000, 100_000).Draw(t, "b"), 10_000)

		left, right := a.Compound(b), b.Compound(a)
		if left.Num()*right.Den() != right.Num()*left.Den() {
			t.Fatalf("%v と %v を合わせる順番で %v と %v に分かれた", a, b, left, right)
		}
	})
}
