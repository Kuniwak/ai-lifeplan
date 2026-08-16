package law

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

func disabilityTable(t *testing.T) DisabilityDeductionTable {
	t.Helper()

	parsed := MustLoadDisabilityDeductions(t, os.DirFS("../"+LawDirectory))
	return parsed
}

func TestDisabilityDeduction(t *testing.T) {
	loaded := disabilityTable(t)

	got, err := loaded.Lookup(OrdinaryDisability)

	if err != nil {
		t.Fatalf("Lookup(%q): %v", OrdinaryDisability, err)
	}
	if want := (Deduction{IncomeTax: 270_000, Resident: 260_000}); got != want {
		t.Errorf("障害者控除 = %+v, want %+v", got, want)
	}
}

func TestDisabilityDeductionShouldReportAnUnsourcedCategory(t *testing.T) {
	type testCase struct {
		Category DisabilityCategoryValue
	}

	testCases := map[string]testCase{
		"見出しの打ち間違い": {Category: "障がい者"},
		"条文にない区分":   {Category: "重度障害者"},
		"空":         {Category: ""},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			loaded := disabilityTable(t)

			_, err := loaded.Lookup(tc.Category)

			if err == nil {
				t.Fatalf("Lookup(%q) succeeded; a category with no sourced amount must be reported", tc.Category)
			}
		})
	}
}

func TestCategoriesShouldBeInOrderWhateverOrderTheTableIsWrittenIn(t *testing.T) {
	written := &tsv.Table{
		Header: []tsv.ColumnName{DisabilityCategoryColumn, DisabilityIncomeTaxColumn, DisabilityResidentColumn},
		Rows: [][]string{
			{"特別障害者", "400000", "300000"},
			{"障害者", "270000", "260000"},
			{"同居特別障害者", "750000", "530000"},
		},
	}
	parsed, err := ParseDisabilityDeductionTable(written)
	if err != nil {
		t.Fatalf("ParseDisabilityDeductionTable: %v", err)
	}

	got := parsed.Categories()

	want := []DisabilityCategoryValue{"同居特別障害者", "特別障害者", "障害者"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Categories() mismatch (-want +got):\n%s", diff)
	}
}

func TestTheDisabilityHumanDeductionGapShouldMatchTheStatute(t *testing.T) {
	statutory := map[DisabilityCategoryValue]money.Yen{
		OrdinaryDisability:          10_000,
		SpecialDisability:           100_000,
		CohabitingSpecialDisability: 220_000,
	}

	loaded := disabilityTable(t)

	for category, want := range statutory {
		t.Run(string(category), func(t *testing.T) {
			got, err := loaded.Lookup(category)
			if err != nil {
				t.Fatalf("Lookup(%q): %v", category, err)
			}

			if gap := got.IncomeTax - got.Resident; gap != want {
				t.Errorf("%s: 引き算で出した差は %d、条文の差額表は %d", category, gap, want)
			}
		})
	}
}
