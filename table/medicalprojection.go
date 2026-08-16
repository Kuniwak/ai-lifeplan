package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	CompressionMeasuredFrom = 10
	CompressionMeasuredTo   = 65
)

var StatutoryShareWhereMeasured = money.NewRate(3, 10)

type AgeCurve struct {
	cost, outOfPocket relation.Bands[int, money.Yen]
}

func AgeCurveFrom(tables map[tsv.Slot]*tsv.Table) (AgeCurve, error) {
	var empty AgeCurve

	table := tables[input.MedicalCostByAgeSlot]
	if table == nil {
		return empty, fmt.Errorf("table.AgeCurveFrom: %s の表がありません", input.MedicalCostByAgeSlot)
	}
	r, err := tsv.NewReader(table, input.MedicalCostByAgeSlot,
		input.AgeFromColumn, input.MedicalCostAtAge, input.OutOfPocketAtAge)
	if err != nil {
		return empty, err
	}

	cost := make([]relation.Band[int, money.Yen], 0, len(table.Rows))
	pocket := make([]relation.Band[int, money.Yen], 0, len(table.Rows))
	for row := range r.Rows() {
		from, err := r.Count(row, input.AgeFromColumn)
		if err != nil {
			return empty, err
		}
		at, err := r.Yen(row, input.MedicalCostAtAge)
		if err != nil {
			return empty, err
		}
		paid, err := r.Yen(row, input.OutOfPocketAtAge)
		if err != nil {
			return empty, err
		}
		cost = append(cost, relation.Band[int, money.Yen]{Lower: from, Value: at})
		pocket = append(pocket, relation.Band[int, money.Yen]{Lower: from, Value: paid})
	}

	curve := AgeCurve{cost: relation.NewBands(cost), outOfPocket: relation.NewBands(pocket)}
	lowest, ok := curve.cost.Min()
	if !ok {
		return empty, fmt.Errorf("table.AgeCurveFrom: %s に行がありません", input.MedicalCostByAgeSlot)
	}
	if lowest != 0 {
		return empty, fmt.Errorf(
			"table.AgeCurveFrom: %s の最初の下限年齢が %d です。0 歳が引けません",
			input.MedicalCostByAgeSlot, lowest)
	}
	return curve, nil
}

func (a AgeCurve) CostAt(age int) money.Yen { return a.cost.Lookup(age) }

func (a AgeCurve) OutOfPocketAt(age int) money.Yen { return a.outOfPocket.Lookup(age) }

func (a AgeCurve) PaidAt(age int, share money.Rate) money.Yen {
	statutory := a.CostAt(age).Mul(share, money.HalfUp)

	at := min(max(age, CompressionMeasuredFrom), CompressionMeasuredTo)
	return money.ShareOf(statutory, a.OutOfPocketAt(at),
		a.CostAt(at).Mul(StatutoryShareWhereMeasured, money.HalfUp))
}

type MedicalProjectionInput struct {
	Calendar relation.Table[CalendarRow]

	Recorded   relation.Table[money.Yen]
	RecordedTo date.Year

	Curve AgeCurve

	Copay CopayShare

	Projected []PersonName
}

type CopayShare func(name PersonName, year date.Year, month int, born date.Date) money.Rate

func FlatCopay(share money.Rate) CopayShare {
	return func(PersonName, date.Year, int, date.Date) money.Rate { return share }
}

const MedicalProjectionRounding money.Yen = 1_000

func MedicalProjection(in MedicalProjectionInput) (relation.Table[money.Yen], error) {
	var empty relation.Table[money.Yen]

	if in.Copay == nil {
		return empty, fmt.Errorf(
			"table.MedicalProjection: 窓口負担割合が渡されていません。据え置くなら FlatCopay で言うこと")
	}

	from := in.RecordedTo
	base, ok := in.Recorded.At(from)
	if !ok {
		return empty, fmt.Errorf(
			"table.MedicalProjection: 記録の最後の年としている %d の医療費がありません", from)
	}

	divisor, err := in.outOfPocket(from)
	if err != nil {
		return empty, err
	}
	if divisor <= 0 {
		return empty, fmt.Errorf(
			"table.MedicalProjection: %d 年の窓口負担が %d 円である。割れない", from, divisor)
	}

	years := in.Calendar.Years()
	rows := make([]relation.Row[money.Yen], 0, len(years))
	for _, y := range years {
		if y <= from {
			recorded, ok := in.Recorded.At(y)
			if !ok {
				return empty, fmt.Errorf("table.MedicalProjection: %d 年の医療費の記録がありません", y)
			}
			rows = append(rows, relation.Row[money.Yen]{Year: y, Value: recorded})
			continue
		}

		paid, err := in.outOfPocket(y)
		if err != nil {
			return empty, err
		}
		projected := money.ShareOf(base, paid, divisor)
		rows = append(rows, relation.Row[money.Yen]{
			Year:  y,
			Value: money.Yen(money.HalfUp(int64(projected), int64(MedicalProjectionRounding))) * MedicalProjectionRounding,
		})
	}
	return relation.New(rows), nil
}

func (in MedicalProjectionInput) outOfPocket(year date.Year) (money.Yen, error) {
	calendar, ok := in.Calendar.At(year)
	if !ok {
		return 0, fmt.Errorf("table.MedicalProjection: %d 年が暦にありません", year)
	}

	var total money.Yen
	for _, name := range in.Projected {
		if _, ok := calendar.AgeOf(name); !ok {
			return 0, fmt.Errorf("table.MedicalProjection: %d 年に %q の年齢がありません", year, name)
		}
		born, ok := calendar.BornOnOf(name)
		if !ok {
			return 0, fmt.Errorf("table.MedicalProjection: %d 年に %q の生年月日がありません。年末の満年齢では窓口負担の段が 1 月から効いてしまいます", year, name)
		}

		for month := 1; month <= date.MonthsAYear; month++ {
			total += in.Curve.PaidAt(born.AgeInMonth(year, month), in.Copay(name, year, month, born))
		}
	}
	return total, nil
}

func MedicalCopayFrom(
	residentTax map[PersonName]relation.Table[ResidentTaxRow],
	income map[PersonName]relation.Table[IncomeRow],
) CopayShare {
	return func(name PersonName, year date.Year, month int, born date.Date) money.Rate {
		var highest, pension money.Yen
		for who, table := range residentTax {
			row, ok := table.At(year)
			if !ok {
				panic(fmt.Sprintf(
					"table.MedicalCopayFrom: %d 年の %s の住民税がありません。所得を 0 と読めば最も安い割合になります", year, who))
			}
			highest = max(highest, row.Taxable)

			earned, ok := income[who].At(year)
			if !ok {
				panic(fmt.Sprintf(
					"table.MedicalCopayFrom: %d 年の %s の収入がありません。年金を 0 と読めば最も安い割合になります", year, who))
			}
			pension += earned.PensionReceived
		}
		return law.CopaymentShareInMonth(year, month, born, law.CopaymentIncome{
			HighestTaxableIncome:  highest,
			PensionAndOtherIncome: pension,
			Revenue:               pension,
		})
	}
}

func LastRecordedMedicalYear(tables map[tsv.Slot]*tsv.Table) (date.Year, error) {
	table := tables[input.MedicalExpenseSlot]
	if table == nil {
		return 0, fmt.Errorf("table.LastRecordedMedicalYear: %s の表がありません", input.MedicalExpenseSlot)
	}
	at, ok := table.ColumnIndex(input.YearColumn)
	if !ok {
		return 0, fmt.Errorf("table.LastRecordedMedicalYear: %s に %q の列がありません",
			input.MedicalExpenseSlot, input.YearColumn)
	}

	var last date.Year
	for row, fields := range table.Rows {
		year, err := date.ParseYear(fields[at])
		if err != nil {
			return 0, fmt.Errorf("table.LastRecordedMedicalYear: row %d: %w", row+1, err)
		}
		last = max(last, year)
	}
	if last == 0 {
		return 0, fmt.Errorf("table.LastRecordedMedicalYear: %s に行がありません", input.MedicalExpenseSlot)
	}
	return last, nil
}
