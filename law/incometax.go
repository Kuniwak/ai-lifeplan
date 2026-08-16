package law

import (
	"github.com/Kuniwak/lifeplan/money"

	"github.com/Kuniwak/lifeplan/date"
)

var incomeTaxBands = []struct {
	Ceiling  money.Yen
	Rate     int64
	Subtract money.Yen
}{
	{Ceiling: 1_949_000, Rate: 5, Subtract: 0},
	{Ceiling: 3_299_000, Rate: 10, Subtract: 98_000},
	{Ceiling: 6_949_000, Rate: 20, Subtract: 428_000},
	{Ceiling: 8_999_000, Rate: 23, Subtract: 636_000},
	{Ceiling: 17_999_000, Rate: 33, Subtract: 1_536_000},
	{Ceiling: 39_999_000, Rate: 40, Subtract: 2_796_000},
}

const (
	incomeTaxTopRate                   = 45
	incomeTaxTopSubtract     money.Yen = 4_796_000
	ReconstructionRatePer100           = 21
)

const ReconstructionLastYear = 2037

func IncomeTax(taxableIncome money.Yen) money.Yen {
	taxable := taxableIncome.TruncateTaxableIncome()
	if taxable <= 0 {
		return 0
	}

	for _, band := range incomeTaxBands {
		if taxable <= band.Ceiling {
			return taxable.Mul(money.NewPercent(band.Rate), money.Truncate) - band.Subtract
		}
	}
	return taxable.Mul(money.NewPercent(incomeTaxTopRate), money.Truncate) - incomeTaxTopSubtract
}

func ReconstructionSurtax(baseTax money.Yen, year date.Year) money.Yen {
	if year > ReconstructionLastYear {
		return 0
	}
	if baseTax <= 0 {
		return 0
	}
	return baseTax.Mul(money.NewRate(ReconstructionRatePer100, 1000), money.Truncate)
}

type basicDeductionBand struct {
	Ceiling money.Yen

	Deduction
}

type basicDeductionTable struct {
	Bands []basicDeductionBand

	Above Deduction
}

var noBasicDeduction = Deduction{}

var basicDeductionBands = NewAmended("基礎控除",
	YearRow[basicDeductionTable]{FromYear: 0, Value: basicDeductionTable{
		Above: Deduction{IncomeTax: 380_000, Resident: 330_000},
	}},
	YearRow[basicDeductionTable]{FromYear: 2020, Value: basicDeductionTable{
		Bands: []basicDeductionBand{
			{Ceiling: 24_000_000, Deduction: Deduction{IncomeTax: 480_000, Resident: 430_000}},
			{Ceiling: 24_500_000, Deduction: Deduction{IncomeTax: 320_000, Resident: 290_000}},
			{Ceiling: 25_000_000, Deduction: Deduction{IncomeTax: 160_000, Resident: 150_000}},
		},
		Above: noBasicDeduction,
	}},
	YearRow[basicDeductionTable]{FromYear: 2025, Value: basicDeductionTable{
		Bands: []basicDeductionBand{
			{Ceiling: 23_500_000, Deduction: Deduction{IncomeTax: 580_000, Resident: 430_000}},
			{Ceiling: 24_000_000, Deduction: Deduction{IncomeTax: 480_000, Resident: 430_000}},
			{Ceiling: 24_500_000, Deduction: Deduction{IncomeTax: 320_000, Resident: 290_000}},
			{Ceiling: 25_000_000, Deduction: Deduction{IncomeTax: 160_000, Resident: 150_000}},
		},
		Above: noBasicDeduction,
	}},
)

func BasicDeduction(totalIncome money.Yen, year date.Year) money.Yen {
	table := basicDeductionBands.At(year)
	for _, band := range table.Bands {
		if totalIncome <= band.Ceiling {
			return band.IncomeTax
		}
	}
	return table.Above.IncomeTax
}

func ResidentBasicDeduction(totalIncome money.Yen, year date.Year) money.Yen {
	table := basicDeductionBands.At(year)
	for _, band := range table.Bands {
		if totalIncome <= band.Ceiling {
			return band.Resident
		}
	}
	return table.Above.Resident
}
