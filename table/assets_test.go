package table_test

import (
	"os"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

func assetsOfTheBaseProject(t *testing.T) relation.Table[table.AssetsRow] {
	t.Helper()

	tables := tablesOfTheBaseProject(t)
	calendar := calendarOfTheBaseProject(t)
	years := calendar.Years()
	from, to := years[0], years[len(years)-1]

	contribution, err := table.ReadYenStep(tables[input.InvestmentSlot], input.InvestmentSlot, "積立額[円/月]", from, to)
	if err != nil {
		t.Fatalf("read 積立額: %v", err)
	}
	floor, err := table.ReadYenStep(tables[input.InvestmentSlot], input.InvestmentSlot, "貯蓄維持目標[円]", from, to)
	if err != nil {
		t.Fatalf("read 貯蓄維持目標: %v", err)
	}

	rates := relation.Constant(years, money.NewRate(4, 100))

	balance := theActualBalances(t)
	startsAfter, opening, ok := balance.Latest()
	if !ok {
		t.Fatal("the actuals carry no year to start from")
	}
	recorded := make(map[date.Year]table.Balance, len(balance.Years()))
	for _, y := range balance.Years() {
		held, _ := balance.At(y)
		recorded[y] = held
	}

	mutualAid, err := table.ReadYenStep(
		tables[input.MutualAidContributionSlot], input.MutualAidContributionSlot,
		input.MutualAidContributionColumn, from, to)
	if err != nil {
		t.Fatalf("read %s: %v", input.MutualAidContributionColumn, err)
	}
	receivedIn, serviceYears := table.PensionReceipt(calendar, "夫", mutualAid)

	built, err := table.AssetsTable(table.AssetsInput{
		Timeline:            timelineOfTheBaseProject(t),
		Opening:             opening,
		StartsAfter:         startsAfter,
		Actual:              recorded,
		ContributionMonthly: contribution,
		NISAAllowance:       zeroAllowance(years),
		CashFloor:           floor,
		Return:              rates,
		Crash:               map[date.Year]money.Rate{},
		MutualAid:           mutualAid,
		PensionReceivedIn:   receivedIn,
		PensionServiceYears: serviceYears,
		ResidentTax:         residentTaxTableForTest(t, receivedIn),
		MaturityOfOldNISA:   law.TsumitateNISAMaturityOf(actuals.OldNISABoughtIn),
	})
	if err != nil {
		t.Fatalf("table.AssetsTable: %v", err)
	}
	return built
}

func zeroAllowance(years []date.Year) relation.Table[money.Yen] {
	return relation.Constant[money.Yen](years, 0)
}

func flatPrices(years []date.Year) relation.Table[money.Factor] {
	return relation.Constant(years, money.NoInflation())
}

func theActualBalances(t *testing.T) actuals.BalanceTable {
	t.Helper()

	read, err := tsv.ReadFile("../actuals/balance.tsv")
	if err != nil {
		t.Fatalf("tsv.ReadFile: %v", err)
	}
	balance, err := actuals.ParseBalanceTable(read)
	if err != nil {
		t.Fatalf("actuals.ParseBalanceTable: %v", err)
	}
	return balance
}

func residentTaxTableForTest(t *testing.T, receivedIn *date.Year) law.ResidentRates {
	t.Helper()

	if receivedIn == nil {
		t.Fatal("PensionReceipt が受給年を返さない。掛金の入力か暦を確かめること")
	}

	const municipality = "世田谷区"
	rates, _, _ := law.MustLoadResidentLevies(t, os.DirFS("../"+law.LawDirectory), municipality).
		MustTablesFor(t, municipality).
		MustAt(t, *receivedIn)
	return rates
}

func TestTheAssetsShouldFollowTheMoneyForAnyPlan(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		const years = 12

		timeline := make([]relation.Row[table.TimelineRow], 0, years)
		contribution := make([]relation.Row[money.Yen], 0, years)
		floor := make([]relation.Row[money.Yen], 0, years)
		rates := make([]relation.Row[money.Rate], 0, years)
		for i := range years {
			y := date.Year(2000 + i)
			timeline = append(timeline, relation.Row[table.TimelineRow]{
				Year:  y,
				Value: table.TimelineRow{Balance: money.Yen(rapid.Int64Range(-5_000_000, 5_000_000).Draw(t, "balance"))},
			})
			contribution = append(contribution, relation.Row[money.Yen]{Year: y, Value: 100_000})
			floor = append(floor, relation.Row[money.Yen]{Year: y, Value: 2_500_000})
			rates = append(rates, relation.Row[money.Rate]{Year: y, Value: money.NewRate(4, 100)})
		}

		opening := money.Yen(rapid.Int64Range(0, 20_000_000).Draw(t, "opening"))
		built, err := table.AssetsTable(table.AssetsInput{
			Timeline:            relation.New(timeline),
			Opening:             table.Balance{Cash: opening},
			StartsAfter:         1999,
			ContributionMonthly: relation.New(contribution),
			NISAAllowance:       zeroAllowance(relation.New(timeline).Years()),
			CashFloor:           relation.New(floor),
			Return:              relation.New(rates),
		})
		if err != nil {
			t.Fatalf("AssetsTable: %v", err)
		}

		held := opening
		for _, row := range built.Rows() {
			v := row.Value
			held += v.Balance + v.Returns + v.Crash + v.Shortfall - v.InvestmentTax
			if v.Total != held {
				t.Fatalf("%d: 資産合計 %d, but the money adds up to %d", row.Year, v.Total, held)
			}
		}
	})
}

func TestAMarketCrashShouldOnlyTouchTheInvestments(t *testing.T) {
	years := []date.Year{2000, 2001}
	timeline := relation.New([]relation.Row[table.TimelineRow]{
		{Year: 2000, Value: table.TimelineRow{Balance: 5_000_000}},
		{Year: 2001, Value: table.TimelineRow{}},
	})
	constant := func(v money.Yen) relation.Table[money.Yen] { return relation.Constant(years, v) }
	rates := relation.New([]relation.Row[money.Rate]{
		{Year: 2000, Value: money.NewRate(0, 100)},
		{Year: 2001, Value: money.NewRate(0, 100)},
	})

	built, err := table.AssetsTable(table.AssetsInput{
		Timeline:            timeline,
		Opening:             table.Balance{Cash: 1_000_000},
		StartsAfter:         1999,
		ContributionMonthly: constant(1_000_000),
		NISAAllowance:       zeroAllowance(years),
		CashFloor:           constant(0),
		Return:              rates,
		Crash:               map[date.Year]money.Rate{2001: money.NewRate(-40, 100)},
	})
	if err != nil {
		t.Fatalf("AssetsTable: %v", err)
	}

	before, _ := built.At(date.Year(2000))
	after, _ := built.At(date.Year(2001))

	if want := before.Invested * -40 / 100; after.Crash != want {
		t.Errorf("the crash took %d, want %d", after.Crash, want)
	}
	if after.Cash != before.Cash {
		t.Errorf("the crash moved the cash from %d to %d", before.Cash, after.Cash)
	}
}

func TestTheMutualAidContributionShouldMoveCashIntoThePension(t *testing.T) {
	const contribution money.Yen = 276_000

	drawnAfterTheYearUnderTest := date.Year(2001)

	years := []date.Year{2000, 2001}
	constant := func(v money.Yen) relation.Table[money.Yen] { return relation.Constant(years, v) }

	built, err := table.AssetsTable(table.AssetsInput{
		Timeline: relation.New([]relation.Row[table.TimelineRow]{
			{Year: 2000, Value: table.TimelineRow{Balance: 1_000_000}},
			{Year: 2001},
		}),
		Opening:             table.Balance{Cash: 2_000_000, Locked: 500_000},
		StartsAfter:         1999,
		ContributionMonthly: constant(0),
		NISAAllowance:       zeroAllowance(years),
		CashFloor:           constant(0),
		Return:              relation.Constant(years, money.NewRate(0, 100)),
		MutualAid:           relation.New([]relation.Row[money.Yen]{{Year: 2000, Value: contribution}, {Year: 2001}}),
		PensionReceivedIn:   &drawnAfterTheYearUnderTest,
		PensionServiceYears: 1,
	})
	if err != nil {
		t.Fatalf("AssetsTable: %v", err)
	}

	row, _ := built.At(date.Year(2000))
	if want := money.Yen(2_000_000 + 1_000_000 - contribution); row.Cash != want {
		t.Errorf("貯蓄: %d, want %d", row.Cash, want)
	}
	if want := money.Yen(500_000 + contribution); row.Locked != want {
		t.Errorf("年金資産: %d, want %d", row.Locked, want)
	}
	if want := money.Yen(2_000_000 + 1_000_000 + 500_000); row.Total != want {
		t.Errorf("資産合計: %d, want %d（掛金は世帯の外に出ていない）", row.Total, want)
	}
}

func TestThePensionShouldBeReceivedAtSixty(t *testing.T) {
	const held money.Yen = 8_000_000

	years := []date.Year{2048, 2049, 2050}
	constant := func(v money.Yen) relation.Table[money.Yen] { return relation.Constant(years, v) }
	timeline := relation.Constant(years, table.TimelineRow{})
	rates := relation.Constant(years, money.NewRate(0, 100))

	receivedIn := date.Year(2049)
	built, err := table.AssetsTable(table.AssetsInput{
		Timeline:            timeline,
		Opening:             table.Balance{Cash: 1_000_000, Locked: held},
		StartsAfter:         2047,
		ContributionMonthly: constant(0),
		NISAAllowance:       zeroAllowance(years),
		CashFloor:           constant(10_000_000),
		Return:              rates,
		PensionReceivedIn:   &receivedIn,
		PensionServiceYears: 26,
	})
	if err != nil {
		t.Fatalf("AssetsTable: %v", err)
	}

	before, _ := built.At(date.Year(2048))
	if before.Locked != held {
		t.Errorf("2048 年金資産: %d, want %d", before.Locked, held)
	}
	if before.Available() != 1_000_000 {
		t.Errorf("2048 手が届く額: %d, want 1,000,000（まだ届かない）", before.Available())
	}

	on, _ := built.At(date.Year(2049))
	if on.PensionReceived != held {
		t.Errorf("2049 年金受取: %d, want %d", on.PensionReceived, held)
	}
	if on.PensionTax != 0 {
		t.Errorf("2049 退職所得への税: %d, want 0", on.PensionTax)
	}
	if on.Locked != 0 {
		t.Errorf("2049 年金資産: %d, want 0（受け取ったので残らない）", on.Locked)
	}
	if want := money.Yen(1_000_000 + held); on.Available() != want {
		t.Errorf("2049 手が届く額: %d, want %d", on.Available(), want)
	}

	after, _ := built.At(date.Year(2050))
	if after.PensionReceived != 0 {
		t.Errorf("2050 年金受取: %d, want 0（受け取るのは一度きり）", after.PensionReceived)
	}
}

func TestAPensionLeftStandingAtTheEndShouldBeRefused(t *testing.T) {
	receivedIn := date.Year(2049)

	cases := map[string]struct {
		years       []date.Year
		opening     table.Balance
		mutualAid   map[date.Year]money.Yen
		receivedIn  *date.Year
		serviceYear int
	}{
		"受給年を決めていない": {
			years:   []date.Year{2048, 2049},
			opening: table.Balance{Cash: 1_000_000, Locked: 8_000_000},
		},
		"起点は 0 で、掛金だけが積み上がる": {
			years:     []date.Year{2048, 2049},
			opening:   table.Balance{Cash: 10_000_000},
			mutualAid: map[date.Year]money.Yen{2048: 276_000, 2049: 276_000},
		},
		"受給年より後にも掛金が続く": {
			years:       []date.Year{2048, 2049, 2050},
			opening:     table.Balance{Cash: 10_000_000},
			mutualAid:   map[date.Year]money.Yen{2048: 276_000, 2049: 276_000, 2050: 276_000},
			receivedIn:  &receivedIn,
			serviceYear: 3,
		},
		"掛金が負で、年金資産が負のまま終わる": {
			years:     []date.Year{2048, 2049},
			opening:   table.Balance{Cash: 10_000_000},
			mutualAid: map[date.Year]money.Yen{2049: -276_000},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			constant := func(v money.Yen) relation.Table[money.Yen] { return relation.Constant(c.years, v) }
			mutualAid := make([]relation.Row[money.Yen], 0, len(c.years))
			for _, y := range c.years {
				mutualAid = append(mutualAid, relation.Row[money.Yen]{Year: y, Value: c.mutualAid[y]})
			}

			_, err := table.AssetsTable(table.AssetsInput{
				Timeline:            relation.Constant(c.years, table.TimelineRow{}),
				Opening:             c.opening,
				StartsAfter:         c.years[0] - 1,
				ContributionMonthly: constant(0),
				NISAAllowance:       zeroAllowance(c.years),
				CashFloor:           constant(100_000_000),
				Return:              relation.Constant(c.years, money.NewRate(0, 100)),
				MutualAid:           relation.New(mutualAid),
				PensionReceivedIn:   c.receivedIn,
				PensionServiceYears: c.serviceYear,
			})
			if err == nil {
				t.Fatal("最終年に年金資産が残っているのに、通ってしまった")
			}
			if !strings.Contains(err.Error(), "年金資産") {
				t.Errorf("何が起きたか分からない: %v", err)
			}
		})
	}
}

func TestAssetsRowShouldSayWhatTaxIsBuriedInTheUnrealisedGain(t *testing.T) {
	cases := map[string]struct {
		row       table.AssetsRow
		gain, tax money.Yen
	}{
		"a gain, with a NISA balance beside it that carries none": {
			row:  table.AssetsRow{NISA: 10_000_000, Taxable: 30_000_000, Basis: 10_000_000},
			gain: 20_000_000,
			tax:  4_063_000,
		},
		"a loss": {
			row:  table.AssetsRow{Taxable: 8_000_000, Basis: 10_000_000},
			gain: -2_000_000,
			tax:  0,
		},
		"neither": {
			row:  table.AssetsRow{Taxable: 10_000_000, Basis: 10_000_000},
			gain: 0,
			tax:  0,
		},
		"no taxable holding at all": {
			row:  table.AssetsRow{NISA: 50_000_000},
			gain: 0,
			tax:  0,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := c.row.UnrealisedGain(); got != c.gain {
				t.Errorf("含み益が %v である（%v のはず）", got, c.gain)
			}
			if got := c.row.DeferredTax(); got != c.tax {
				t.Errorf("埋まっている税が %v である（%v のはず）", got, c.tax)
			}
		})
	}
}
