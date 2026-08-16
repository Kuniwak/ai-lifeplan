package law

import (
	"os"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestTheYearBandedTablesShouldRefuseAGapNobodyWrote(t *testing.T) {
	type testCase struct {
		Rows  [][]string
		Says  string
		Wants bool
	}

	testCases := map[string]testCase{
		"帯が隙間なく連なっている": {
			Rows:  [][]string{{"2017", "0.30%", "2021"}, {"2022", "0.50%", "無期限"}},
			Wants: true,
		},
		"帯に穴がある": {
			Rows: [][]string{{"2017", "0.30%", "2021"}, {"2023", "0.50%", "無期限"}},
			Says: "2022",
		},
		"帯が重なっている": {
			Rows: [][]string{{"2017", "0.30%", "2023"}, {"2022", "0.50%", "無期限"}},
			Says: "2022",
		},
		"無期限の行が最後でない": {
			Rows: [][]string{{"2017", "0.30%", "無期限"}, {"2022", "0.50%", "無期限"}},
			Says: "終わらない帯は最後にしか置けない",
		},
		"最後の行が終わっている": {
			Rows: [][]string{{"2017", "0.30%", "2021"}, {"2022", "0.50%", "2025"}},
			Says: "最後",
		},
		"1 行しかなく、それが終わっている": {
			Rows: [][]string{{"不明", "0.30%", "2021"}},
			Says: "最後",
		},
		"不明から始まる": {
			Rows:  [][]string{{"不明", "0.30%", "2021"}, {"2022", "0.50%", "無期限"}},
			Wants: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			table := &tsv.Table{
				Header: []tsv.ColumnName{LawStartYearColumn, EmploymentInsuranceRateColumn, LawEndYearColumn},
				Rows:   tc.Rows,
			}

			_, err := ParseYearRateTable(table, EmploymentInsuranceRateTableName, EmploymentInsuranceRateColumn)

			switch {
			case tc.Wants && err != nil:
				t.Fatalf("読めるはずが %v", err)
			case !tc.Wants && err == nil:
				t.Fatal("断るはずが読めてしまった")
			case !tc.Wants && !strings.Contains(err.Error(), tc.Says):
				t.Errorf("言われたのは %q だが、%q を含んでほしい", err, tc.Says)
			}
		})
	}
}

func TestTheYearBandedTablesInDataLawShouldHaveNoGaps(t *testing.T) {
	root := os.DirFS("../" + LawDirectory)

	MustLoadEmploymentInsuranceRates(t, root)
	MustLoadNationalPensionPremiums(t, root)
	MustLoadSocialInsuranceRates(t, root)
	MustLoadSpouseIncomeCeilings(t, root)
	MustLoadChildcareLeaveBenefits(t, root)

	if _, err := LoadForestEnvironmentTaxTable(root); err != nil {
		t.Errorf("law.LoadForestEnvironmentTaxTable: %v", err)
	}
}
