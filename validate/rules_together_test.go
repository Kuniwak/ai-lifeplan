package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestPositiveNeeds(t *testing.T) {
	const slot = "income_husband"

	rule := PositiveNeeds(slot, "西暦", []PositivePair{{
		Positive: "給与収入[円/年]", Needed: "週所定労働時間[時間/週]",
		Why: "給与を受け取る年は働いている年である",
	}})

	for _, c := range []struct {
		name     string
		rows     [][]string
		mentions [][]string
	}{
		{
			name: "働いていて給与がある",
			rows: [][]string{{"2018", "8080000", "40"}},
		},
		{
			name: "働いておらず給与も無い（引退）",
			rows: [][]string{{"2059", "0", "0"}},
		},
		{
			name: "働いているが給与が無い（休職や育休の年）",
			rows: [][]string{{"2022", "0", "40"}},
		},
		{
			name:     "給与があるのに働いていない",
			rows:     [][]string{{"2059", "10000000", "0"}},
			mentions: [][]string{{"2059", "給与収入[円/年]", "週所定労働時間[時間/週]", "給与を受け取る年は働いている年である"}},
		},
		{
			name: "年が違えば別々に報告する",
			rows: [][]string{{"2059", "10000000", "0"}, {"2060", "9000000", "0"}},
			mentions: [][]string{
				{"2059", "週所定労働時間[時間/週]"},
				{"2060", "週所定労働時間[時間/週]"},
			},
		},
		{
			name:     "労働時間が負",
			rows:     [][]string{{"2059", "10000000", "-1"}},
			mentions: [][]string{{"2059", "週所定労働時間[時間/週]"}},
		},
		{
			name:     "数でない",
			rows:     [][]string{{"2059", "10000000", "四十"}},
			mentions: [][]string{{"週所定労働時間[時間/週]"}},
		},
		{
			name: "桁区切りのある円と、無い時間",
			rows: [][]string{{"2018", "8,080,000", "40"}},
		},
		{
			name:     "桁区切りのある円で、時間が 0",
			rows:     [][]string{{"2059", "10,000,000", "0"}},
			mentions: [][]string{{"2059", "週所定労働時間[時間/週]"}},
		},
		{
			name:     "桁区切りに見えて数でない",
			rows:     [][]string{{"2059", "1,0x0", "0"}},
			mentions: [][]string{{"給与収入[円/年]"}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			tables := map[tsv.Slot]*tsv.Table{slot: {
				Header: []tsv.ColumnName{"西暦", "給与収入[円/年]", "週所定労働時間[時間/週]"},
				Rows:   c.rows,
			}}

			result := Run([]Rule{rule}, tables, RequireAll)

			assertFindings(t, messagesOf(result.Findings), c.mentions...)
		})
	}
}
