package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestLoanSettlementInTerm(t *testing.T) {
	const loan, settlement = "loan", "loan_settlement"

	rule := LoanSettlementInTerm(loan, "初回返済年", "初回返済月", "借入期間[年]", settlement, "一括返済年", "しない")

	for _, c := range []struct {
		name     string
		settled  string
		mentions []string
	}{
		{name: "期間のなか", settled: "2042"},
		{name: "初回の年と同じ", settled: "2023"},
		{name: "完済の年", settled: "2057"},
		{name: "しない", settled: "しない"},
		{name: "初回より前", settled: "2022", mentions: []string{"2022"}},
		{name: "完済の翌年", settled: "2058", mentions: []string{"2058"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			tables := map[tsv.Slot]*tsv.Table{
				loan: {
					Header: []tsv.ColumnName{"初回返済年", "初回返済月", "借入期間[年]"},
					Rows:   [][]string{{"2023", "1", "35"}},
				},
				settlement: {
					Header: []tsv.ColumnName{"一括返済年"},
					Rows:   [][]string{{c.settled}},
				},
			}
			result := Run([]Rule{rule}, tables, RequireAll)

			assertFinding(t, messagesOf(result.Findings), c.mentions...)
		})
	}
}

func TestWholeNumberShouldRefuseAnythingButTheNumber(t *testing.T) {
	for _, c := range []struct {
		field string
		want  int64
		ok    bool
	}{
		{field: "24000", want: 24000, ok: true},
		{field: "-100000", want: -100000, ok: true},
		{field: "0", want: 0, ok: true},
		{field: "5622000", want: 5622000, ok: true},
		{field: "24000口"},
		{field: "24000.7"},
		{field: "1,234"},
		{field: " 24000"},
		{field: ""},
		{field: "口"},
	} {
		t.Run(c.field, func(t *testing.T) {
			got, err := wholeNumber(c.field)
			if c.ok {
				if err != nil {
					t.Fatalf("%q を拒んだ: %v", c.field, err)
				}
				if got != c.want {
					t.Errorf("%q が %d になった。%d のはず", c.field, got, c.want)
				}
				return
			}
			if err == nil {
				t.Errorf("%q を %d として受け入れた", c.field, got)
			}
		})
	}
}
