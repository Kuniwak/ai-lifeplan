package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type CostGrowthCurve struct {
	byYear relation.Table[money.Rate]

	stated bool
}

func GrowingBy(byYear relation.Table[money.Rate]) CostGrowthCurve {
	return CostGrowthCurve{byYear: byYear, stated: true}
}

func GrowingSteadilyBy(from, to date.Year, rate money.Rate) CostGrowthCurve {
	return GrowingBy(relation.Constant(relation.Span(from, to), rate))
}

func NoCostGrowthCurve() CostGrowthCurve { return CostGrowthCurve{stated: true} }

func (g CostGrowthCurve) answered() bool { return g.stated }

func (g CostGrowthCurve) assertStated(refusal func() string) {
	if !g.answered() {
		panic(refusal())
	}
}

func (g CostGrowthCurve) ByYear() relation.Table[money.Rate] { return g.byYear }

func (g CostGrowthCurve) Since(lastWritten, year date.Year) money.Factor {
	factor := money.NoInflation()
	for y := lastWritten + 1; y <= year; y++ {
		rate, ok := g.byYear.At(y)
		if !ok {
			continue
		}
		factor = factor.After(rate)
	}
	return factor
}

func (g CostGrowthCurve) AssertCovers(what string, lastWritten, to date.Year) error {
	g.assertStated(func() string {
		return "law.CostGrowthCurve.AssertCovers: " + what +
			" の伸びを誰も答えていない。伸ばさないなら law.NoCostGrowth() を表に渡すこと"
	})

	if len(g.byYear.Years()) == 0 {
		return nil
	}
	for y := lastWritten + 1; y <= to; y++ {
		if _, ok := g.byYear.At(y); !ok {
			return fmt.Errorf(
				"law: %s は %d 年までしか書かれていないので %d 年から伸ばすが、医療費上昇率に %d 年の行が無い",
				what, lastWritten, lastWritten+1, y)
		}
	}
	return nil
}

func (g CostGrowthCurve) GrowPremium(premium money.Yen, lastWritten LastWrittenYear, year date.Year) money.Yen {
	g.assertStated(func() string {
		return "law.CostGrowthCurve.GrowPremium: 医療費・介護費の伸びを誰も答えていない。" +
			"伸ばさないなら law.NoCostGrowth() を表に渡すこと"
	})

	last, ok := lastWritten()
	if !ok {
		return premium
	}
	return g.Since(last, year).Apply(premium)
}

type LastWrittenYear func() (date.Year, bool)

type FirstWrittenYear func() (date.Year, bool)

func AssertRecordReaches(what string, firstWritten FirstWrittenYear, firstAsked date.Year) error {
	first, ok := firstWritten()
	if !ok {
		return nil
	}
	if firstAsked < first {
		return fmt.Errorf(
			"law: %s が書かれているのは %d 年からで、この計画はそれを %d 年から引く。"+
				"%d 年から %d 年までは誰も値を書いておらず、最初の行の値をその年に立てるのは"+
				"誰も選んでいない数字を使うことになる",
			what, first, firstAsked, firstAsked, first-1)
	}
	return nil
}

func (t YearRateTable) LastWrittenYear() (date.Year, bool) { return t.bands.Max() }

func (t YearRateTable) FirstWrittenYear() (date.Year, bool) { return t.bands.Min() }

func (t YearYenTable) LastWrittenYear() (date.Year, bool) { return t.bands.Max() }

func (t YearYenTable) FirstWrittenYear() (date.Year, bool) { return t.bands.Min() }

func (t YearTable[V]) LastWrittenYear() (date.Year, bool) { return t.bands.Max() }

func (t YearTable[V]) FirstWrittenYear() (date.Year, bool) { return t.bands.Min() }

type CostGrowth struct {
	Medical CostGrowthCurve

	Care CostGrowthCurve

	CarePremium CostGrowthCurve
}

func (g CostGrowth) AssertStated() {
	if !g.Medical.answered() {
		panic("law.CostGrowth: 医療費の伸びを誰も答えていない。" +
			"伸ばさないなら Medical に law.NoCostGrowthCurve() を書くこと")
	}
	if !g.Care.answered() {
		panic("law.CostGrowth: 介護費の伸びを誰も答えていない。" +
			"伸ばさないなら Care に law.NoCostGrowthCurve() を書くこと")
	}
	if !g.CarePremium.answered() {
		panic("law.CostGrowth: 介護保険料の伸びを誰も答えていない。" +
			"伸ばさないなら CarePremium に law.NoCostGrowthCurve() を書くこと")
	}
}

func NoCostGrowth() CostGrowth {
	return CostGrowth{Medical: NoCostGrowthCurve(), Care: NoCostGrowthCurve(), CarePremium: NoCostGrowthCurve()}
}

func GrowingEverythingBy(g CostGrowthCurve) CostGrowth {
	return CostGrowth{Medical: g, Care: g, CarePremium: g}
}
