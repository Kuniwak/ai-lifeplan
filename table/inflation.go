package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/wording"
)

func InflationRatesFrom(tables map[tsv.Slot]*tsv.Table, from, to date.Year) (relation.Table[money.Rate], error) {
	return readRateStep(tables[input.InflationSlot], input.InflationSlot,
		input.InflationRateColumn, from, to)
}

func PriceLevelsFrom(tables map[tsv.Slot]*tsv.Table, from, to date.Year) (relation.Table[money.Factor], error) {
	rates, err := InflationRatesFrom(tables, from, to)
	if err != nil {
		return relation.Table[money.Factor]{}, err
	}
	return PriceLevels(rates), nil
}

func RealWageGrowthFrom(tables map[tsv.Slot]*tsv.Table, from, to date.Year) (relation.Table[money.Rate], error) {
	return readRateStep(tables[input.RealWageGrowthSlot], input.RealWageGrowthSlot,
		input.RealWageGrowthColumn, from, to)
}

func WageLevelsFrom(tables map[tsv.Slot]*tsv.Table, from, to date.Year) (relation.Table[money.Factor], error) {
	rates, err := RealWageGrowthFrom(tables, from, to)
	if err != nil {
		return relation.Table[money.Factor]{}, err
	}
	return PriceLevels(rates), nil
}

func CostGrowthFrom(tables map[tsv.Slot]*tsv.Table, from, to date.Year) (law.CostGrowth, error) {
	medical, err := readRateStep(tables[input.CostGrowthSlot], input.CostGrowthSlot,
		input.MedicalCostGrowthColumn, from, to)
	if err != nil {
		return law.CostGrowth{}, err
	}
	care, err := readRateStep(tables[input.CostGrowthSlot], input.CostGrowthSlot,
		input.NursingCareCostGrowthColumn, from, to)
	if err != nil {
		return law.CostGrowth{}, err
	}
	premium, err := readRateStep(tables[input.CostGrowthSlot], input.CostGrowthSlot,
		input.NursingCarePremiumGrowthColumn, from, to)
	if err != nil {
		return law.CostGrowth{}, err
	}
	return law.CostGrowth{
		Medical:     law.GrowingBy(medical),
		Care:        law.GrowingBy(care),
		CarePremium: law.GrowingBy(premium),
	}, nil
}

type PensionLevel struct {
	Basic, Proportional money.Rate
}

func PensionLevelAt(tables map[tsv.Slot]*tsv.Table, start date.Year) (PensionLevel, error) {
	slot := input.PensionLevelSlot

	r, err := tsv.NewReader(tables[slot], slot,
		input.YearColumn, input.PensionBasicLevelColumn, input.PensionProportionalLevelColumn)
	if err != nil {
		return PensionLevel{}, err
	}

	for row := range r.Rows() {
		year, err := r.Year(row, input.YearColumn)
		if err != nil {
			return PensionLevel{}, err
		}
		if year != start {
			continue
		}
		basic, err := r.Percent(row, input.PensionBasicLevelColumn)
		if err != nil {
			return PensionLevel{}, err
		}
		proportional, err := r.Percent(row, input.PensionProportionalLevelColumn)
		if err != nil {
			return PensionLevel{}, err
		}
		return PensionLevel{Basic: basic, Proportional: proportional}, nil
	}

	return PensionLevel{}, fmt.Errorf(
		"table.PensionLevelAt: %s に %d 年の行が無い。年金の受け取りが始まる年の水準を書く", slot, start)
}

func NominalReturns(real, prices relation.Table[money.Rate]) relation.Table[money.Rate] {
	return relation.Join(real, prices, func(_ date.Year, r, p money.Rate) money.Rate {
		return r.Compound(p)
	})
}

func InflationRatiosFrom(tables map[tsv.Slot]*tsv.Table) (map[input.PricedItem]money.PriceMove, error) {
	slot := input.InflationTargetSlot

	r, err := tsv.NewReader(tables[slot], slot,
		input.PricedItemColumn, input.InflationRatioColumn)
	if err != nil {
		return nil, err
	}

	ratios := make(map[input.PricedItem]money.PriceMove, r.Rows())
	for row := range r.Rows() {
		item := input.PricedItem(r.Field(row, input.PricedItemColumn))
		if _, twice := ratios[item]; twice {
			return nil, wording.DuplicateKeyError(fmt.Sprintf("table: %s: row %d", slot, row+1),
				string(input.PricedItemColumn), wording.Name(item), "which move grows that item")
		}

		ratio, err := r.PriceMove(row, input.InflationRatioColumn)
		if err != nil {
			return nil, err
		}
		if !ratio.IsDifference() && ratio.Rate().Cmp(input.InflationRatioFloor) < 0 {
			return nil, fmt.Errorf("table: %s: %d 行目: %q の %s が %s である。%s を下回っている",
				slot, row+1, item, input.InflationRatioColumn,
				ratio.Rate().Percent(), input.InflationRatioFloor.Percent())
		}

		ratios[item] = ratio
	}

	for _, item := range input.PricedItems {
		if _, answered := ratios[item]; !answered {
			return nil, fmt.Errorf("table: %s: %q について誰も答えていない", slot, item)
		}
	}
	if len(ratios) != len(input.PricedItems) {
		return nil, fmt.Errorf("table: %s: 計画に場所の無い項目が書かれている", slot)
	}
	return ratios, nil
}

func PriceLevelsByItem(
	rates relation.Table[money.Rate], ratios map[input.PricedItem]money.PriceMove,
) (map[input.PricedItem]relation.Table[money.Factor], error) {
	levels := make(map[input.PricedItem]relation.Table[money.Factor], len(input.PricedItems))
	for _, item := range input.PricedItems {
		ratio, answered := ratios[item]
		if !answered {
			return nil, fmt.Errorf(
				"table.PriceLevelsByItem: %q が物価とともに動くかどうかを誰も答えていない", item)
		}
		levels[item] = PriceLevels(relation.Map(rates,
			func(_ date.Year, r money.Rate) money.Rate { return ratio.Applied(r) }))
	}
	return levels, nil
}

func PriceLevelsByItemFrom(
	tables map[tsv.Slot]*tsv.Table, from, to date.Year,
) (map[input.PricedItem]relation.Table[money.Factor], error) {
	rates, err := InflationRatesFrom(tables, from, to)
	if err != nil {
		return nil, err
	}
	ratios, err := InflationRatiosFrom(tables)
	if err != nil {
		return nil, err
	}
	return PriceLevelsByItem(rates, ratios)
}

func InflateItem(
	levels map[input.PricedItem]relation.Table[money.Factor],
	item input.PricedItem,
	written relation.Table[money.Yen],
) (relation.Table[money.Yen], error) {
	byYear, answered := levels[item]
	if !answered {
		return relation.Table[money.Yen]{}, fmt.Errorf(
			"table.InflateItem: %q が物価とともに動くかどうかを誰も答えていない", item)
	}
	out := make([]relation.Row[money.Yen], 0, len(written.Years()))
	for _, row := range written.Rows() {
		level, ok := byYear.At(row.Year)
		if !ok {
			return relation.Table[money.Yen]{}, fmt.Errorf(
				"table.InflateItem: %d の %q の物価が分からない", row.Year, item)
		}
		out = append(out, relation.Row[money.Yen]{Year: row.Year, Value: level.Apply(row.Value)})
	}
	return relation.New(out), nil
}

func UniformInflationRatios(ratio money.Rate) map[input.PricedItem]money.PriceMove {
	ratios := make(map[input.PricedItem]money.PriceMove, len(input.PricedItems))
	for _, item := range input.PricedItems {
		ratios[item] = money.RatioMove(ratio)
	}
	return ratios
}

func PriceLevels(rates relation.Table[money.Rate]) relation.Table[money.Factor] {
	years := rates.Years()
	rows := make([]relation.Row[money.Factor], 0, len(years))

	level := money.NoInflation()
	for i, y := range years {
		rate, _ := rates.At(y)
		if i > 0 {
			level = level.After(rate)
		}
		rows = append(rows, relation.Row[money.Factor]{Year: y, Value: level})
	}

	return relation.New(rows)
}
