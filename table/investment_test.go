package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func twoYears(t *testing.T, in table.AssetsInput, need money.Yen) relation.Table[table.AssetsRow] {
	t.Helper()

	years := []date.Year{2000, 2001}
	constant := func(v money.Yen) relation.Table[money.Yen] {
		return relation.Constant(years, v)
	}

	in.Timeline = relation.New([]relation.Row[table.TimelineRow]{
		{Year: 2000, Value: table.TimelineRow{}},
		{Year: 2001, Value: table.TimelineRow{Balance: -need}},
	})

	in.StartsAfter = 1999
	if in.ContributionMonthly.Len() == 0 {
		in.ContributionMonthly = constant(0)
	}
	if in.CashFloor.Len() == 0 {
		in.CashFloor = constant(0)
	}
	if in.NISAAllowance.Len() == 0 {
		in.NISAAllowance = constant(0)
	}
	if in.Return.Len() == 0 {
		in.Return = relation.New([]relation.Row[money.Rate]{
			{Year: 2000, Value: money.NewRate(0, 100)},
			{Year: 2001, Value: money.NewRate(100, 100)},
		})
	}

	built, err := table.AssetsTable(in)
	if err != nil {
		t.Fatalf("AssetsTable: %v", err)
	}
	return built
}

func TestSellingAHolding(t *testing.T) {
	for _, c := range []struct {
		name    string
		opening actuals.Balance
		first   bool
		wantTax bool
	}{
		{
			name:    "課税口座に値上がりぶんがある",
			opening: actuals.Balance{Invested: 10_000_000, Taxable: 10_000_000, Basis: 10_000_000},
			wantTax: true,
		},
		{
			name:    "課税口座が値上がりしていない",
			opening: actuals.Balance{Invested: 10_000_000, Taxable: 10_000_000, Basis: 10_000_000},
		},
		{
			name:    "NISA だけ持っている",
			opening: actuals.Balance{Invested: 10_000_000, NISA: 10_000_000},
		},
		{
			name:    "両方持っていて NISA から先に売る",
			opening: actuals.Balance{Invested: 10_000_000, NISA: 5_000_000, Taxable: 5_000_000, Basis: 5_000_000},
			first:   true,
		},
		{
			name:    "両方持っていて課税口座から先に売る",
			opening: actuals.Balance{Invested: 10_000_000, NISA: 5_000_000, Taxable: 5_000_000, Basis: 5_000_000},
			wantTax: true,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			in := table.AssetsInput{Opening: c.opening, SellNISAFirst: c.first}
			if c.name == "課税口座が値上がりしていない" {
				in.Return = relation.New([]relation.Row[money.Rate]{
					{Year: 2000, Value: money.NewRate(0, 100)},
					{Year: 2001, Value: money.NewRate(0, 100)},
				})
			}
			row, ok := twoYears(t, in, 5_000_000).At(2001)
			if !ok {
				t.Fatal("2001 が無い")
			}

			if c.wantTax && row.InvestmentTax == 0 {
				t.Errorf("税がかかるはずが 0 である: %+v", row)
			}
			if !c.wantTax && row.InvestmentTax != 0 {
				t.Errorf("税がかからないはずが %d である: %+v", row.InvestmentTax, row)
			}
			if got := row.Withdrawn - row.InvestmentTax; got < 5_000_000 || got > 5_000_002 {
				t.Errorf("手取り %d が必要額 5,000,000 をわずかに上回っていない（取崩 %d − 税 %d）",
					got, row.Withdrawn, row.InvestmentTax)
			}
			if row.Shortfall != 0 {
				t.Errorf("足りているのに不足 %d が出ている", row.Shortfall)
			}
			if row.NISA+row.Taxable != row.Invested {
				t.Errorf("NISA %d ＋ 課税 %d が金融資産 %d と合わない", row.NISA, row.Taxable, row.Invested)
			}
		})
	}
}

func TestContributionsShouldFillTheNISAAllowanceFirst(t *testing.T) {
	years := []date.Year{2000, 2001}
	constant := func(v money.Yen) relation.Table[money.Yen] {
		return relation.Constant(years, v)
	}

	built, err := table.AssetsTable(table.AssetsInput{
		Timeline: relation.New([]relation.Row[table.TimelineRow]{
			{Year: 2000, Value: table.TimelineRow{Balance: 12_000_000}},
			{Year: 2001, Value: table.TimelineRow{}},
		}),
		Opening:             actuals.Balance{},
		StartsAfter:         1999,
		ContributionMonthly: constant(1_000_000),
		CashFloor:           constant(0),
		NISAAllowance:       constant(5_000_000),
		Return: relation.New([]relation.Row[money.Rate]{
			{Year: 2000, Value: money.NewRate(0, 100)},
			{Year: 2001, Value: money.NewRate(0, 100)},
		}),
	})
	if err != nil {
		t.Fatalf("AssetsTable: %v", err)
	}

	row, _ := built.At(2000)
	if row.NISA != 5_000_000 {
		t.Errorf("NISA が %d である。枠の 5,000,000 まで埋まるはず", row.NISA)
	}
	if row.Taxable != 7_000_000 {
		t.Errorf("課税口座が %d である。あふれた 7,000,000 のはず", row.Taxable)
	}
	if row.Basis != 7_000_000 {
		t.Errorf("取得価額が %d である。課税口座に入れた 7,000,000 のはず", row.Basis)
	}
}
