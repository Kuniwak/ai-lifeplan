package validate_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

func TestYearsOutsideComparison(t *testing.T) {
	const flows, balances = "cashflow.tsv", "balance.tsv"

	for _, test := range []struct {
		name     string
		months   [][]string
		years    [][]string
		expected []string
		findings int
		says     string
	}{
		{
			name:     "想定どおり外に落ちている",
			months:   [][]string{{"2021-05"}, {"2022-03"}, {"2023-07"}, {"2024-01"}, {"2025-12"}},
			years:    [][]string{{"2022"}, {"2023"}, {"2024"}},
			expected: []string{"2021", "2022", "2025"},
			findings: 0,
		},
		{
			name:     "新しい年が外に落ちた",
			months:   [][]string{{"2021-05"}, {"2022-03"}, {"2023-07"}, {"2024-01"}, {"2025-12"}, {"2026-02"}},
			years:    [][]string{{"2022"}, {"2023"}, {"2024"}},
			expected: []string{"2021", "2022", "2025"},
			findings: 1,
			says:     "2026",
		},
		{
			name:     "想定に書いた年がもう外ではない",
			months:   [][]string{{"2022-03"}, {"2023-07"}},
			years:    [][]string{{"2021"}, {"2022"}, {"2023"}},
			expected: []string{"2021", "2022"},
			findings: 1,
			says:     "2021",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			rule := validate.YearsOutsideComparison(
				flows, "年月", balances, "西暦", validate.AgainstMovement, test.expected)
			found := rule.Check(map[tsv.Slot]*tsv.Table{
				flows:    {Header: []tsv.ColumnName{"年月"}, Rows: test.months},
				balances: {Header: []tsv.ColumnName{"西暦"}, Rows: test.years},
			})
			if len(found) != test.findings {
				t.Fatalf("findings が %d 件である（%d 件のはず）: %v", len(found), test.findings, found)
			}
			if test.says != "" && !strings.Contains(found[0].Message, test.says) {
				t.Errorf("%q が %q を名指ししていない", found[0].Message, test.says)
			}
		})
	}
}

func TestYearsOutsideComparisonAgainstLevel(t *testing.T) {
	const flows, balances tsv.Slot = "bank_balance", "balance"

	tables := map[tsv.Slot]*tsv.Table{
		flows:    {Header: []tsv.ColumnName{"西暦"}, Rows: [][]string{{"2021"}, {"2022"}, {"2023"}}},
		balances: {Header: []tsv.ColumnName{"西暦"}, Rows: [][]string{{"2022"}, {"2023"}}},
	}

	level := validate.YearsOutsideComparison(
		flows, "西暦", balances, "西暦", validate.AgainstLevel, []string{"2021"})
	if found := level.Check(tables); len(found) != 0 {
		t.Errorf("水準どうしの突合で findings が出た: %v", found)
	}

	movement := validate.YearsOutsideComparison(
		flows, "西暦", balances, "西暦", validate.AgainstMovement, []string{"2021"})
	found := movement.Check(tables)
	if len(found) != 1 {
		t.Fatalf("findings が %d 件である（1 件のはず）: %v", len(found), found)
	}
	if !strings.Contains(found[0].Message, "2022") {
		t.Errorf("%q が 2022 を名指ししていない", found[0].Message)
	}
}
