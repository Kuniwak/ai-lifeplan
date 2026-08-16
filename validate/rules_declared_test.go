package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestKeysAreDeclared(t *testing.T) {
	const parts, declared = "bank_balance", "bank_accounts"

	rule := KeysAreDeclared(parts, "口座", declared, "口座")

	for name, c := range map[string]struct {
		parts    []string
		declared []string
		mentions [][]string
	}{
		"宣言どおり": {
			parts:    []string{"銀行B", "銀行B", "口座A"},
			declared: []string{"銀行B", "口座A"},
		},
		"宣言されているのに一度も書かれていない": {
			parts:    []string{"銀行B"},
			declared: []string{"銀行B", "口座B"},
			mentions: [][]string{{"口座B", "一度も"}},
		},
		"書かれているのに宣言されていない": {
			parts:    []string{"銀行B", "みずほ銀行"},
			declared: []string{"銀行B"},
			mentions: [][]string{{"みずほ銀行", "宣言"}},
		},
		"両方": {
			parts:    []string{"みずほ銀行"},
			declared: []string{"銀行B"},
			mentions: [][]string{{"銀行B", "一度も"}, {"みずほ銀行", "宣言"}},
		},
		"宣言が空": {
			parts:    []string{"銀行B"},
			declared: nil,
			mentions: [][]string{{"宣言", "1 つも"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			rows := func(values []string) [][]string {
				out := make([][]string, 0, len(values))
				for _, v := range values {
					out = append(out, []string{v})
				}
				return out
			}
			tables := map[tsv.Slot]*tsv.Table{
				parts:    {Header: []tsv.ColumnName{"口座"}, Rows: rows(c.parts)},
				declared: {Header: []tsv.ColumnName{"口座"}, Rows: rows(c.declared)},
			}

			result := Run([]Rule{rule}, tables, RequireAll)

			assertFindings(t, messagesOf(result.Findings), c.mentions...)
		})
	}
}
