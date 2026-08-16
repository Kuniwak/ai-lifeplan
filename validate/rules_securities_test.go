package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestHoldingsFollowTheLedger(t *testing.T) {
	const holdings, transaction = "holdings", "transaction"

	rule := HoldingsFollowTheLedger(
		HoldingSide{Slot: holdings, AsOf: "基準日", Fund: "ファンド", Pocket: "口座種別", Units: "数量[口]"},
		LedgerSide{
			Slot: transaction, Traded: "約定日", Fund: "ファンド",
			Deposit: "預り区分", Deal: "取引", Units: "数量[口]",
		},
		LedgerVocabulary{
			Pockets: map[string]string{"特定預り": "特定口座", "ＮＩＳＡ預り": "ＮＩＳＡ"},
			Bought:  "購入",
			Sold:    "解約",
		},
	)

	for _, c := range []struct {
		name  string
		rows  [][]string
		holds [][]string
		wants [][]string
	}{
		{
			name:  "買っただけ",
			rows:  [][]string{{"2024-01-05", "A", "特定預り", "購入", "100"}},
			holds: [][]string{{"2024-03-31", "A", "特定口座", "100"}},
		},
		{
			name: "買って売る",
			rows: [][]string{
				{"2024-01-05", "A", "特定預り", "購入", "100"},
				{"2024-02-05", "A", "特定預り", "解約", "30"},
			},
			holds: [][]string{{"2024-03-31", "A", "特定口座", "70"}},
		},
		{
			name: "基準日をまたいで積む",
			rows: [][]string{
				{"2024-01-05", "A", "特定預り", "購入", "100"},
				{"2024-05-05", "A", "特定預り", "購入", "50"},
			},
			holds: [][]string{
				{"2024-03-31", "A", "特定口座", "100"},
				{"2024-06-30", "A", "特定口座", "150"},
			},
		},
		{
			name: "口座種別で分かれる",
			rows: [][]string{
				{"2024-01-05", "A", "特定預り", "購入", "100"},
				{"2024-01-05", "A", "ＮＩＳＡ預り", "購入", "40"},
			},
			holds: [][]string{
				{"2024-03-31", "A", "特定口座", "100"},
				{"2024-03-31", "A", "ＮＩＳＡ", "40"},
			},
		},
		{
			name:  "台帳より残高が多い",
			rows:  [][]string{{"2024-01-05", "A", "特定預り", "購入", "100"}},
			holds: [][]string{{"2024-03-31", "A", "特定口座", "120"}},
			wants: [][]string{{"120"}},
		},
		{
			name:  "台帳に無い保有",
			rows:  [][]string{{"2024-01-05", "A", "特定預り", "購入", "100"}},
			holds: [][]string{{"2024-03-31", "B", "特定口座", "10"}},
			wants: [][]string{{"B", "報告書は 10 口"}, {"A", "台帳を積むと 100 口"}},
		},
		{
			name:  "台帳にあるのに残高に無い",
			rows:  [][]string{{"2024-01-05", "A", "特定預り", "購入", "100"}},
			holds: [][]string{{"2024-03-31", "A", "ＮＩＳＡ", "0"}},
			wants: [][]string{{"特定口座"}},
		},
		{
			name:  "基準日より後の取引を数えてはならない",
			rows:  [][]string{{"2024-04-05", "A", "特定預り", "購入", "100"}},
			holds: [][]string{{"2024-03-31", "A", "特定口座", "100"}},
			wants: [][]string{{"特定口座"}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			tables := map[tsv.Slot]*tsv.Table{
				holdings: {
					Header: []tsv.ColumnName{"基準日", "ファンド", "口座種別", "数量[口]"},
					Rows:   c.holds,
				},
				transaction: {
					Header: []tsv.ColumnName{"約定日", "ファンド", "預り区分", "取引", "数量[口]"},
					Rows:   c.rows,
				},
			}
			result := Run([]Rule{rule}, tables, RequireAll)

			assertFindings(t, messagesOf(result.Findings), c.wants...)
		})
	}
}

func TestStatementsCoverEveryQuarter(t *testing.T) {
	const holdings = "holdings"

	rule := StatementsCoverEveryQuarter(holdings, "基準日", "2024-03-31", "2024-12-31")

	for _, c := range []struct {
		name  string
		asOfs []string
		wants [][]string
	}{
		{name: "全部ある", asOfs: []string{"2024-03-31", "2024-06-30", "2024-09-30", "2024-12-31"}},
		{name: "同じ基準日が複数行あってよい", asOfs: []string{"2024-03-31", "2024-03-31", "2024-06-30", "2024-09-30", "2024-12-31"}},
		{name: "真ん中が欠ける", asOfs: []string{"2024-03-31", "2024-12-31"},
			wants: [][]string{{"2024-06-30"}, {"2024-09-30"}}},
		{name: "最後が欠ける", asOfs: []string{"2024-03-31", "2024-06-30", "2024-09-30"}, wants: [][]string{{"2024-12-31"}}},
		{name: "全部欠ける", asOfs: nil,
			wants: [][]string{{"2024-03-31"}, {"2024-06-30"}, {"2024-09-30"}, {"2024-12-31"}}},
		{name: "外の基準日がある", asOfs: []string{"2024-03-31", "2024-06-30", "2024-09-30", "2024-12-31", "2025-03-31"}, wants: [][]string{{"2025-03-31"}}},
		{name: "四半期末でない", asOfs: []string{"2024-03-31", "2024-05-31", "2024-06-30", "2024-09-30", "2024-12-31"}, wants: [][]string{{"2024-05-31"}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			rows := make([][]string, 0, len(c.asOfs))
			for _, asOf := range c.asOfs {
				rows = append(rows, []string{asOf})
			}
			tables := map[tsv.Slot]*tsv.Table{
				holdings: {Header: []tsv.ColumnName{"基準日"}, Rows: rows},
			}
			result := Run([]Rule{rule}, tables, RequireAll)

			assertFindings(t, messagesOf(result.Findings), c.wants...)
		})
	}
}

func TestHoldingsShouldRefuseAPocketNobodyNamed(t *testing.T) {
	const holdings, transaction = "holdings", "transaction"

	rule := HoldingsFollowTheLedger(
		HoldingSide{Slot: holdings, AsOf: "基準日", Fund: "ファンド", Pocket: "口座種別", Units: "数量[口]"},
		LedgerSide{
			Slot: transaction, Traded: "約定日", Fund: "ファンド",
			Deposit: "預り区分", Deal: "取引", Units: "数量[口]",
		},
		LedgerVocabulary{
			Pockets: map[string]string{"特定預り": "特定口座", "ＮＩＳＡ預り": "ＮＩＳＡ"},
			Bought:  "購入",
			Sold:    "解約",
		},
	)

	for _, c := range []struct {
		name    string
		pocket  string
		deposit string
		deal    string
		units   string
		wants   [][]string
	}{
		{name: "そのとおり", pocket: "特定口座", deposit: "特定預り", deal: "購入", units: "100"},
		{name: "口座種別の綴りが違う", pocket: "特定口座 ", deposit: "特定預り", deal: "購入", units: "0", wants: [][]string{{"特定口座 "}}},
		{name: "預り区分を知らない", pocket: "特定口座", deposit: "特定", deal: "購入", units: "100", wants: [][]string{{"特定"}}},
		{name: "取引を知らない", pocket: "特定口座", deposit: "特定預り", deal: "買付", units: "100", wants: [][]string{{"買付"}}},
		{name: "数量が数でない", pocket: "特定口座", deposit: "特定預り", deal: "購入", units: "100口", wants: [][]string{{"100口"}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			tables := map[tsv.Slot]*tsv.Table{
				holdings: {
					Header: []tsv.ColumnName{"基準日", "ファンド", "口座種別", "数量[口]"},
					Rows:   [][]string{{"2024-03-31", "A", c.pocket, c.units}},
				},
				transaction: {
					Header: []tsv.ColumnName{"約定日", "ファンド", "預り区分", "取引", "数量[口]"},
					Rows:   [][]string{{"2024-01-05", "A", c.deposit, c.deal, c.units}},
				},
			}
			result := Run([]Rule{rule}, tables, RequireAll)

			assertFindings(t, messagesOf(result.Findings), c.wants...)
		})
	}
}

func TestCentiYenShouldKeepTheSignOfAZeroWholePart(t *testing.T) {
	for _, c := range []struct {
		field string
		want  int64
	}{
		{"0.50", 50},
		{"-0.50", -50},
		{"-1.50", -150},
		{"1.50", 150},
		{"-1", -100},
	} {
		got, err := centiYen(c.field)
		if err != nil {
			t.Fatalf("centiYen(%q): %v", c.field, err)
		}
		if got != c.want {
			t.Errorf("centiYen(%q) = %d, want %d", c.field, got, c.want)
		}
	}
}
