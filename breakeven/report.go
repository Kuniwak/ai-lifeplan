package breakeven

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	DialColumn                     = input.DialColumn
	SettingColumn   tsv.ColumnName = "設定"
	ShortfallColumn tsv.ColumnName = "不足の累計[円]"
	ShortFromColumn tsv.ColumnName = "初めて不足する年"
	FinalColumn     tsv.ColumnName = "最終年の資産[円]"
	DeferredColumn  tsv.ColumnName = "うち含み益に埋まっている税[円]"
	FlattenedColumn tsv.ColumnName = "掃引が消した行[行]"
)

const Step = 10

func Rates(upTo, step int) ([]Setting, error) {
	return Settings(Percent{}, 0, upTo, step)
}

const MostSettings = 1_000

func Settings(kind Kind, from, to, step int) ([]Setting, error) {
	if to <= from {
		return nil, fmt.Errorf("breakeven.Settings: 上限 %d は下限 %d より大きくなければならない", to, from)
	}
	if step <= 0 {
		return nil, fmt.Errorf("breakeven.Settings: 刻みは正でなければならない: %d", step)
	}
	if n := (to-from)/step + 1; n > MostSettings {
		return nil, fmt.Errorf(
			"breakeven.Settings: %d 点になる。掃引できるのは %d 点までである。刻みを大きくすること",
			n, MostSettings)
	}

	settings := make([]Setting, 0, (to-from)/step+1)
	for steps := from; steps <= to; steps += step {
		setting, err := kind.Of(steps)
		if err != nil {
			return nil, fmt.Errorf("breakeven.Settings: %w", err)
		}
		settings = append(settings, setting)
	}
	return settings, nil
}

type Swept struct {
	Dial Dial

	Now Setting

	Outcomes []Outcome

	Painted Painted

	LeftAlone []date.Year
}

func SweepAll(in *plan.Input, dials []Dial, settings []Setting) ([]Swept, error) {
	swept := make([]Swept, 0, len(dials))
	for _, dial := range dials {
		one, err := Sweep(in, dial, settings)
		if err != nil {
			return nil, err
		}
		swept = append(swept, one)
	}
	return swept, nil
}

func SweptTable(swept []Swept) *tsv.Table {
	out := &tsv.Table{Header: []tsv.ColumnName{
		DialColumn, SettingColumn, ShortfallColumn, ShortFromColumn, FinalColumn,
		DeferredColumn, FlattenedColumn,
	}}
	for _, s := range swept {
		out.Rows = append(out.Rows, SweepTable(s).Rows...)
	}
	return out
}

type Range struct {
	low, high Setting
	source    string
}

func NewRange(low, high Setting, source string) (Range, error) {
	if !low.Comparable(high) {
		return Range{}, fmt.Errorf(
			"breakeven.NewRange: 下限 %s（%s）と上限 %s（%s）の単位が違う",
			low, low.Unit(), high, high.Unit())
	}
	if low.Cmp(high) > 0 {
		return Range{}, fmt.Errorf(
			"breakeven.NewRange: 下限 %s が上限 %s より大きい", low, high)
	}
	return Range{low: low, high: high, source: source}, nil
}

func RangesFrom(in *plan.Input) (map[string]Range, error) {
	table, err := in.Table(input.ReferenceRangeSlot)
	if err != nil {
		return nil, err
	}

	at := make(map[tsv.ColumnName]int, 4)
	for _, column := range []tsv.ColumnName{input.DialColumn, "下限", "上限", "出典"} {
		i, ok := table.ColumnIndex(column)
		if !ok {
			return nil, fmt.Errorf("breakeven: %s に %q 列が無い", input.ReferenceRangeSlot, column)
		}
		at[column] = i
	}

	kinds := make(map[string]Kind, len(Dials(0)))
	for _, dial := range Dials(0) {
		kinds[dial.Name] = dial.Kind
	}

	ranges := make(map[string]Range, len(table.Rows))
	for row, fields := range table.Rows {
		name := fields[at[input.DialColumn]]
		kind, ok := kinds[name]
		if !ok {
			return nil, fmt.Errorf(
				"breakeven: %s: row %d: %q は掃引するダイヤルではないので、幅をどの単位で読むか決められない",
				input.ReferenceRangeSlot, row+1, name)
		}

		low, err := kind.Parse(fields[at["下限"]])
		if err != nil {
			return nil, fmt.Errorf("breakeven: %s: row %d: %w", input.ReferenceRangeSlot, row+1, err)
		}
		high, err := kind.Parse(fields[at["上限"]])
		if err != nil {
			return nil, fmt.Errorf("breakeven: %s: row %d: %w", input.ReferenceRangeSlot, row+1, err)
		}
		r, err := NewRange(low, high, fields[at["出典"]])
		if err != nil {
			return nil, fmt.Errorf("breakeven: %s: row %d: %w", input.ReferenceRangeSlot, row+1, err)
		}
		ranges[name] = r
	}

	for _, dial := range Dials(0) {
		if _, ok := ranges[dial.Name]; !ok {
			return nil, fmt.Errorf(
				"breakeven: %s に %q の行が無い。掃引するダイヤルには幅が要る",
				input.ReferenceRangeSlot, dial.Name)
		}
	}
	return ranges, nil
}

func Against(swept Swept, ranges map[string]Range) string {
	r, ok := ranges[swept.Dial.Name]
	if !ok {
		return fmt.Sprintf("%s: 議論の幅が書かれていないので、境目が近いかどうかを言えない", swept.Dial.Name)
	}
	if !swept.Now.IsZero() && !r.low.Comparable(swept.Now) {
		return fmt.Sprintf("%s: 議論の幅が %s で書かれていて、このダイヤル（%s）とは単位が違う",
			swept.Dial.Name, r.low.Unit(), swept.Now.Unit())
	}

	turns := Turns(swept.Outcomes)
	if len(turns) != 1 {
		return fmt.Sprintf("%s: 議論の幅は %s 〜 %s。%s（いまの前提は %s）",
			swept.Dial.Name, r.low, r.high, r.source, swept.Now)
	}

	t := turns[0]
	var edge Setting
	var safeAll, failsAll bool
	if t.Before.Fails() {
		edge = t.After.Setting
		safeAll, failsAll = r.low.Cmp(edge) >= 0, r.high.Cmp(edge) < 0
	} else {
		edge = t.Before.Setting
		safeAll, failsAll = r.high.Cmp(edge) <= 0, r.low.Cmp(edge) > 0
	}

	var where string
	switch {
	case safeAll:
		where = "幅のどこに置いても不足しない"
	case failsAll:
		where = "**幅のどこに置いても不足する**"
	default:
		where = "**幅の中にある。幅のどこに置くかで不足するかどうかが変わる**"
	}

	return fmt.Sprintf("%s: 境目 %s、議論の幅は %s 〜 %s。%s。%s。%s",
		swept.Dial.Name, edge, r.low, r.high,
		where, standingAgainst(swept.Now, r), r.source)
}

func standingAgainst(now Setting, r Range) string {
	switch {
	case now.Cmp(r.low) < 0:
		return fmt.Sprintf("**いまの前提 %s は幅より下**", now)
	case now.Cmp(r.high) > 0:
		return fmt.Sprintf("**いまの前提 %s は幅より上**", now)
	default:
		return fmt.Sprintf("いまの前提 %s は幅の中", now)
	}
}

func Report(swept []Swept, ranges map[string]Range) []string {
	lines := make([]string, 0, 3*len(swept))
	for _, s := range swept {
		lines = append(lines, Summary(s), "  "+Against(s, ranges))
		if warning := s.Warning(); warning != "" {
			lines = append(lines, "  "+warning)
		}
	}
	return lines
}

func Dials(from date.Year) []Dial {
	return []Dial{
		mustDial(input.InvestmentReturnSlot, "実質運用利率", from),
		mustDial(input.InflationSlot, input.InflationRateColumn, from),
	}
}

func mustDial(slot tsv.Slot, column tsv.ColumnName, from date.Year) Dial {
	dial, err := DialOf(slot, column, from)
	if err != nil {
		panic(err)
	}
	return dial
}

func SweepTable(swept Swept) *tsv.Table {
	dial, outcomes := swept.Dial, swept.Outcomes
	out := &tsv.Table{Header: []tsv.ColumnName{
		DialColumn, SettingColumn, ShortfallColumn, ShortFromColumn, FinalColumn,
		DeferredColumn, FlattenedColumn,
	}}
	flattened := fmt.Sprint(len(swept.Painted.Flattened()))
	for _, o := range outcomes {
		shortFrom := o.ShortFromField()
		out.Rows = append(out.Rows, []string{
			dial.Name, o.Setting.Field(),
			fmt.Sprint(int64(o.Shortfall)), shortFrom, fmt.Sprint(int64(o.Final)),
			fmt.Sprint(int64(o.Deferred)), flattened,
		})
	}
	return out
}

func Summary(swept Swept) string {
	dial, outcomes := swept.Dial, swept.Outcomes
	turns := Turns(outcomes)

	switch {
	case len(outcomes) == 0:
		return fmt.Sprintf("%s: 掃引していない", dial.Name)

	case len(turns) == 0 && outcomes[0].Fails():
		return fmt.Sprintf("%s: %s から %s まで、どこに置いても不足する。**このダイヤルでは足りない**",
			dial.Name, outcomes[0].Setting, outcomes[len(outcomes)-1].Setting)

	case len(turns) == 0:
		return fmt.Sprintf("%s: %s から %s まで、どこに置いても不足しない。**推定しなくてよい**",
			dial.Name, outcomes[0].Setting, outcomes[len(outcomes)-1].Setting)

	case len(turns) == 1:
		t := turns[0]
		safe, failing := fmt.Sprintf("%s 以上", t.After.Setting), t.Before
		if !t.Before.Fails() {
			safe, failing = fmt.Sprintf("%s 以下", t.Before.Setting), t.After
		}
		return fmt.Sprintf("%s: %sなら不足しない。%s では %d 年から %d 円の不足。%s",
			dial.Name, safe, failing.Setting, int(failing.ShortFrom), int64(failing.Shortfall),
			standing(swept, t))

	default:
		s := fmt.Sprintf("%s: **境目が %d 個ある。単調でない**", dial.Name, len(turns))
		for _, t := range turns {
			s += fmt.Sprintf("\n    %s → %s で %s",
				t.Before.Setting, t.After.Setting, turned(t))
		}
		return s
	}
}

func standing(swept Swept, t Turn) string {
	now := swept.Now
	side := "危ないほう"
	switch {
	case t.Before.Fails() && now.Cmp(t.After.Setting) >= 0:
		side = "安全なほう"
	case !t.Before.Fails() && now.Cmp(t.Before.Setting) <= 0:
		side = "安全なほう"
	}
	return fmt.Sprintf("いまの前提は %s（%s）", now, side)
}

func turned(t Turn) string {
	if t.Before.Fails() {
		return "不足しなくなる"
	}
	return "不足するようになる"
}
