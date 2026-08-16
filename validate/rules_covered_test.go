package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestYearsAreCovered(t *testing.T) {
	const cashflow, sources tsv.Slot = "cashflow", "sources"

	rule := YearsAreCovered(
		sources, "ファイル",
		cashflow, "年月",
	)

	said := [][]string{{"収入・支出詳細_2024.csv"}, {"収入・支出詳細_2025.csv"}}

	for _, c := range []struct {
		name     string
		sources  [][]string
		months   [][]string
		mentions string
	}{
		{
			name:    "書き出しの年がすべて明細にある",
			sources: said,
			months:  [][]string{{"2024-01"}, {"2024-12"}, {"2025-06"}},
		},
		{
			name:     "読んだはずの年が明細から消えている",
			sources:  said,
			months:   [][]string{{"2025-06"}},
			mentions: "2024",
		},
		{
			name:     "書き出しに無い年が明細にある",
			sources:  said,
			months:   [][]string{{"2024-01"}, {"2025-06"}, {"2019-03"}},
			mentions: "2019",
		},
		{
			name:     "年を読み取れない書き出し名",
			sources:  [][]string{{"export.csv"}},
			months:   [][]string{{"2024-01"}},
			mentions: "export.csv",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			found := rule.Check(map[tsv.Slot]*tsv.Table{
				sources:  {Header: []tsv.ColumnName{"ファイル"}, Rows: c.sources},
				cashflow: {Header: []tsv.ColumnName{"年月"}, Rows: c.months},
			})

			messages := messagesOf(found)
			if c.mentions == "" {
				assertFinding(t, messages)
				return
			}
			assertFinding(t, messages, c.mentions)
		})
	}
}
