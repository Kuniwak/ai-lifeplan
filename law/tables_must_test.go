package law

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

type refusal struct{ said string }

func (r *refusal) Helper() {}

func (r *refusal) Fatalf(format string, args ...any) {
	r.said = fmt.Sprintf(format, args...)
}

func TestTheMustLoadersShouldRefuseTheWrongKindOfTable(t *testing.T) {
	type testCase struct {
		Load func(testingT)
		Says string
	}

	root := os.DirFS("../" + LawDirectory)
	testCases := map[string]testCase{
		"regional table asked for as a national one": {
			Load: func(fake testingT) { MustLoadTable(fake, root, ResidentRateTableName) },
			Says: "MustLoadRegionalTable",
		},
		"national table asked for as a regional one": {
			Load: func(fake testingT) {
				MustLoadRegionalTable(fake, root, "世田谷区", NationalPensionPremiumTableName)
			},
			Says: "MustLoadTable",
		},
		"a table Shapes does not register": {
			Load: func(fake testingT) { MustLoadTable(fake, root, "national/no-such-table") },
			Says: "law.Shapes()",
		},
		"no municipality named": {
			Load: func(fake testingT) { MustLoadRegionalTable(fake, root, "", KokuhoTableName) },
			Says: "自治体が空",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			fake := &refusal{}

			tc.Load(fake)

			if !strings.Contains(fake.said, tc.Says) {
				t.Errorf("言われたのは %q だが、%q を含んでほしい", fake.said, tc.Says)
			}
		})
	}
}

func TestMustLoadRegionalTableShouldReachAMunicipalitysCopy(t *testing.T) {
	table := MustLoadRegionalTable(t, os.DirFS("../"+LawDirectory), "世田谷区", KokuhoTableName)

	if len(table.Rows) == 0 {
		t.Fatal("世田谷区の国保の表に行が無い")
	}
	if _, ok := table.ColumnIndex(KokuhoPartColumn); !ok {
		t.Errorf("読めた表に %s の列が無い。別の表を読んでいる", KokuhoPartColumn)
	}
}
