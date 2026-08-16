package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type PropertyTaxRow struct {
	law.PropertyTaxBill
}

type PropertyTaxInput struct {
	Calendar relation.Table[CalendarRow]

	BuiltIn date.Year

	LandValue money.Yen

	HouseBaseAt money.Yen
	AssessedIn  date.Year

	ConstructionLevel relation.Table[money.Factor]

	Depreciation law.DepreciationRateTable
	Table        law.PropertyTaxTable
}

func PropertyTaxTable(in PropertyTaxInput) (relation.Table[PropertyTaxRow], error) {
	var empty relation.Table[PropertyTaxRow]

	anchor := in.Depreciation.Rate(int(baseYearOf(in.AssessedIn) - in.BuiltIn))

	var previous money.Yen

	years := in.Calendar.Years()
	rows := make([]relation.Row[PropertyTaxRow], 0, len(years))

	for _, y := range years {
		var row PropertyTaxRow
		if y < in.BuiltIn {
			rows = append(rows, relation.Row[PropertyTaxRow]{Year: y, Value: row})
			continue
		}

		elapsed := int(baseYearOf(y) - in.BuiltIn)
		houseBase := in.HouseBaseAt.Mul(in.Depreciation.Rate(elapsed), money.Truncate).
			Mul(reciprocal(anchor), money.Truncate).
			Mul(rebuildFactor(in.ConstructionLevel, in.AssessedIn, y), money.Truncate)

		if previous > 0 && houseBase > previous {
			houseBase = previous
		}
		previous = houseBase

		bill, err := in.Table.Bill(in.LandValue, houseBase, elapsed, y)
		if err != nil {
			return empty, fmt.Errorf("table.PropertyTaxTable: %d: %w", y, err)
		}
		row.PropertyTaxBill = bill

		rows = append(rows, relation.Row[PropertyTaxRow]{Year: y, Value: row})
	}

	return relation.New(rows), nil
}

func reciprocal(r money.Rate) money.Rate {
	return money.NewRate(r.Den(), r.Num())
}

const PropertyBaseYear date.Year = 2024

const PropertyBaseYearCycle = 3

const PropertyRebuildLag date.Year = 2

func baseYearOf(y date.Year) date.Year {
	if y < PropertyBaseYear {
		return PropertyBaseYear
	}
	return PropertyBaseYear + (y-PropertyBaseYear)/PropertyBaseYearCycle*PropertyBaseYearCycle
}

func rebuildFactor(level relation.Table[money.Factor], from, to date.Year) money.Rate {
	const unchanged = 1_000_000

	at := func(y date.Year) (int64, bool) {
		f, ok := level.At(baseYearOf(y) - PropertyRebuildLag)
		if !ok {
			return 0, false
		}
		return int64(f.Apply(unchanged)), true
	}
	was, ok := at(from)
	if !ok || was == 0 {
		return money.NewRate(1, 1)
	}
	now, ok := at(to)
	if !ok {
		return money.NewRate(1, 1)
	}
	return money.NewRate(now, was)
}
