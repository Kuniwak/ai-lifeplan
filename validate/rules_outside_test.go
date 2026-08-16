package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestOutsideFollowsTheHoldings(t *testing.T) {
	const outside, holdings = "outside", "holdings"

	rule := OutsideFollowsTheHoldings(
		outside, "西暦", "報告書の外の金融資産[円]", "口座種別",
		holdings, "基準日", "口座種別", "時価評価額[円]",
	)

	held := [][]string{
		{"2024-09-30", "ＮＩＳＡ", "100000"},
		{"2024-12-30", "ＮＩＳＡ", "200000"},
		{"2025-09-30", "ＮＩＳＡ", "250000"},
		{"2025-12-30", "ＮＩＳＡ", "300000"},
	}

	for _, c := range []struct {
		name     string
		rest     [][]string
		holds    [][]string
		mentions []string
	}{
		{
			name: "写しが合っている",
			rest: [][]string{
				{"2024", "200000", "ＮＩＳＡ"}, {"2024", "0", "特定口座"},
				{"2025", "300000", "ＮＩＳＡ"}, {"2025", "100000", "特定口座"},
			},
			holds: held,
		},
		{
			name: "MoneyForward の 12/26 の値を写している",
			rest: [][]string{
				{"2024", "200000", "ＮＩＳＡ"}, {"2024", "0", "特定口座"},
				{"2025", "301000", "ＮＩＳＡ"}, {"2025", "100000", "特定口座"},
			},
			holds: held, mentions: []string{"301000"},
		},
		{
			name: "持ち高の表が無い枠は何も言わない",
			rest: [][]string{
				{"2025", "300000", "ＮＩＳＡ"}, {"2025", "100000", "特定口座"},
			},
			holds: [][]string{{"2025-12-30", "ＮＩＳＡ", "300000"}},
		},
		{
			name: "その年の最後の基準日が年末でない",
			rest: [][]string{{"2026", "1", "ＮＩＳＡ"}},
			holds: [][]string{
				{"2026-03-31", "ＮＩＳＡ", "1"},
			},
			mentions: []string{"2026-03-31"},
		},
		{
			name: "持ち高はあるのに書き留めた行が無い",
			rest: [][]string{{"2025", "100000", "特定口座"}},
			holds: [][]string{
				{"2025-12-30", "ＮＩＳＡ", "300000"},
			},
			mentions: []string{"ＮＩＳＡ"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			tables := map[tsv.Slot]*tsv.Table{
				outside: {
					Header: []tsv.ColumnName{"西暦", "報告書の外の金融資産[円]", "口座種別"},
					Rows:   c.rest,
				},
				holdings: {
					Header: []tsv.ColumnName{"基準日", "口座種別", "時価評価額[円]"},
					Rows:   c.holds,
				},
			}
			result := Run([]Rule{rule}, tables, RequireAll)

			assertFinding(t, messagesOf(result.Findings), c.mentions...)
		})
	}
}
