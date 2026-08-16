package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestEveryChoiceMade(t *testing.T) {
	const slot = "inflation_target"
	rule := EveryChoiceMade(slot, "費目", "適用",
		[]string{"家賃", "ローン返済"}, OneOf("はい", "いいえ"), "いいえ")

	for _, c := range []struct {
		name  string
		rows  [][]string
		wants []string
	}{
		{
			name: "すべての費目にちょうど一度ずつ答えてある",
			rows: [][]string{{"家賃", "はい"}, {"ローン返済", "いいえ"}},
		},
		{
			name:  "誰も答えていない費目がある",
			rows:  [][]string{{"家賃", "はい"}},
			wants: []string{"ローン返済"},
		},
		{
			name:  "同じ費目に二度答えている",
			rows:  [][]string{{"家賃", "はい"}, {"家賃", "いいえ"}, {"ローン返済", "いいえ"}},
			wants: []string{"家賃"},
		},
		{
			name:  "答えが はい でも いいえ でもない",
			rows:  [][]string{{"家賃", ""}, {"ローン返済", "たぶん"}},
			wants: []string{"家賃", "たぶん"},
		},
		{
			name:  "計画に場所の無い費目が書かれている",
			rows:  [][]string{{"家賃", "はい"}, {"ローン返済", "いいえ"}, {"食費", "はい"}},
			wants: []string{"食費"},
		},
		{
			name:  "二度答えたうえに片方が読めない",
			rows:  [][]string{{"家賃", "はい"}, {"家賃", "たぶん"}, {"ローン返済", "いいえ"}},
			wants: []string{"二度書かれており", "たぶん"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			messages := check(t, rule, slot, &tsv.Table{Header: []tsv.ColumnName{"費目", "適用"}, Rows: c.rows})

			want := make([][]string, 0, len(c.wants))
			for _, w := range c.wants {
				want = append(want, []string{w})
			}
			assertFindings(t, messages, want...)
		})
	}
}
