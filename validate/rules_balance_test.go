package validate

import (
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestBalanceFollowsTheStatements(t *testing.T) {
	const balance, holdings, outside = "balance", "holdings", "outside"

	rule := BalanceFollowsTheStatements(
		SplitBalanceSide{
			Slot: balance, Year: "西暦", NISA: "金融資産(NISA)[円]",
			Taxable: "金融資産(課税)[円]", Basis: "金融資産(課税)の取得価額[円]",
		},
		StatementSide{
			Slot: holdings, AsOf: "基準日", Pocket: "口座種別",
			Value: "時価評価額[円]", Gain: "評価損益額[円]",
		},
		OutsideSide{Slot: outside, Year: "西暦", Amount: "報告書の外の金融資産[円]", Pocket: "口座種別"},
		BalancePockets{NISA: "ＮＩＳＡ", Taxable: "特定口座"},
	)

	held := [][]string{
		{"2025-12-31", "ＮＩＳＡ", "3000000", "500000"},
		{"2025-12-31", "ＮＩＳＡ", "2000000", "300000"},
		{"2025-12-31", "特定口座", "8000000", "1500000"},
	}

	for _, c := range []struct {
		name                 string
		nisa, taxable, basis string
		holds                [][]string
		wants                [][]string
	}{
		{
			name: "そのとおり", nisa: "5000000", taxable: "8400000", basis: "6900000", holds: held,
		},
		{
			name: "報告書に無い年は何も言わない",
			nisa: "999", taxable: "999", basis: "999",
			holds: [][]string{{"2024-12-31", "ＮＩＳＡ", "1", "0"}},
		},
		{
			name: "NISA が合わない", nisa: "5001000", taxable: "8400000", basis: "6900000",
			holds: held, wants: [][]string{{"5001000"}},
		},
		{
			name: "課税が報告書と外の分の合計に合わない", nisa: "5000000", taxable: "7999000", basis: "6900000",
			holds: held, wants: [][]string{{"7999000"}},
		},
		{
			name: "報告書の外の分に含み益を持たせている",
			nisa: "5000000", taxable: "8400000", basis: "6500000",
			holds: held, wants: [][]string{{"400000"}},
		},
		{
			name: "含み損で取得価額が時価を超える",
			nisa: "5000000", taxable: "8400000", basis: "8500000",
			holds: [][]string{
				{"2025-12-31", "ＮＩＳＡ", "3000000", "500000"},
				{"2025-12-31", "ＮＩＳＡ", "2000000", "300000"},
				{"2025-12-31", "特定口座", "8000000", "-100000"},
			},
			wants: [][]string{{"超えている"}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			tables := map[tsv.Slot]*tsv.Table{
				balance: {
					Header: []tsv.ColumnName{"西暦", "金融資産(NISA)[円]", "金融資産(課税)[円]", "金融資産(課税)の取得価額[円]"},
					Rows:   [][]string{{"2025", c.nisa, c.taxable, c.basis}},
				},
				holdings: {
					Header: []tsv.ColumnName{"基準日", "口座種別", "時価評価額[円]", "評価損益額[円]"},
					Rows:   c.holds,
				},
				outside: {
					Header: []tsv.ColumnName{"西暦", "報告書の外の金融資産[円]", "口座種別"},
					Rows:   [][]string{{"2025", "400000", "特定口座"}, {"2025", "0", "ＮＩＳＡ"}},
				},
			}
			result := Run([]Rule{rule}, tables, RequireAll)

			assertFindings(t, messagesOf(result.Findings), c.wants...)
		})
	}
}

func TestBalanceShouldPinWhatTheStatementsDoNotCover(t *testing.T) {
	const balance, holdings, outside = "balance", "holdings", "outside"

	rule := BalanceFollowsTheStatements(
		SplitBalanceSide{
			Slot: balance, Year: "西暦", NISA: "金融資産(NISA)[円]",
			Taxable: "金融資産(課税)[円]", Basis: "金融資産(課税)の取得価額[円]",
		},
		StatementSide{
			Slot: holdings, AsOf: "基準日", Pocket: "口座種別",
			Value: "時価評価額[円]", Gain: "評価損益額[円]",
		},
		OutsideSide{Slot: outside, Year: "西暦", Amount: "報告書の外の金融資産[円]", Pocket: "口座種別"},
		BalancePockets{NISA: "ＮＩＳＡ", Taxable: "特定口座"},
	)

	held := [][]string{
		{"2025-12-31", "ＮＩＳＡ", "3000000", "500000"},
		{"2025-12-31", "ＮＩＳＡ", "2000000", "300000"},
		{"2025-12-31", "特定口座", "8000000", "1500000"},
	}

	for _, c := range []struct {
		name                 string
		nisa, taxable, basis string
		rest                 [][]string
		wants                [][]string
	}{
		{
			name: "そのとおり", nisa: "5000000", taxable: "8400000", basis: "6900000",
			rest: [][]string{{"2025", "400000", "特定口座"}, {"2025", "0", "ＮＩＳＡ"}},
		},
		{
			name: "課税と取得価額を一緒に水増しする",
			nisa: "5000000", taxable: "9400000", basis: "7900000",
			rest:  [][]string{{"2025", "400000", "特定口座"}, {"2025", "0", "ＮＩＳＡ"}},
			wants: [][]string{{"金融資産(課税)[円]", "9400000"}, {"取得価額", "7900000"}},
		},
		{
			name: "報告書の外の分を書き落とす",
			nisa: "5000000", taxable: "8400000", basis: "6900000",
			rest:  [][]string{{"2024", "400000", "特定口座"}, {"2024", "0", "ＮＩＳＡ"}},
			wants: [][]string{{"2025", "ＮＩＳＡ", "行が無い"}, {"2025", "特定口座", "行が無い"}},
		},
		{
			name: "報告書の外の分が食い違う",
			nisa: "5000000", taxable: "8400000", basis: "6900000",
			rest:  [][]string{{"2025", "399000", "特定口座"}, {"2025", "0", "ＮＩＳＡ"}},
			wants: [][]string{{"金融資産(課税)[円]", "8399000"}, {"取得価額", "6899000"}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			tables := map[tsv.Slot]*tsv.Table{
				balance: {
					Header: []tsv.ColumnName{"西暦", "金融資産(NISA)[円]", "金融資産(課税)[円]", "金融資産(課税)の取得価額[円]"},
					Rows:   [][]string{{"2025", c.nisa, c.taxable, c.basis}},
				},
				holdings: {
					Header: []tsv.ColumnName{"基準日", "口座種別", "時価評価額[円]", "評価損益額[円]"},
					Rows:   held,
				},
				outside: {
					Header: []tsv.ColumnName{"西暦", "報告書の外の金融資産[円]", "口座種別"},
					Rows:   c.rest,
				},
			}
			result := Run([]Rule{rule}, tables, RequireAll)

			assertFindings(t, messagesOf(result.Findings), c.wants...)
		})
	}
}

func TestBalanceShouldPutOutsideNISAInTheTaxFreePot(t *testing.T) {
	const balance, holdings, outside = "balance", "holdings", "outside"

	rule := BalanceFollowsTheStatements(
		SplitBalanceSide{
			Slot: balance, Year: "西暦", NISA: "金融資産(NISA)[円]",
			Taxable: "金融資産(課税)[円]", Basis: "金融資産(課税)の取得価額[円]",
		},
		StatementSide{
			Slot: holdings, AsOf: "基準日", Pocket: "口座種別",
			Value: "時価評価額[円]", Gain: "評価損益額[円]",
		},
		OutsideSide{Slot: outside, Year: "西暦", Amount: "報告書の外の金融資産[円]", Pocket: "口座種別"},
		BalancePockets{NISA: "ＮＩＳＡ", Taxable: "特定口座"},
	)

	holds := [][]string{
		{"2025-12-31", "ＮＩＳＡ", "5000000", "800000"},
		{"2025-12-31", "特定口座", "8000000", "1500000"},
	}
	rest := [][]string{
		{"2025", "200000", "ＮＩＳＡ"},
		{"2025", "100000", "特定口座"},
	}

	for _, c := range []struct {
		name                 string
		nisa, taxable, basis string
		wants                [][]string
	}{
		{
			name: "外の NISA は NISA 側に足される",
			nisa: "5200000", taxable: "8100000", basis: "6600000",
		},
		{
			name: "外の NISA を課税側に入れると断られる",
			nisa: "5000000", taxable: "8299000", basis: "6799000",
			wants: [][]string{{"金融資産(NISA)[円]", "5200000"}, {"金融資産(課税)[円]", "8100000"}, {"取得価額", "6600000"}},
		},
		{
			name: "外の NISA を取得価額に足すと断られる",
			nisa: "5200000", taxable: "8100000", basis: "6800000",
			wants: [][]string{{"6600000"}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			tables := map[tsv.Slot]*tsv.Table{
				balance: {
					Header: []tsv.ColumnName{"西暦", "金融資産(NISA)[円]", "金融資産(課税)[円]", "金融資産(課税)の取得価額[円]"},
					Rows:   [][]string{{"2025", c.nisa, c.taxable, c.basis}},
				},
				holdings: {
					Header: []tsv.ColumnName{"基準日", "口座種別", "時価評価額[円]", "評価損益額[円]"},
					Rows:   holds,
				},
				outside: {
					Header: []tsv.ColumnName{"西暦", "報告書の外の金融資産[円]", "口座種別"},
					Rows:   rest,
				},
			}
			result := Run([]Rule{rule}, tables, RequireAll)

			assertFindings(t, messagesOf(result.Findings), c.wants...)
		})
	}
}

func TestBalanceShouldRequireARowForEveryPot(t *testing.T) {
	const balance, holdings, outside = "balance", "holdings", "outside"

	rule := BalanceFollowsTheStatements(
		SplitBalanceSide{
			Slot: balance, Year: "西暦", NISA: "金融資産(NISA)[円]",
			Taxable: "金融資産(課税)[円]", Basis: "金融資産(課税)の取得価額[円]",
		},
		StatementSide{
			Slot: holdings, AsOf: "基準日", Pocket: "口座種別",
			Value: "時価評価額[円]", Gain: "評価損益額[円]",
		},
		OutsideSide{Slot: outside, Year: "西暦", Amount: "報告書の外の金融資産[円]", Pocket: "口座種別"},
		BalancePockets{NISA: "ＮＩＳＡ", Taxable: "特定口座"},
	)

	holds := [][]string{
		{"2025-12-31", "ＮＩＳＡ", "5000000", "800000"},
		{"2025-12-31", "特定口座", "8000000", "1500000"},
	}

	for _, c := range []struct {
		name                 string
		nisa, taxable, basis string
		rest                 [][]string
		wants                [][]string
	}{
		{
			name: "両方の枠が書いてある",
			nisa: "5200000", taxable: "8100000", basis: "6600000",
			rest: [][]string{{"2025", "200000", "ＮＩＳＡ"}, {"2025", "100000", "特定口座"}},
		},
		{
			name: "外に何も無い年も、無いと書いてあれば通る",
			nisa: "5000000", taxable: "8000000", basis: "6500000",
			rest: [][]string{{"2025", "0", "ＮＩＳＡ"}, {"2025", "0", "特定口座"}},
		},
		{
			name: "NISA の行が無い",
			nisa: "5000000", taxable: "8100000", basis: "6600000",
			rest:  [][]string{{"2025", "100000", "特定口座"}},
			wants: [][]string{{"ＮＩＳＡ"}},
		},
		{
			name: "課税の行が無い",
			nisa: "5200000", taxable: "8000000", basis: "6500000",
			rest:  [][]string{{"2025", "200000", "ＮＩＳＡ"}},
			wants: [][]string{{"特定口座"}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			tables := map[tsv.Slot]*tsv.Table{
				balance: {
					Header: []tsv.ColumnName{"西暦", "金融資産(NISA)[円]", "金融資産(課税)[円]", "金融資産(課税)の取得価額[円]"},
					Rows:   [][]string{{"2025", c.nisa, c.taxable, c.basis}},
				},
				holdings: {
					Header: []tsv.ColumnName{"基準日", "口座種別", "時価評価額[円]", "評価損益額[円]"},
					Rows:   holds,
				},
				outside: {
					Header: []tsv.ColumnName{"西暦", "報告書の外の金融資産[円]", "口座種別"},
					Rows:   c.rest,
				},
			}
			result := Run([]Rule{rule}, tables, RequireAll)

			assertFindings(t, messagesOf(result.Findings), c.wants...)
		})
	}
}
