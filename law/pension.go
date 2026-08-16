package law

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

type pensionWording struct {
	FlatUntilUnder65, FlatUntilOver65 money.Yen
	FlatUnder65, FlatOver65           money.Yen

	Steps SpeedTable

	Reduction [3]money.Yen
}

var pensionTables = NewAmended("公的年金等控除",
	YearRow[pensionWording]{FromYear: 0, Value: pensionWording{
		FlatUntilUnder65: 1_300_000, FlatUnder65: 700_000,
		FlatUntilOver65: 3_300_000, FlatOver65: 1_200_000,
		Steps: NewSpeedTable(
			SpeedTableStep{Upto: 4_100_000, Rate: 25, Add: 375_000},
			SpeedTableStep{Upto: 7_700_000, Rate: 15, Add: 785_000},
			SpeedTableStep{Rate: 5, Add: 1_555_000},
		),
		Reduction: [3]money.Yen{0, 0, 0},
	}},
	YearRow[pensionWording]{FromYear: 2020, Value: pensionWording{
		FlatUntilUnder65: 1_300_000, FlatUnder65: 600_000,
		FlatUntilOver65: 3_300_000, FlatOver65: 1_100_000,
		Steps: NewSpeedTable(
			SpeedTableStep{Upto: 4_100_000, Rate: 25, Add: 275_000},
			SpeedTableStep{Upto: 7_700_000, Rate: 15, Add: 685_000},
			SpeedTableStep{Upto: 10_000_000, Rate: 5, Add: 1_455_000},
			SpeedTableStep{Add: 1_955_000},
		),
		Reduction: [3]money.Yen{0, 100_000, 200_000},
	}},
)

const PensionDeductionAge = 65

const (
	TotalIncomeTier1 money.Yen = 10_000_000
	TotalIncomeTier2 money.Yen = 20_000_000
)

func PensionIncomeDeduction(received money.Yen, age int, totalIncome money.Yen, incomeYear date.Year) money.Yen {
	table := pensionTables.At(incomeYear)
	base := pensionDeductionBefore(received, age, table)

	deduction := base - pensionReduction(totalIncome, table)

	return max(deduction, 0)
}

func pensionDeductionBefore(received money.Yen, age int, t pensionWording) money.Yen {
	flatUntil, flat := t.FlatUntilUnder65, t.FlatUnder65
	if age >= PensionDeductionAge {
		flatUntil, flat = t.FlatUntilOver65, t.FlatOver65
	}
	if received <= flatUntil {
		return flat
	}

	return t.Steps.At(received)
}

func pensionReduction(totalIncome money.Yen, t pensionWording) money.Yen {
	switch {
	case totalIncome <= TotalIncomeTier1:
		return t.Reduction[0]
	case totalIncome <= TotalIncomeTier2:
		return t.Reduction[1]
	default:
		return t.Reduction[2]
	}
}

func PensionIncome(received money.Yen, age int, totalIncome money.Yen, incomeYear date.Year) money.Yen {
	income := received - PensionIncomeDeduction(received, age, totalIncome, incomeYear)
	return max(income, 0)
}
