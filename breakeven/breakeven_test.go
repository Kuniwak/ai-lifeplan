package breakeven_test

import (
	"fmt"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/table"
	"slices"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/breakeven"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/panictest"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

func theBaseProject(t *testing.T) *plan.Input {
	t.Helper()

	in, err := plan.Load(plan.Sources{Root: "..", ProjectPath: "../projects/classic.tsv"})
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}
	return in
}

func sweepsFrom(t *testing.T, in *plan.Input) date.Year {
	t.Helper()

	startsAfter, err := in.StartsAfter()
	if err != nil {
		t.Fatalf("plan.Input.StartsAfter: %v", err)
	}
	return startsAfter + 1
}

func theInflation(t *testing.T, in *plan.Input) breakeven.Dial {
	t.Helper()

	for _, dial := range breakeven.Dials(sweepsFrom(t, in)) {
		if dial.Slot == input.InflationSlot {
			return dial
		}
	}
	t.Fatal("インフレ率のダイヤルが無い")
	return breakeven.Dial{}
}

var theReturn = dialOf(input.InvestmentReturnSlot, "実質運用利率", 2026)

func dialOf(slot tsv.Slot, column tsv.ColumnName, from date.Year) breakeven.Dial {
	dial, err := breakeven.DialOf(slot, column, from)
	if err != nil {
		panic(err)
	}
	return dial
}

func rate(t *testing.T, num, den int64) breakeven.Setting {
	t.Helper()

	setting, err := breakeven.RateSetting(money.NewRate(num, den))
	if err != nil {
		t.Fatalf("breakeven.RateSetting: %v", err)
	}
	return setting
}

func TestTurns(t *testing.T) {
	for _, c := range []struct {
		name  string
		outs  []breakeven.Outcome
		turns int
	}{
		{"不足する側から不足しない側へ一度だけ変わる", outcomes(300, 100, 0, 0), 1},
		{"どこでも不足しない", outcomes(0, 0, 0), 0},
		{"どこでも不足する", outcomes(5, 4, 3), 0},
		{"行って戻ってまた行く", outcomes(300, 0, 0, 200, 0), 3},
	} {
		t.Run(string(c.name), func(t *testing.T) {
			if got := breakeven.Turns(c.outs); len(got) != c.turns {
				t.Errorf("境目は %d 個のはずが %d 個: %+v", c.turns, len(got), got)
			}
		})
	}
}

func outcomes(shortfalls ...money.Yen) []breakeven.Outcome {
	built := make([]breakeven.Outcome, 0, len(shortfalls))
	for i, short := range shortfalls {
		setting, err := breakeven.RateSetting(money.NewRate(int64(i), 100))
		if err != nil {
			panic(err)
		}
		built = append(built, breakeven.Outcome{
			Setting: setting,
			Outcome: plan.Outcome{Shortfall: short},
		})
	}
	return built
}

func sweptAt(now breakeven.Setting, shortfalls ...money.Yen) breakeven.Swept {
	return breakeven.Swept{Dial: theReturn, Now: now, Outcomes: outcomes(shortfalls...)}
}

func TestSweepShouldRefuseToTurnADialOutsideTheProjection(t *testing.T) {
	in := theBaseProject(t)
	startsAfter := sweepsFrom(t, in) - 1

	for _, from := range []date.Year{startsAfter - 1, startsAfter, startsAfter + 2, 2200} {
		dial := theReturn
		dial.From = from
		if _, err := breakeven.Sweep(in, dial, []breakeven.Setting{rate(t, 4, 100)}); err == nil {
			t.Errorf("%d 年から回すのを拒んでいない（実績は %d 年まで）", from, startsAfter)
		}
	}
}

func TestSummaryShouldNameTheFailingSideWhicheverWayTheDialRuns(t *testing.T) {
	rising := breakeven.Summary(sweptAt(rate(t, 4, 100), 300, 0))
	falling := breakeven.Summary(breakeven.Swept{
		Dial: breakeven.Dial{Name: "インフレ率"}, Now: rate(t, 0, 100),
		Outcomes: outcomes(0, 300),
	})

	if !strings.Contains(rising, "1.00% 以上なら不足しない") || !strings.Contains(rising, "0.00% では") {
		t.Errorf("上げると助かるダイヤルで不足する側が名指しされていない: %q", rising)
	}
	if !strings.Contains(falling, "0.00% 以下なら不足しない") || !strings.Contains(falling, "1.00% では") {
		t.Errorf("上げると苦しくなるダイヤルで不足する側が名指しされていない: %q", falling)
	}
}

func TestSummaryShouldSayWhenADialNeverDecidesTheAnswer(t *testing.T) {
	for _, c := range []struct {
		name table.PersonName
		outs []breakeven.Outcome
		want string
	}{
		{"どこでも不足しない", outcomes(0, 0, 0), "推定しなくてよい"},
		{"どこでも不足する", outcomes(3, 2, 1), "このダイヤルでは足りない"},
	} {
		t.Run(string(c.name), func(t *testing.T) {
			swept := breakeven.Swept{Dial: theReturn, Now: rate(t, 4, 100), Outcomes: c.outs}
			if got := breakeven.Summary(swept); !strings.Contains(got, c.want) {
				t.Errorf("%q を含まない: %q", c.want, got)
			}
		})
	}
}

func TestSummaryShouldNameEveryTurnWhenTheSweepIsNotMonotone(t *testing.T) {
	got := breakeven.Summary(sweptAt(rate(t, 4, 100), 300, 0, 0, 200, 0))

	if !strings.Contains(got, "単調でない") {
		t.Errorf("単調でないことが書かれていない: %q", got)
	}
	if strings.Count(got, "→") != 3 {
		t.Errorf("境目 3 つが並んでいない: %q", got)
	}
}

func TestSweepTableShouldLeaveTheYearBlankWhenNothingRanOut(t *testing.T) {
	table := breakeven.SweepTable(sweptAt(rate(t, 4, 100), 0, 300))

	at, ok := table.ColumnIndex(breakeven.ShortFromColumn)
	if !ok {
		t.Fatalf("%q 列が無い", breakeven.ShortFromColumn)
	}
	if got := table.Rows[0][at]; got != "" {
		t.Errorf("不足しない年の %q が %q である", breakeven.ShortFromColumn, got)
	}
}

func TestRateSettingShouldRefuseARateFinerThanATableHolds(t *testing.T) {
	if _, err := breakeven.RateSetting(money.NewRate(1, 3)); err == nil {
		t.Error("1/3 という率の設定が作れてしまう")
	}
}

func aRange(t *testing.T, low, high int64) map[string]breakeven.Range {
	t.Helper()

	r, err := breakeven.NewRange(rate(t, low, 10_000), rate(t, high, 10_000), "出典")
	if err != nil {
		t.Fatalf("breakeven.NewRange: %v", err)
	}
	return map[string]breakeven.Range{theReturn.Name: r}
}

func TestAgainstShouldSayWhetherTheWholeRangeIsSafe(t *testing.T) {
	rising := outcomes(300, 0, 0)
	falling := outcomes(0, 0, 300)

	for _, c := range []struct {
		name      string
		outs      []breakeven.Outcome
		low, high int64
		want      string
	}{
		{"上げると助かる／幅が境目より上", rising, 140, 340, "幅のどこに置いても不足しない"},
		{"上げると助かる／幅が境目より下", rising, 10, 50, "幅のどこに置いても不足する"},
		{"上げると助かる／幅が境目をまたぐ", rising, 50, 340, "幅の中にある"},
		{"上げると苦しくなる／幅が境目より下", falling, 10, 50, "幅のどこに置いても不足しない"},
		{"上げると苦しくなる／幅が境目より上", falling, 140, 340, "幅のどこに置いても不足する"},
		{"上げると苦しくなる／幅が境目をまたぐ", falling, 50, 340, "幅の中にある"},
	} {
		t.Run(string(c.name), func(t *testing.T) {
			swept := breakeven.Swept{Dial: theReturn, Now: rate(t, 4, 100), Outcomes: c.outs}
			if got := breakeven.Against(swept, aRange(t, c.low, c.high)); !strings.Contains(got, c.want) {
				t.Errorf("%q を含まない: %q", c.want, got)
			}
		})
	}
}

func TestAgainstShouldSayWhereTheProjectsOwnSettingSits(t *testing.T) {
	for _, c := range []struct {
		name table.PersonName
		now  int64
		want string
	}{
		{"幅より上", 400, "いまの前提 4.00% は幅より上"},
		{"幅より下", 100, "いまの前提 1.00% は幅より下"},
		{"幅の中", 200, "いまの前提 2.00% は幅の中"},
	} {
		t.Run(string(c.name), func(t *testing.T) {
			swept := breakeven.Swept{
				Dial: theReturn, Now: rate(t, c.now, 10_000), Outcomes: outcomes(300, 0, 0),
			}
			if got := breakeven.Against(swept, aRange(t, 140, 340)); !strings.Contains(got, c.want) {
				t.Errorf("%q を含まない: %q", c.want, got)
			}
		})
	}
}

func TestNewRangeShouldRefuseABottomAboveItsTop(t *testing.T) {
	if _, err := breakeven.NewRange(rate(t, 340, 10_000), rate(t, 140, 10_000), ""); err == nil {
		t.Error("下限が上限より大きい幅を拒んでいない")
	}
}

func TestAgainstShouldSayWhenNoRangeIsWritten(t *testing.T) {
	swept := breakeven.Swept{Dial: theReturn, Now: rate(t, 4, 100), Outcomes: outcomes(300, 0)}

	if got := breakeven.Against(swept, map[string]breakeven.Range{}); !strings.Contains(got, "言えない") {
		t.Errorf("幅が無いことが言われていない: %q", got)
	}
}

func TestKindOfShouldComeFromTheShapeOfTheInput(t *testing.T) {
	for _, c := range []struct {
		slot   tsv.Slot
		column tsv.ColumnName
		unit   string
	}{
		{input.InvestmentReturnSlot, "実質運用利率", "%"},
		{input.LivingCostSlot, "生活費[円/月]", "円"},
		{input.IncomeHusbandSlot, "給与収入[円/年]", "円"},
	} {
		t.Run(string(c.slot), func(t *testing.T) {
			kind, err := breakeven.KindOf(c.slot, c.column)
			if err != nil {
				t.Fatalf("breakeven.KindOf: %v", err)
			}
			if kind.Unit() != c.unit {
				t.Errorf("%s の %s を %s のダイヤルとして読んでいる（%s のはず）",
					c.slot, c.column, kind.Unit(), c.unit)
			}
		})
	}
}

func TestKindOfShouldRefuseAColumnThatIsNotANumber(t *testing.T) {
	for _, c := range []struct {
		slot   tsv.Slot
		column tsv.ColumnName
	}{
		{input.HouseholdSlot, "続柄"},
		{input.LivingCostSlot, "そんな列は無い"},
		{"そんな slot は無い", "生活費[円/月]"},
	} {
		t.Run(string(c.slot)+":"+string(c.column), func(t *testing.T) {
			if _, err := breakeven.KindOf(c.slot, c.column); err == nil {
				t.Errorf("%s の %q をダイヤルとして受け付けている", c.slot, c.column)
			}
		})
	}
}

func TestDialOfShouldRefuseAListOfEvents(t *testing.T) {
	for _, c := range []struct {
		slot   tsv.Slot
		column tsv.ColumnName
	}{
		{input.ExtraordinaryCostSlot, "費用[円]"},
		{input.PropertyInsuranceSlot, "火災保険料[円]"},
		{input.HousingSlot, "頭金[円]"},
	} {
		t.Run(string(c.slot), func(t *testing.T) {
			if _, err := breakeven.DialOf(c.slot, c.column, 2026); err == nil {
				t.Error("起きたことの一覧をダイヤルとして受け付けている")
			}
		})
	}

	if _, err := breakeven.DialOf(input.LivingCostSlot, "生活費[円/月]", 2026); err != nil {
		t.Errorf("step の表を拒んでいる: %v", err)
	}
}

func TestDialOfShouldRefuseATableThatIsNotReadByYear(t *testing.T) {
	if _, err := breakeven.DialOf(input.PensionSlot, "基礎年金額[円/年]", 2026); err == nil {
		t.Error("Lookup の表をダイヤルとして受け付けている")
	}
}

func TestPostponeDialTurnShouldMoveOnlyTheLastRowsYear(t *testing.T) {
	in := theBaseProject(t)
	written, err := in.Table(input.IncomeHusbandSlot)
	if err != nil {
		t.Fatalf("plan.Input.Table: %v", err)
	}

	dial, err := breakeven.PostponeDialOf(input.IncomeHusbandSlot, 2026)
	if err != nil {
		t.Fatalf("breakeven.PostponeDialOf: %v", err)
	}
	turned, err := dial.Turn(written, breakeven.YearsSetting(5))
	if err != nil {
		t.Fatalf("breakeven.Dial.Turn: %v", err)
	}

	if len(turned.Rows) != len(written.Rows) {
		t.Fatalf("行が %d 行から %d 行になった", len(written.Rows), len(turned.Rows))
	}
	if !slices.Equal(turned.Header, written.Header) {
		t.Fatalf("列が %v から %v になった", written.Header, turned.Header)
	}

	yearAt, _ := written.ColumnIndex(input.YearColumn)
	last := len(written.Rows) - 1

	for row := range written.Rows {
		if row != last {
			if !slices.Equal(written.Rows[row], turned.Rows[row]) {
				t.Errorf("最後の行でないのに %d 行目が %v から %v に変わった",
					row, written.Rows[row], turned.Rows[row])
			}
			continue
		}
		wasYear, err := date.ParseYear(written.Rows[row][yearAt])
		if err != nil {
			t.Fatalf("date.ParseYear: %v", err)
		}
		for at := range written.Header {
			if at == yearAt {
				if want := fmt.Sprint(int(wasYear) + 5); turned.Rows[row][at] != want {
					t.Errorf("最後の行の年が %q である（%q のはず）", turned.Rows[row][at], want)
				}
				continue
			}
			if written.Rows[row][at] != turned.Rows[row][at] {
				t.Errorf("最後の行の %q が %q から %q に変わった",
					written.Header[at], written.Rows[row][at], turned.Rows[row][at])
			}
		}
	}
}

func TestPostponeDialOfShouldRefuseASlotThatIsNotAboutWorkingYears(t *testing.T) {
	for _, slot := range []tsv.Slot{input.PensionSlot, input.LivingCostSlot, input.InflationSlot} {
		if _, err := breakeven.PostponeDialOf(slot, 2026); err == nil {
			t.Errorf("%s を延長ダイヤルとして受け付けている", slot)
		}
	}
}

func TestPostponeDialWrittenShouldBeZeroYears(t *testing.T) {
	in := theBaseProject(t)
	written, err := in.Table(input.IncomeHusbandSlot)
	if err != nil {
		t.Fatalf("plan.Input.Table: %v", err)
	}
	dial, err := breakeven.PostponeDialOf(input.IncomeHusbandSlot, 2026)
	if err != nil {
		t.Fatalf("breakeven.PostponeDialOf: %v", err)
	}
	now, err := dial.Written(written)
	if err != nil {
		t.Fatalf("breakeven.Dial.Written: %v", err)
	}
	if want := breakeven.YearsSetting(0); now.Cmp(want) != 0 {
		t.Errorf("延長ダイヤルの現在値が %v である（0 年のはず）", now)
	}
}

func TestPostponeDialShouldNotPaintOverAnything(t *testing.T) {
	in := theBaseProject(t)
	written, err := in.Table(input.IncomeHusbandSlot)
	if err != nil {
		t.Fatalf("plan.Input.Table: %v", err)
	}
	dial, err := breakeven.PostponeDialOf(input.IncomeHusbandSlot, 2026)
	if err != nil {
		t.Fatalf("breakeven.PostponeDialOf: %v", err)
	}
	now, err := dial.Written(written)
	if err != nil {
		t.Fatalf("breakeven.Dial.Written: %v", err)
	}
	painted, err := dial.PaintedOver(written, now)
	if err != nil {
		t.Fatalf("breakeven.Dial.PaintedOver: %v", err)
	}
	if got := len(painted.Rows); got != 0 {
		t.Errorf("塗り潰す行が %d 行ある（0 行のはず）", got)
	}
	if warning := (breakeven.Swept{Dial: dial, Painted: painted}).Warning(); warning != "" {
		t.Errorf("塗り潰していないのに何か言っている: %q", warning)
	}
}

func TestSettingsOfDifferentDialsShouldNotBeOrdered(t *testing.T) {
	yen, percent := breakeven.YenSetting(250_000), rate(t, 2, 100)

	if yen.Comparable(percent) {
		t.Error("円 と % の設定が比べられることになっている")
	}
	if !yen.Comparable(breakeven.YenSetting(1)) {
		t.Error("円 同士が比べられないことになっている")
	}

	if panictest.Recovered(func() { yen.Cmp(percent) }) == nil {
		t.Error("単位の違う設定を Cmp に渡して何も起きない")
	}
}

func TestNewRangeShouldRefuseBoundsInDifferentUnits(t *testing.T) {
	if _, err := breakeven.NewRange(breakeven.YenSetting(1), rate(t, 2, 100), ""); err == nil {
		t.Error("下限が円で上限が率の幅を作れてしまう")
	}
}

func TestAgainstShouldRefuseARangeInAnotherUnit(t *testing.T) {
	swept := breakeven.Swept{
		Dial:     dialOf(input.LivingCostSlot, "生活費[円/月]", 2026),
		Now:      breakeven.YenSetting(300_000),
		Outcomes: outcomes(0, 0, 300),
	}
	ranges := map[string]breakeven.Range{"生活費[円/月]": aRange(t, 40, 200)["実質運用利率"]}

	if got := breakeven.Against(swept, ranges); !strings.Contains(got, "単位が違う") {
		t.Errorf("率の幅を円のダイヤルに当てている: %q", got)
	}
}

func TestSweepShouldSeeAShapeLostToACliffAtTheYearItStartsFrom(t *testing.T) {
	in := theBaseProject(t)
	from := sweepsFrom(t, in)
	dial := dialOf(input.IncomeHusbandSlot, "給与収入[円/年]", from)

	written, err := in.Table(input.IncomeHusbandSlot)
	if err != nil {
		t.Fatalf("plan.Input.Table: %v", err)
	}
	yearAt, salaryAt, ok := 0, 1, false
	if yearAt, ok = written.ColumnIndex(input.YearColumn); !ok {
		t.Fatal("西暦 列が無い")
	}
	if salaryAt, ok = written.ColumnIndex("給与収入[円/年]"); !ok {
		t.Fatal("給与収入[円/年] 列が無い")
	}

	flat := &tsv.Table{Header: slices.Clone(written.Header)}
	for _, fields := range written.Rows {
		row := slices.Clone(fields)
		year, err := date.ParseYear(row[yearAt])
		if err != nil {
			t.Fatalf("%v", err)
		}
		if year > from {
			row[salaryAt] = "3,000,000"
		}
		flat.Rows = append(flat.Rows, row)
	}

	now, err := dial.Written(flat)
	if err != nil {
		t.Fatalf("breakeven.Dial.Written: %v", err)
	}
	painted, err := dial.PaintedOver(flat, now)
	if err != nil {
		t.Fatalf("breakeven.Dial.PaintedOver: %v", err)
	}

	if got := len(painted.Flattened()); got != len(painted.Rows) {
		t.Errorf("形の消える行が %d 行である（塗り潰す %d 行すべてのはず）",
			got, len(painted.Rows))
	}
	warning := breakeven.Swept{Dial: dial, Painted: painted}.Warning()
	if !strings.Contains(warning, "消す") {
		t.Errorf("段差が消えることを言っていない: %q", warning)
	}
}

func TestDialTurnShouldNotPutPayIntoAYearWithNoWorkingHours(t *testing.T) {
	in := theBaseProject(t)
	written, err := in.Table(input.IncomeHusbandSlot)
	if err != nil {
		t.Fatalf("plan.Input.Table: %v", err)
	}

	dial := dialOf(input.IncomeHusbandSlot, "給与収入[円/年]", 2026)
	turned, err := dial.Turn(written, breakeven.YenSetting(10_000_000))
	if err != nil {
		t.Fatalf("breakeven.Dial.Turn: %v", err)
	}

	yearAt, _ := turned.ColumnIndex(input.YearColumn)
	salaryAt, _ := turned.ColumnIndex("給与収入[円/年]")
	hoursAt, _ := turned.ColumnIndex(input.WeeklyHoursColumn)

	retired, working := 0, 0
	for row, fields := range turned.Rows {
		year, err := date.ParseYear(fields[yearAt])
		if err != nil {
			t.Fatalf("date.ParseYear: %v", err)
		}
		if year < 2026 {
			continue
		}
		if fields[hoursAt] == "0" {
			retired++
			if fields[salaryAt] != "0" {
				t.Errorf("%d 年は週所定労働時間 0 なのに給与収入が %q になった。"+
					"働いていない年に給与が立つと被用者保険から外れる（row %d）",
					year, fields[salaryAt], row+1)
			}
			continue
		}
		working++
		if fields[salaryAt] != "10000000" {
			t.Errorf("%d 年の給与収入が %q である（10000000 のはず）", year, fields[salaryAt])
		}
	}

	if retired == 0 {
		t.Error("週所定労働時間 0 の年が 1 つも無い。この検査は何も見ていない")
	}
	if working == 0 {
		t.Error("給与が書き換わった年が 1 つも無い。ダイヤルが何も回していない")
	}
}

func TestSweepShouldRefuseADialThatWouldTurnNothing(t *testing.T) {
	in := theBaseProject(t)

	written, err := in.Table(input.IncomeWifeSlot)
	if err != nil {
		t.Fatalf("plan.Input.Table: %v", err)
	}
	hoursAt, ok := written.ColumnIndex(input.WeeklyHoursColumn)
	if !ok {
		t.Fatalf("%s の列が無い", input.WeeklyHoursColumn)
	}
	idle := &tsv.Table{Header: slices.Clone(written.Header)}
	for _, fields := range written.Rows {
		row := slices.Clone(fields)
		row[hoursAt] = "0"
		idle.Rows = append(idle.Rows, row)
	}

	dial := dialOf(input.IncomeWifeSlot, "給与収入[円/年]", 2026)
	_, err = breakeven.Sweep(in.With(input.IncomeWifeSlot, idle), dial,
		[]breakeven.Setting{breakeven.YenSetting(1_000_000)})
	if err == nil {
		t.Fatal("週所定労働時間 が全年 0 の表を掃引したのに、何も言われなかった")
	}
	for _, want := range []string{"週所定労働時間", "給与収入[円/年]"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("エラーが %q に触れていない: %v", want, err)
		}
	}
}

func TestDialTurnShouldNotPutABonusIntoAYearWithNoWorkingHours(t *testing.T) {
	in := theBaseProject(t)
	written, err := in.Table(input.IncomeHusbandSlot)
	if err != nil {
		t.Fatalf("plan.Input.Table: %v", err)
	}

	dial := dialOf(input.IncomeHusbandSlot, input.BonusColumn, 2026)
	turned, err := dial.Turn(written, breakeven.YenSetting(3_000_000))
	if err != nil {
		t.Fatalf("breakeven.Dial.Turn: %v", err)
	}

	yearAt, _ := turned.ColumnIndex(input.YearColumn)
	bonusAt, _ := turned.ColumnIndex(input.BonusColumn)
	hoursAt, _ := turned.ColumnIndex(input.WeeklyHoursColumn)

	retired := 0
	for _, fields := range turned.Rows {
		year, err := date.ParseYear(fields[yearAt])
		if err != nil {
			t.Fatalf("date.ParseYear: %v", err)
		}
		if year < 2026 || fields[hoursAt] != "0" {
			continue
		}
		retired++
		if fields[bonusAt] != "0" {
			t.Errorf("%d 年は週所定労働時間 0 なのに賞与収入が %q になった", year, fields[bonusAt])
		}
	}
	if retired == 0 {
		t.Error("週所定労働時間 0 の年が 1 つも無い。この検査は何も見ていない")
	}
}

func TestDialTurnShouldKeepEveryColumnOfAWideTable(t *testing.T) {
	in := theBaseProject(t)

	for _, c := range []struct {
		slot   tsv.Slot
		column tsv.ColumnName
	}{
		{input.InvestmentSlot, "積立額[円/月]"},
		{input.IncomeHusbandSlot, "給与収入[円/年]"},
	} {
		t.Run(string(c.slot), func(t *testing.T) {
			written, err := in.Table(c.slot)
			if err != nil {
				t.Fatalf("plan.Input.Table: %v", err)
			}
			if len(written.Header) <= 2 {
				t.Fatalf("%s は %d 列しかない。この検査は 3 列以上の表のためのもの",
					c.slot, len(written.Header))
			}

			turned, err := dialOf(c.slot, c.column, 2026).Turn(written, breakeven.YenSetting(123_456))
			if err != nil {
				t.Fatalf("breakeven.Dial.Turn: %v", err)
			}

			if !slices.Equal(turned.Header, written.Header) {
				t.Fatalf("列が %v から %v になった", written.Header, turned.Header)
			}
			for row := range written.Rows {
				if got, want := len(turned.Rows[row]), len(written.Rows[row]); got != want {
					t.Errorf("%d 行目のセルが %d 個になった（%d 個のはず）", row+1, got, want)
				}
			}
		})
	}
}
