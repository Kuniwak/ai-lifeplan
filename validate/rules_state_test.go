package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestAsOneOf(t *testing.T) {
	parse := AsOneOf("残高", "未開設", "未取得")

	for _, c := range []struct {
		field   string
		refused bool
	}{
		{field: "残高"},
		{field: "未開設"},
		{field: "未取得"},
		{field: "", refused: true},
		{field: "みかいせつ", refused: true},
		{field: "残高 ", refused: true},
	} {
		t.Run(c.field, func(t *testing.T) {
			err := parse(c.field)
			if c.refused && err == nil {
				t.Errorf("%q が通った", c.field)
			}
			if !c.refused && err != nil {
				t.Errorf("%q が拒まれた: %v", c.field, err)
			}
		})
	}
}

func TestAmountAgreesWithItsState(t *testing.T) {
	const slot tsv.Slot = "bank"

	for _, c := range []struct {
		name     string
		rows     [][]string
		mentions string
	}{
		{
			name: "残高が書いてある",
			rows: [][]string{{"2025", "口座B", "4,089", "残高"}},
		},
		{
			name: "未開設なので空",
			rows: [][]string{{"2023", "銀行B", "", "未開設"}},
		},
		{
			name:     "帳票がまだ取れていない",
			rows:     [][]string{{"2025", "口座B", "", "未取得"}},
			mentions: "未取得",
		},
		{
			name:     "未開設なのに額がある",
			rows:     [][]string{{"2023", "銀行B", "96,000", "未開設"}},
			mentions: "どちらかが違う",
		},
		{
			name:     "残高と言いながら空",
			rows:     [][]string{{"2025", "口座B", "", "残高"}},
			mentions: "空欄は合計に 0 として入る",
		},
		{
			name:     "語彙の外の語",
			rows:     [][]string{{"2025", "口座B", "", "未開設 "}},
			mentions: "語彙",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			rule := AmountAgreesWithItsState(slot, "残高[円]", "状態", AmountStates{
				Written: "残高", Absent: []string{"未開設"}, Unfetched: []string{"未取得"},
			})

			found := rule.Check(map[tsv.Slot]*tsv.Table{
				slot: {
					Header: []tsv.ColumnName{"西暦", "口座", "残高[円]", "状態"},
					Rows:   c.rows,
				},
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

func TestStateOnlyAtTheStart(t *testing.T) {
	const slot tsv.Slot = "bank"

	rule := StateOnlyAtTheStart(slot, "口座", "西暦", "状態", "未開設")

	for _, c := range []struct {
		name     string
		rows     [][]string
		mentions string
	}{
		{
			name: "未開設 が先にある",
			rows: [][]string{
				{"2021", "銀行B", "未開設"},
				{"2022", "銀行B", "未開設"},
				{"2024", "銀行B", "残高"},
			},
		},
		{
			name: "口座ごとに独立している",
			rows: [][]string{
				{"2021", "銀行B", "未開設"},
				{"2021", "口座B", "残高"},
				{"2024", "銀行B", "残高"},
			},
		},
		{
			name: "残高のあとに 未開設 が来る",
			rows: [][]string{
				{"2024", "口座B", "残高"},
				{"2025", "口座B", "未開設"},
			},
			mentions: "2024",
		},
		{
			name: "行の並び順に依らない",
			rows: [][]string{
				{"2025", "口座B", "未開設"},
				{"2024", "口座B", "残高"},
			},
			mentions: "2024",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			found := rule.Check(map[tsv.Slot]*tsv.Table{
				slot: {
					Header: []tsv.ColumnName{"西暦", "口座", "状態"},
					Rows:   c.rows,
				},
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

func TestStateOnlyAtTheEnd(t *testing.T) {
	const slot tsv.Slot = "bank"

	rule := StateOnlyAtTheEnd(slot, "口座", "西暦", "状態", "解約")

	for _, c := range []struct {
		name string
		rows [][]string

		says string
	}{
		{
			name: "解約 が最後にある",
			rows: [][]string{
				{"2024", "銀行B", "残高"},
				{"2025", "銀行B", "解約"},
				{"2026", "銀行B", "解約"},
			},
		},
		{
			name: "口座ごとに独立している",
			rows: [][]string{
				{"2025", "銀行B", "解約"},
				{"2025", "口座B", "残高"},
				{"2026", "口座B", "残高"},
			},
		},
		{
			name: "解約のあとに 残高 が来る",
			rows: [][]string{
				{"2025", "口座B", "解約"},
				{"2026", "口座B", "残高"},
			},
			says: `2026 の 口座B が "解約" ではない。2025 に "解約" になったので、戻ることはない`,
		},
		{
			name: "行の並び順に依らない",
			rows: [][]string{
				{"2026", "口座B", "残高"},
				{"2025", "口座B", "解約"},
			},
			says: `2026 の 口座B が "解約" ではない。2025 に "解約" になったので、戻ることはない`,
		},
		{
			name: "解約が一度も無い",
			rows: [][]string{
				{"2024", "口座B", "残高"},
				{"2025", "口座B", "残高"},
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			tables := map[tsv.Slot]*tsv.Table{slot: {
				Header: []tsv.ColumnName{"西暦", "口座", "状態"},
				Rows:   c.rows,
			}}

			found := Run([]Rule{rule}, tables, RequireAll).Findings

			if c.says == "" {
				assertFinding(t, messagesOf(found))
				return
			}
			assertFinding(t, messagesOf(found), c.says)
		})
	}
}
