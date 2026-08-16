package law

import (
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
)

func TestInputVocabularyShouldAgreeWithTheConstants(t *testing.T) {
	want := []string{
		string(WhiteForm), string(BlueFormSimplified),
		string(BlueFormDoubleEntry), string(BlueFormDoubleEntryElectronic),
	}
	got := slices.Clone(input.BlueFormRecordKeepingWords)
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("input.BlueFormRecordKeepingWords = %v, want %v", got, want)
	}
}

func TestBlueFormDeduction(t *testing.T) {
	type testCase struct {
		Kind     BlueFormRecordKeeping
		Income   money.Yen
		Expected money.Yen
	}

	testCases := map[string]testCase{
		"white form deducts nothing, whatever the income": {
			Kind: WhiteForm, Income: 10_000_000, Expected: 0,
		},
		"simplified blue form deducts up to 十万円": {
			Kind: BlueFormSimplified, Income: 10_000_000, Expected: 100_000,
		},
		"double-entry blue form deducts up to 五十五万円": {
			Kind: BlueFormDoubleEntry, Income: 10_000_000, Expected: 550_000,
		},
		"double-entry blue form with e-Tax or electronic records deducts up to 六十五万円": {
			Kind: BlueFormDoubleEntryElectronic, Income: 10_000_000, Expected: 650_000,
		},
		"the deduction cannot exceed the income it is taken from (boundary)": {
			Kind: BlueFormDoubleEntryElectronic, Income: 650_000, Expected: 650_000,
		},
		"income below the ceiling caps the deduction at the income": {
			Kind: BlueFormDoubleEntryElectronic, Income: 350_000, Expected: 350_000,
		},
		"nought income deducts nothing": {
			Kind: BlueFormDoubleEntryElectronic, Income: 0, Expected: 0,
		},
		"a loss deducts nothing rather than turning more negative": {
			Kind: BlueFormDoubleEntryElectronic, Income: -50_000, Expected: 0,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := BlueFormDeduction(tc.Kind, tc.Income)
			if err != nil {
				t.Fatalf("BlueFormDeduction(%v, %d): %v", tc.Kind, tc.Income, err)
			}
			if got != tc.Expected {
				t.Errorf("BlueFormDeduction(%v, %d) = %d, want %d", tc.Kind, tc.Income, got, tc.Expected)
			}
		})
	}
}

func TestBlueFormDeductionShouldRejectAnUnrecognisedWord(t *testing.T) {
	_, err := BlueFormDeduction(BlueFormRecordKeeping("なにか別の言葉"), 10_000_000)
	if err == nil {
		t.Fatal("BlueFormDeduction: 未知の語なのにエラーにならなかった")
	}
}
