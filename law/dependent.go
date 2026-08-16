package law

import (
	"github.com/Kuniwak/lifeplan/money"

	"github.com/Kuniwak/lifeplan/date"
)

const ElderlyAge = 70

var taxpayerTiers = NewAmendedFrom("配偶者控除等の納税者本人の所得段階", 2018,
	YearRow[taxpayerTierRow]{FromYear: 2018, Value: taxpayerTierRow{
		Tops: [3]money.Yen{9_000_000, 9_500_000, 10_000_000},
	}},
)

type taxpayerTierRow struct {
	Tops [3]money.Yen
}

type Deduction struct {
	IncomeTax money.Yen
	Resident  money.Yen
}

func taxpayerTier(taxpayerIncome money.Yen, incomeYear date.Year) int {
	tops := taxpayerTiers.At(incomeYear).Tops
	for i, top := range tops {
		if taxpayerIncome <= top {
			return i
		}
	}
	return -1
}

var spouseDeductions = NewAmendedFrom("配偶者控除", 2018,
	YearRow[spouseDeductionRow]{FromYear: 2018, Value: spouseDeductionRow{
		General: [3]Deduction{
			{IncomeTax: 380_000, Resident: 330_000},
			{IncomeTax: 260_000, Resident: 220_000},
			{IncomeTax: 130_000, Resident: 110_000},
		},
		Elderly: [3]Deduction{
			{IncomeTax: 480_000, Resident: 380_000},
			{IncomeTax: 320_000, Resident: 260_000},
			{IncomeTax: 160_000, Resident: 130_000},
		},
	}},
)

type spouseDeductionRow struct {
	General, Elderly [3]Deduction
}

func SpouseDeduction(spouseIncome, taxpayerIncome money.Yen, spouseAge int, ceiling money.Yen, incomeYear date.Year) Deduction {
	tier := taxpayerTier(taxpayerIncome, incomeYear)
	if tier < 0 || spouseIncome > ceiling {
		return Deduction{}
	}
	row := spouseDeductions.At(incomeYear)
	if spouseAge >= ElderlyAge {
		return row.Elderly[tier]
	}
	return row.General[tier]
}

var spouseSpecialDeductions = [9][3]Deduction{
	{{380_000, 330_000}, {260_000, 220_000}, {130_000, 110_000}},
	{{360_000, 330_000}, {240_000, 220_000}, {120_000, 110_000}},
	{{310_000, 310_000}, {210_000, 210_000}, {110_000, 110_000}},
	{{260_000, 260_000}, {180_000, 180_000}, {90_000, 90_000}},
	{{210_000, 210_000}, {140_000, 140_000}, {70_000, 70_000}},
	{{160_000, 160_000}, {110_000, 110_000}, {60_000, 60_000}},
	{{110_000, 110_000}, {80_000, 80_000}, {40_000, 40_000}},
	{{60_000, 60_000}, {40_000, 40_000}, {20_000, 20_000}},
	{{30_000, 30_000}, {20_000, 20_000}, {10_000, 10_000}},
}

var spouseSpecial = NewAmendedFrom("配偶者特別控除", 2018,
	YearRow[spouseSpecialTable]{FromYear: 2018, Value: spouseSpecialTable{
		Bands: [9]money.Yen{
			850_000, 900_000, 950_000, 1_000_000,
			1_050_000, 1_100_000, 1_150_000, 1_200_000, 1_230_000,
		},
		Deductions: spouseSpecialDeductions,
	}},
	YearRow[spouseSpecialTable]{FromYear: 2020, Value: spouseSpecialTable{
		Bands: [9]money.Yen{
			950_000, 1_000_000, 1_050_000, 1_100_000,
			1_150_000, 1_200_000, 1_250_000, 1_300_000, 1_330_000,
		},
		Deductions: spouseSpecialDeductions,
	}},
)

type spouseSpecialTable struct {
	Bands [9]money.Yen

	Deductions [9][3]Deduction
}

func SpouseDeductionRecordFloors() []RecordFloor {
	return []RecordFloor{
		{taxpayerTiers.name, taxpayerTiers.FirstWrittenYear},
		{spouseDeductions.name, spouseDeductions.FirstWrittenYear},
		{spouseSpecial.name, spouseSpecial.FirstWrittenYear},
		{spouseGaps.name, spouseGaps.FirstWrittenYear},
		{spouseSpecialGap.name, spouseSpecialGap.FirstWrittenYear},
	}
}

func SpouseSpecialIncomeCeilingAt(incomeYear date.Year) money.Yen {
	bands := spouseSpecial.At(incomeYear).Bands
	return bands[len(bands)-1]
}

func SpouseSpecialDeduction(spouseIncome, taxpayerIncome money.Yen, ceiling money.Yen, incomeYear date.Year) Deduction {
	tier := taxpayerTier(taxpayerIncome, incomeYear)
	if tier < 0 || spouseIncome <= ceiling {
		return Deduction{}
	}

	table := spouseSpecial.At(incomeYear)
	for i, band := range table.Bands {
		if spouseIncome <= band {
			return table.Deductions[i][tier]
		}
	}
	return Deduction{}
}

func SpouseDeductionsOf(spouseIncome, taxpayerIncome money.Yen, spouseAge int, ceiling money.Yen, incomeYear date.Year) (Deduction, money.Yen) {
	return SpouseDeductionTotal(spouseIncome, taxpayerIncome, spouseAge, ceiling, incomeYear),
		SpouseHumanDeductionGap(spouseIncome, taxpayerIncome, spouseAge, ceiling, incomeYear)
}

func SpouseDeductionTotal(spouseIncome, taxpayerIncome money.Yen, spouseAge int, ceiling money.Yen, incomeYear date.Year) Deduction {
	if d := SpouseDeduction(spouseIncome, taxpayerIncome, spouseAge, ceiling, incomeYear); d != (Deduction{}) {
		return d
	}
	return SpouseSpecialDeduction(spouseIncome, taxpayerIncome, ceiling, incomeYear)
}

var spouseGaps = NewAmendedFrom("控除対象配偶者の人的控除の差", 2018,
	YearRow[spouseGapRow]{FromYear: 2018, Value: spouseGapRow{
		General: [3]money.Yen{50_000, 40_000, 20_000},
		Elderly: [3]money.Yen{100_000, 60_000, 30_000},
	}},
)

type spouseGapRow struct {
	General, Elderly [3]money.Yen
}

var (
	spouseSpecialGapFull   = [3]money.Yen{50_000, 40_000, 20_000}
	spouseSpecialGapHalved = [3]money.Yen{30_000, 20_000, 10_000}
)

var spouseSpecialGap = NewAmendedFrom("配偶者特別控除の人的控除の差", 2018,
	YearRow[spouseSpecialGapRow]{FromYear: 2018, Value: spouseSpecialGapRow{
		Top: 450_000, Halved: 400_000,
		Full: spouseSpecialGapFull, Small: spouseSpecialGapHalved,
	}},
	YearRow[spouseSpecialGapRow]{FromYear: 2020, Value: spouseSpecialGapRow{
		Top: 550_000, Halved: 500_000,
		Full: spouseSpecialGapFull, Small: spouseSpecialGapHalved,
	}},
)

type spouseSpecialGapRow struct {
	Top money.Yen

	Halved money.Yen

	Full, Small [3]money.Yen
}

func SpouseHumanDeductionGap(spouseIncome, taxpayerIncome money.Yen, spouseAge int, ceiling money.Yen, incomeYear date.Year) money.Yen {
	tier := taxpayerTier(taxpayerIncome, incomeYear)
	if tier < 0 {
		return 0
	}

	if spouseIncome <= ceiling {
		row := spouseGaps.At(incomeYear)
		if spouseAge >= ElderlyAge {
			return row.Elderly[tier]
		}
		return row.General[tier]
	}

	row := spouseSpecialGap.At(incomeYear)
	switch {
	case spouseIncome >= row.Top:
		return 0
	case spouseIncome >= row.Halved:
		return row.Small[tier]
	default:
		return row.Full[tier]
	}
}

const (
	DependentMinimumAge = 16

	SpecificDependentFrom  = 19
	SpecificDependentUntil = 23
)

var (
	generalDependent  = Deduction{IncomeTax: 380_000, Resident: 330_000}
	specificDependent = Deduction{IncomeTax: 630_000, Resident: 450_000}
	elderlyDependent  = Deduction{IncomeTax: 480_000, Resident: 380_000}

	elderlyLivingInDependent = Deduction{IncomeTax: 580_000, Resident: 450_000}
)

func DependentDeduction(age int, livingWithTaxpayer bool) Deduction {
	switch {
	case age < DependentMinimumAge:
		return Deduction{}
	case age >= SpecificDependentFrom && age < SpecificDependentUntil:
		return specificDependent
	case age >= ElderlyAge && livingWithTaxpayer:
		return elderlyLivingInDependent
	case age >= ElderlyAge:
		return elderlyDependent
	default:
		return generalDependent
	}
}
