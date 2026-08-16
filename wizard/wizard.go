package wizard

import (
	"fmt"
	"path/filepath"

	"github.com/Kuniwak/lifeplan/breakeven"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

type Question struct {
	Name string

	Why string

	Dial breakeven.Dial

	Choices []Choice
}

func (q Question) AnsweredByTable() bool { return q.Dial.Slot == "" }

type Choice struct {
	Label   string
	Setting breakeven.Setting

	Path string

	Slot tsv.Slot
}

type Answer struct {
	Question Question
	Setting  breakeven.Setting

	Choice Choice
}

func TurnsFrom(in *plan.Input) (date.Year, error) {
	after, err := in.StartsAfter()
	if err != nil {
		return 0, fmt.Errorf("wizard.TurnsFrom: %w", err)
	}
	return after + 1, nil
}

func Questions(in *plan.Input) ([]Question, error) {
	from, err := TurnsFrom(in)
	if err != nil {
		return nil, err
	}

	yen := func(amounts ...money.Yen) []Choice {
		out := make([]Choice, 0, len(amounts))
		for _, a := range amounts {
			out = append(out, Choice{Label: a.String() + " 円", Setting: breakeven.YenSetting(a)})
		}
		return out
	}
	rates := func(hundredths ...int64) ([]Choice, error) {
		out := make([]Choice, 0, len(hundredths))
		for _, h := range hundredths {
			setting, err := breakeven.RateSetting(money.NewRate(h, 10_000))
			if err != nil {
				return nil, fmt.Errorf("wizard.Questions: %w", err)
			}
			out = append(out, Choice{Label: money.NewRate(h, 10_000).Percent(), Setting: setting})
		}
		return out, nil
	}

	returns, err := rates(0, 200, 400, 600)
	if err != nil {
		return nil, err
	}
	inflation, err := rates(0, 100, 200)
	if err != nil {
		return nil, err
	}

	catalogue := []struct {
		name, why string
		slot      tsv.Slot
		column    tsv.ColumnName
		choices   []Choice
	}{
		{
			name: "夫婦生活費", slot: input.LivingCostSlot, column: input.CoupleLivingColumn,
			why: "**この計画でいちばん大きく効く。**毎月の額を 5万 動かすだけで、" +
				"最終年の資産は倍にも半分にもなり、尽きる木と尽きない木が入れ替わる。",
			choices: yen(250_000, 275_000, 300_000, 325_000, 350_000),
		},
		{
			name: "実質運用利率", slot: input.InvestmentReturnSlot, column: input.ReturnColumn,
			why: "**分岐点を決めるのはこれである。**`lifeplan breakeven` が「どこを境に" +
				"結論が変わるか」を出すのも、既定ではこのダイヤルである。",
			choices: returns,
		},
		{
			name: "インフレ率", slot: input.InflationSlot, column: input.InflationRateColumn,
			why: "**名目では増え、実質では減る。**上げると資産の名目額は大きくなるが、" +
				"実質の手取りは下がる。",
			choices: inflation,
		},
	}

	questions := make([]Question, 0, len(catalogue)+1)
	for _, c := range catalogue {
		dial, err := breakeven.DialOf(c.slot, c.column, from)
		if err != nil {
			return nil, fmt.Errorf("wizard.Questions: %s: %w", c.name, err)
		}
		questions = append(questions, Question{Name: c.name, Why: c.why, Dial: dial, Choices: c.choices})
	}

	questions = append(questions, Question{
		Name: "妻の働き方",
		Why: "**税でも資産でも無視できない差が出る。**事業なら青色申告特別控除 65 万円が" +
			"引けるかわりに国民年金・国民健康保険を自分で払い、給与なら厚生年金と健康保険に" +
			"入る。どちらが有利かは年によって入れ替わる。",
		Choices: []Choice{
			{Label: "事業（いまの計画）", Slot: input.IncomeWifeSlot, Path: "data/controllable/income-wife.tsv"},
			{Label: "短時間", Slot: input.IncomeWifeSlot, Path: "data/controllable/scenario/income-wife-parttime.tsv"},
		},
	})

	questions = append(questions, Question{
		Name: "修繕費の伸び（建設費が物価より速い分）",
		Why: "**建設工事費デフレーター（住宅総合）は 55 年どの窓でも消費者物価より速い。**" +
			"速さのほうは窓で振れる——1970→2025 なら +0.262pt/年、1990→2025 なら +0.541pt/年、" +
			"2015→2025 なら +1.428pt/年（data/environment/README.md）。" +
			"**修繕費は今の物価で書いてあるので、実質額は選んだ窓で変わる。**",
		Choices: []Choice{
			{Label: "+0.262pt（1970→2025）", Slot: input.InflationTargetSlot, Path: "data/environment/scenario/repair-low.tsv"},
			{Label: "+0.541pt（1990→2025、いまの計画）", Slot: input.InflationTargetSlot, Path: "data/environment/inflation-target.tsv"},
			{Label: "+1.428pt（2015→2025）", Slot: input.InflationTargetSlot, Path: "data/environment/scenario/repair-high.tsv"},
		},
	})

	questions = append(questions, Question{
		Name: "金融危機（リーマンショック級）",
		Why: "**下落率 −20% は選んだ値である。**平成20年度のベンチマーク収益率は国内債券 +1.36%・" +
			"国内株式 −34.78%・外国債券 −7.17%・外国株式 −43.32%（GPIF 業務概況書）で、" +
			"何をどれだけ持つかで下がり方は変わる。**自分の資産構成で置き直すこと。**" +
			"**いつ起きるかで効きが変わる**——早いほど複利の年数が残り、取り戻せる。",
		Choices: []Choice{
			{Label: "起こさない（いまの計画）", Slot: input.FinancialCrisisSlot, Path: "data/environment/financial-crisis.tsv"},
			{Label: "2030 年に起こす", Slot: input.FinancialCrisisSlot, Path: "data/environment/scenario/crisis-2030.tsv"},
			{Label: "2040 年に起こす", Slot: input.FinancialCrisisSlot, Path: "data/environment/scenario/crisis-2040.tsv"},
			{Label: "2050 年に起こす", Slot: input.FinancialCrisisSlot, Path: "data/environment/scenario/crisis-2050.tsv"},
		},
	})
	return questions, nil
}

func WrittenPath(in *plan.Input, q Question) string {
	if len(q.Choices) == 0 {
		return ""
	}
	return in.SlotPaths()[q.Choices[0].Slot]
}

func Written(in *plan.Input, q Question) (breakeven.Setting, error) {
	table, err := in.Table(q.Dial.Slot)
	if err != nil {
		return breakeven.Setting{}, fmt.Errorf("wizard.Written: %s: %w", q.Name, err)
	}
	setting, err := q.Dial.Written(table)
	if err != nil {
		return breakeven.Setting{}, fmt.Errorf("wizard.Written: %s: %w", q.Name, err)
	}
	return setting, nil
}

func Ask(in *plan.Input, q Question) ([]breakeven.Outcome, error) {
	if q.AnsweredByTable() {
		return askTable(in, q)
	}

	settings := make([]breakeven.Setting, 0, len(q.Choices))
	for _, c := range q.Choices {
		settings = append(settings, c.Setting)
	}

	swept, err := breakeven.Sweep(in, q.Dial, settings)
	if err != nil {
		return nil, fmt.Errorf("wizard.Ask: %s: %w", q.Name, err)
	}
	return swept.Outcomes, nil
}

func askTable(in *plan.Input, q Question) ([]breakeven.Outcome, error) {
	out := make([]breakeven.Outcome, 0, len(q.Choices))
	for _, c := range q.Choices {
		table, err := tsv.ReadFile(filepath.Join(in.Root(), filepath.FromSlash(c.Path)))
		if err != nil {
			return nil, fmt.Errorf("wizard.Ask: %s: %s: %w", q.Name, c.Path, err)
		}
		built, err := in.With(c.Slot, table).Build()
		if err != nil {
			return nil, fmt.Errorf("wizard.Ask: %s: %s: %w", q.Name, c.Label, err)
		}
		outcome, err := built.Outcome()
		if err != nil {
			return nil, fmt.Errorf("wizard.Ask: %s: %s: %w", q.Name, c.Label, err)
		}
		out = append(out, breakeven.Outcome{Outcome: outcome})
	}
	return out, nil
}
