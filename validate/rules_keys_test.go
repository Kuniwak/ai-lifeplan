package validate_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

func TestKeysCoverYears(t *testing.T) {
	const parts, whole = "parts.tsv", "whole.tsv"

	partsTable := &tsv.Table{
		Header: []tsv.ColumnName{"西暦", "口座", "残高[円]"},
		Rows: [][]string{
			{"2024", "A銀行", "100"},
			{"2024", "B銀行", "200"},
			{"2025", "A銀行", "300"},
		},
	}

	for _, test := range []struct {
		name     string
		years    [][]string
		findings int
		says     string
	}{
		{"揃っている年だけ", [][]string{{"2024"}}, 0, ""},
		{"欠けた年がある", [][]string{{"2024"}, {"2025"}}, 1, "B銀行"},
		{"表がまったく触れない年", [][]string{{"2023"}}, 1, "2023"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rule := validate.KeysCoverYears(parts, "西暦", "口座", whole, "西暦")
			found := rule.Check(map[tsv.Slot]*tsv.Table{
				parts: partsTable,
				whole: {Header: []tsv.ColumnName{"西暦"}, Rows: test.years},
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
