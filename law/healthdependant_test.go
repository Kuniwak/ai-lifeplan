package law

import (
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
)

func TestTheHealthDependantCeilingShouldStepAtSixty(t *testing.T) {
	for _, tc := range []struct {
		age  int
		want int64
	}{
		{age: 0, want: 1_300_000},
		{age: 59, want: 1_300_000},
		{age: 60, want: 1_800_000},
		{age: 100, want: 1_800_000},
	} {
		if got := HealthDependantIncomeLimitAt(tc.age, DisabilityPensionNo); int64(got) != tc.want {
			t.Errorf("%d 歳の認定基準が %d 円（%d 円のはず）", tc.age, got, tc.want)
		}
	}
}

func TestTheCeilingShouldNotDependOnAgeForSomebodyOnTheDisabilityLimb(t *testing.T) {
	for _, age := range []int{0, 30, 59, 60, 100} {
		if got := HealthDependantIncomeLimitAt(age, DisabilityPensionYes); got != HealthDependantOlderIncomeLimit {
			t.Errorf("%d 歳・障害厚生年金の受給要件に該当する場合の認定基準が %d 円（%d 円のはず）",
				age, got, HealthDependantOlderIncomeLimit)
		}
	}
}

func TestTheDisabilityPensionAnswerShouldRefuseAWordNobodyWrote(t *testing.T) {
	for _, c := range []struct {
		word DisabilityPensionEligible
		want bool
		ok   bool
	}{
		{word: DisabilityPensionYes, want: true, ok: true},
		{word: DisabilityPensionNo, want: false, ok: true},
		{word: "", ok: false},
		{word: "はいはい", ok: false},
		{word: "yes", ok: false},
	} {
		got, err := c.word.Eligible()
		if c.ok {
			if err != nil {
				t.Errorf("Eligible(%q): %v", c.word, err)
			} else if got != c.want {
				t.Errorf("Eligible(%q) = %v, want %v", c.word, got, c.want)
			}
			continue
		}
		if err == nil {
			t.Errorf("Eligible(%q) = %v、断るはず", c.word, got)
		}
	}
}

func TestTheDisabilityPensionVocabularyShouldAgreeWithTheConstants(t *testing.T) {
	want := make([]string, 0, len(DisabilityPensionAnswers()))
	for _, a := range DisabilityPensionAnswers() {
		want = append(want, string(a))
	}
	got := slices.Clone(input.DisabilityPensionWords)
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("input.DisabilityPensionWords = %v, want %v", got, want)
	}
}
