package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestBalanceFollowsTheBank(t *testing.T) {
	const balance, bank = "balance", "bank"

	rule := BalanceFollowsTheBank(
		balance, "西暦", "貯蓄[円]",
		bank, "西暦", "残高[円]",
	)

	rows := [][]string{
		{"2025", "201000"},
		{"2025", "96000"},
		{"2025", "71000"},
		{"2025", "1019000"},
		{"2025", "4000"},
	}

	for _, c := range []struct {
		name     string
		cash     string
		bank     [][]string
		mentions []string
	}{
		{name: "そのとおり", cash: "1391000", bank: rows},
		{
			name: "合計が合わない", cash: "1392000", bank: rows,
			mentions: []string{"1392000"},
		},
		{
			name: "空欄の行がある",
			cash: "1391000",
			bank: append([][]string{{"2025", ""}}, rows...),
		},
		{
			name: "銀行の記録が無い年は何も言わない",
			cash: "999", bank: [][]string{{"2024", "1"}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			tables := map[tsv.Slot]*tsv.Table{
				balance: {
					Header: []tsv.ColumnName{"西暦", "貯蓄[円]"},
					Rows:   [][]string{{"2025", c.cash}},
				},
				bank: {
					Header: []tsv.ColumnName{"西暦", "残高[円]"},
					Rows:   c.bank,
				},
			}
			result := Run([]Rule{rule}, tables, RequireAll)

			assertFinding(t, messagesOf(result.Findings), c.mentions...)
		})
	}
}
