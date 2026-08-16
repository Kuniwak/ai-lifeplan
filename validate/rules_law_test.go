package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/panictest"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestLawRangeTotal(t *testing.T) {
	type testCase struct {
		Table *tsv.Table
		Wants [][]string
	}

	header := []tsv.ColumnName{"下限"}

	testCases := map[string]testCase{
		"the lowest band starts at the bottom (boundary value)": {
			Table: &tsv.Table{Header: header, Rows: [][]string{{"0"}, {"1,950,000"}, {"3,300,000"}}},
		},
		"the rows need not be in order": {
			Table: &tsv.Table{Header: header, Rows: [][]string{{"3300000"}, {"0"}, {"1950000"}}},
		},
		"a single band covering everything": {
			Table: &tsv.Table{Header: header, Rows: [][]string{{"0"}}},
		},
		"the bottom of the domain is uncovered": {
			Table: &tsv.Table{Header: header, Rows: [][]string{{"1950000"}, {"3300000"}}},
			Wants: [][]string{{"1950000", "uncovered"}},
		},
		"two bands start at the same bound": {
			Table: &tsv.Table{Header: header, Rows: [][]string{{"0"}, {"1950000"}, {"1950000"}}},
			Wants: [][]string{{"1950000", "two bands"}},
		},
		"no bands at all": {
			Table: &tsv.Table{Header: header},
			Wants: [][]string{{"no bands"}},
		},
		"a bound that is not a number": {
			Table: &tsv.Table{Header: header, Rows: [][]string{{"195万"}}},
			Wants: [][]string{{"195万"}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := check(t, LawRangeTotal("income_tax", "下限", 0), "income_tax", tc.Table)

			assertFindings(t, got, tc.Wants...)
		})
	}
}

func TestLawRangeTotalShouldGuaranteeLookupCannotMiss(t *testing.T) {
	table := &tsv.Table{Header: []tsv.ColumnName{"下限"}, Rows: [][]string{{"0"}, {"1950000"}, {"3300000"}}}
	if got := check(t, LawRangeTotal("income_tax", "下限", 0), "income_tax", table); len(got) != 0 {
		t.Fatalf("the table under test does not pass the rule: %v", got)
	}

	bands := relation.NewBands([]relation.Band[int64, string]{
		{Lower: 0, Value: "5%"},
		{Lower: 1_950_000, Value: "10%"},
		{Lower: 3_300_000, Value: "20%"},
	})

	for _, key := range []int64{0, 1, 1_949_999, 1_950_000, 3_300_000, 900_000_000} {
		if r := panictest.Recovered(func() { bands.Lookup(key) }); r != nil {
			t.Errorf("Lookup(%d) missed a table that passed law-range-total: %v", key, r)
		}
	}
}

func TestLawSource(t *testing.T) {
	type testCase struct {
		Rows  [][]string
		Wants [][]string
	}

	testCases := map[string]testCase{
		"every row cites something": {
			Rows: [][]string{{"8%", "https://www.city.setagaya.lg.jp/..."}},
		},
		"a row with no source": {
			Rows:  [][]string{{"8%", "https://example.invalid/a"}, {"2.025%", ""}},
			Wants: [][]string{{"row 2", "no source"}},
		},
		"a source of only spaces": {
			Rows:  [][]string{{"8%", "   "}},
			Wants: [][]string{{"row 1"}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			table := &tsv.Table{Header: []tsv.ColumnName{"料率", "出典"}, Rows: tc.Rows}

			got := check(t, LawSource("law", "出典"), "law", table)

			assertFindings(t, got, tc.Wants...)
		})
	}
}

func TestLawValidity(t *testing.T) {
	type testCase struct {
		Rows  [][]string
		Wants [][]string
	}

	testCases := map[string]testCase{
		"an announced end": {
			Rows: [][]string{{"2013", "2037"}},
		},
		"no announced end is written out": {
			Rows: [][]string{{"2013", string(Indefinite)}},
		},
		"starting and ending in the same year (boundary value)": {
			Rows: [][]string{{"2022", "2022"}},
		},
		"a start nobody has looked up is written out": {
			Rows: [][]string{{string(Unknown), string(Indefinite)}},
		},
		"an unknown start still needs an end": {
			Rows:  [][]string{{string(Unknown), ""}},
			Wants: [][]string{{"row 1", string(Indefinite)}},
		},
		"a blank end year": {
			Rows:  [][]string{{"2013", ""}},
			Wants: [][]string{{"row 1", string(Indefinite)}},
		},
		"a blank start year": {
			Rows:  [][]string{{"", string(Indefinite)}},
			Wants: [][]string{{"row 1", string(Unknown)}},
		},
		"an end before the start": {
			Rows:  [][]string{{"2037", "2013"}},
			Wants: [][]string{{"2013", "before"}},
		},
		"a start that is not a year": {
			Rows:  [][]string{{"平成25年", "2037"}},
			Wants: [][]string{{"平成25年"}},
		},
		"an end that is neither a year nor the word": {
			Rows:  [][]string{{"2013", "ずっと"}},
			Wants: [][]string{{"ずっと"}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			table := &tsv.Table{Header: []tsv.ColumnName{"開始年", "終了年"}, Rows: tc.Rows}

			got := check(t, LawValidity("law", "開始年", "終了年"), "law", table)

			assertFindings(t, got, tc.Wants...)
		})
	}
}

func TestMunicipalitySupported(t *testing.T) {
	type testCase struct {
		Rows      [][]string
		Supported []string
		Wants     [][]string
	}

	testCases := map[string]testCase{
		"every municipality has tables": {
			Rows:      [][]string{{"2018", "世田谷区"}, {"2023", "世田谷区"}},
			Supported: []string{"世田谷区"},
		},
		"a municipality nobody wrote up": {
			Rows:      [][]string{{"2018", "世田谷区"}, {"2094", "札幌市"}},
			Supported: []string{"世田谷区"},
			Wants:     [][]string{{"札幌市", "data/law/"}},
		},
		"an empty municipality is left to column-schema": {
			Rows:      [][]string{{"2018", ""}},
			Supported: []string{"世田谷区"},
		},
		"the same municipality on many rows is reported once": {
			Rows:      [][]string{{"2018", "札幌市"}, {"2019", "札幌市"}},
			Supported: []string{"世田谷区"},
			Wants:     [][]string{{"札幌市"}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			table := &tsv.Table{Header: []tsv.ColumnName{"西暦", "住所"}, Rows: tc.Rows}

			got := check(t, MunicipalitySupported("household", "住所", MunicipalityGate{
				Rule:      MunicipalityRule,
				What:      "住む自治体",
				Supported: tc.Supported,
				Missing:   func(name string) []string { return []string{"data/law/" + name + "/a.tsv"} },
			}), "household", table)

			assertFindings(t, got, tc.Wants...)
		})
	}
}
