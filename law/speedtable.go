package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/money"
)

type SpeedTableStep struct {
	Upto money.Yen

	Rate int64
	Add  money.Yen
}

type SpeedTable struct {
	steps []SpeedTableStep
}

func NewSpeedTable(steps ...SpeedTableStep) SpeedTable {
	if len(steps) == 0 {
		panic("law.NewSpeedTable: no rows, so there is no amount it could answer for")
	}
	if last := steps[len(steps)-1]; last.Upto != 0 {
		panic(fmt.Sprintf(
			"law.NewSpeedTable: the last row carries an upper edge of %d, which nothing would read; the last row is the one reached by falling off the end",
			last.Upto))
	}
	for i := 1; i < len(steps)-1; i++ {
		if steps[i].Upto <= steps[i-1].Upto {
			panic(fmt.Sprintf(
				"law.NewSpeedTable: row %d ends at %d, which is not above row %d's %d, so one of the two can never be reached",
				i+1, steps[i].Upto, i, steps[i-1].Upto))
		}
	}
	return SpeedTable{steps: steps}
}

func (t SpeedTable) At(amount money.Yen) money.Yen {
	if len(t.steps) == 0 {
		panic("law.SpeedTable.At: this was not built by law.NewSpeedTable, so it has no rows")
	}
	for _, step := range t.steps[:len(t.steps)-1] {
		if amount <= step.Upto {
			return amount.Mul(money.NewPercent(step.Rate), money.Truncate) + step.Add
		}
	}
	last := t.steps[len(t.steps)-1]
	return amount.Mul(money.NewPercent(last.Rate), money.Truncate) + last.Add
}

func (t SpeedTable) Least() money.Yen {
	return t.steps[0].Add
}

func (t SpeedTable) Most() (money.Yen, bool) {
	last := t.steps[len(t.steps)-1]
	return last.Add, last.Rate == 0
}
