package validate

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

var pension = KnownKinds{"年金": {Column: "年金資産[円]", RecordedFrom: "2026/08/03"}}

func TestBalanceFollowsTheKnown(t *testing.T) {
	const balance, known = "balance", "known"

	for _, c := range []struct {
		name      string
		kinds     KnownKinds
		locked    string
		partial   string
		known     [][]string
		alsoLower []string
		mentions  []string
	}{
		{
			name:   "そのとおり",
			kinds:  pension,
			locked: "400000",
			known:  [][]string{{"2025", "年金", "400,000"}},
		},
		{
			name:     "手で消された",
			kinds:    pension,
			locked:   "0",
			known:    [][]string{{"2025", "年金", "400,000"}},
			mentions: []string{"400000"},
		},
		{
			name:   "写し元の行ごと消された",
			kinds:  pension,
			locked: "400000",
			known:  [][]string{},
			mentions: []string{
				"印が立っていなければならない",
				"400000",
			},
		},
		{
			name:    "記録が無い年は 0 と印",
			kinds:   pension,
			locked:  "0",
			partial: "はい",
			known:   [][]string{},
		},
		{
			name:     "0 と書いてある年",
			kinds:    pension,
			locked:   "1",
			known:    [][]string{{"2025", "年金", "0"}},
			mentions: []string{"年金資産[円]"},
		},
		{
			name:     "理由が無いのに印が立っている",
			kinds:    pension,
			locked:   "400000",
			partial:  "はい",
			known:    [][]string{{"2025", "年金", "400,000"}},
			mentions: []string{"印が立っていてはいけない"},
		},
		{
			name:      "外から渡された理由で印が立っている",
			kinds:     pension,
			locked:    "400000",
			partial:   "はい",
			known:     [][]string{{"2025", "年金", "400,000"}},
			alsoLower: []string{"2025"},
		},
		{
			name:     "書き出しが記録している年の行",
			kinds:    KnownKinds{"年金": {Column: "年金資産[円]", RecordedFrom: "2020/01/01"}},
			locked:   "400000",
			known:    [][]string{{"2025", "年金", "400,000"}},
			mentions: []string{"読まれない"},
		},
		{
			name:     "行き先の決まっていない種別",
			kinds:    pension,
			locked:   "400000",
			known:    [][]string{{"2025", "年金", "400,000"}, {"2025", "骨董品", "1"}},
			mentions: []string{"骨董品"},
		},
		{
			name:     "同じ年に同じ種別が 2 行ある",
			kinds:    pension,
			locked:   "800000",
			known:    [][]string{{"2025", "年金", "400,000"}, {"2025", "年金", "50,000"}},
			mentions: []string{"二度書かれており"},
		},
		{
			name:   "balance.tsv に無い年の行",
			kinds:  pension,
			locked: "0", partial: "はい",
			known:    [][]string{{"2019", "年金", "1"}},
			mentions: []string{"読まれない"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			rule := BalanceFollowsTheKnown(
				balance, "西暦", PartialMark{Column: "一部未記録", Yes: "はい"},
				known, "西暦", "資産種別", "残高[円]",
				c.kinds, LowerBoundYears{Years: func(map[tsv.Slot]*tsv.Table) []string {
					return c.alsoLower
				}},
			)

			tables := map[tsv.Slot]*tsv.Table{
				balance: {
					Header: []tsv.ColumnName{"西暦", "年金資産[円]", "一部未記録"},
					Rows:   [][]string{{"2025", c.locked, c.partial}},
				},
				known: {
					Header: []tsv.ColumnName{"西暦", "資産種別", "残高[円]"},
					Rows:   c.known,
				},
			}
			result := Run([]Rule{rule}, tables, RequireAll)

			var want [][]string
			for _, m := range c.mentions {
				want = append(want, []string{m})
			}
			assertFindings(t, messagesOf(result.Findings), want...)
		})
	}
}

func TestBalanceFollowsTheKnownShouldSayWhenAColumnMixesTwoSources(t *testing.T) {
	const balance, known = "balance", "known"

	rule := BalanceFollowsTheKnown(
		balance, "西暦", PartialMark{Column: "一部未記録", Yes: "はい"},
		known, "西暦", "資産種別", "残高[円]",
		KnownKinds{
			"預金・現金": {Column: "貯蓄[円]"},
			"ポイント":  {Column: "貯蓄[円]", RecordedFrom: "2026/08/03"},
		},
		LowerBoundYears{},
	)

	found := rule.Check(map[tsv.Slot]*tsv.Table{
		balance: {
			Header: []tsv.ColumnName{"西暦", "貯蓄[円]", "一部未記録"},
			Rows:   [][]string{{"2025", "1391000", ""}},
		},
		known: {
			Header: []tsv.ColumnName{"西暦", "資産種別", "残高[円]"},
			Rows:   [][]string{{"2025", "ポイント", "1,000"}},
		},
	})

	messages := messagesOf(found)
	assertFinding(t, messages, "混ざ")
	if len(messages) == 1 && strings.Contains(messages[0], "合計は") {
		t.Errorf("混ざった列に合計を要求している: %v", messages)
	}
}
