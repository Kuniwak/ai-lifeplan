package table

import (
	"strings"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

type MeasureName string

const (
	NoMeasure MeasureName = ""

	ReverseMortgage MeasureName = "リバースモーゲージ"

	SellAndRent MeasureName = "売却して賃貸"
)

type Measure struct {
	Name MeasureName

	FromAge int

	ProceedRate money.Rate

	Interest money.Rate

	RentMonthly money.Yen

	GivesUpHome bool
}

func (m Measure) Proceeds(collateral money.Yen) money.Yen {
	return collateral.Mul(m.ProceedRate, money.Truncate)
}

func (m Measure) InterestOn(borrowed money.Yen) money.Yen {
	return borrowed.Mul(m.Interest, money.Truncate)
}

func (m Measure) EarliestYear(born date.Date, loanClearedIn date.Year) date.Year {
	oldEnough := born.ReachesAge(m.FromAge).Year
	if loanClearedIn > oldEnough {
		return loanClearedIn
	}
	return oldEnough
}

func Measures(t *tsv.Table) (map[MeasureName]Measure, error) {
	out := make(map[MeasureName]Measure, 2)
	if t == nil || len(t.Rows) == 0 {
		return out, nil
	}
	read, err := tsv.NewReader(t, input.LastResortSlot,
		input.MeasureColumn, input.MeasureFromAge, input.MeasureProceedRate,
		input.MeasureInterest, input.MeasureRentMonthly, input.MeasureGivesUpHome)
	if err != nil {
		return nil, err
	}
	for i := range t.Rows {
		name := MeasureName(strings.TrimSpace(read.Field(i, input.MeasureColumn)))
		if name != ReverseMortgage && name != SellAndRent {
			return nil, read.Errorf(i, input.MeasureColumn,
				"%q は知らない手段である。%q か %q を書くこと", name, ReverseMortgage, SellAndRent)
		}
		if _, twice := out[name]; twice {
			return nil, read.Errorf(i, input.MeasureColumn, "%q が二度ある", name)
		}

		fromAge, err := read.Count(i, input.MeasureFromAge)
		if err != nil {
			return nil, err
		}
		proceeds, err := read.Percent(i, input.MeasureProceedRate)
		if err != nil {
			return nil, err
		}
		interest, err := read.Percent(i, input.MeasureInterest)
		if err != nil {
			return nil, err
		}
		rent, err := read.Yen(i, input.MeasureRentMonthly)
		if err != nil {
			return nil, err
		}

		var gives bool
		switch field := strings.TrimSpace(read.Field(i, input.MeasureGivesUpHome)); field {
		case "はい":
			gives = true
		case "いいえ":
			gives = false
		default:
			return nil, read.Errorf(i, input.MeasureGivesUpHome,
				"%q は はい でも いいえ でもない", field)
		}

		out[name] = Measure{
			Name: name, FromAge: fromAge, ProceedRate: proceeds,
			Interest: interest, RentMonthly: rent, GivesUpHome: gives,
		}
	}
	return out, nil
}

type LastResort struct {
	Measure Measure

	From date.Year

	Collateral, Proceeds money.Yen

	Rent, Owning relation.Table[money.Yen]
}

func (r LastResort) Yearly(y date.Year) money.Yen {
	if !r.Measure.GivesUpHome || y < r.From {
		return 0
	}
	rent, _ := r.Rent.At(y)
	owning, _ := r.Owning.At(y)
	return rent - owning
}
