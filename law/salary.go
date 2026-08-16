package law

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
)

const (
	SalaryDeductionFloorUntil2019 money.Yen = 650_000
	SalaryDeductionFloorUntil2024 money.Yen = 550_000
	SalaryDeductionFloorFrom2025  money.Yen = 650_000
)

const (
	SalaryDeductionCeilingUntil2019 money.Yen = 2_200_000
	SalaryDeductionCeilingFrom2020  money.Yen = 1_950_000
)

func SalaryDeductionBoundsAt(year date.Year) (floor, ceiling money.Yen) {
	table := salarySteps.At(year)
	most, capped := table.Most()
	if !capped {
		panic("law.SalaryDeductionBoundsAt: 給与所得控除の速算表に上限の行が無い")
	}
	return table.Least(), most
}

var salarySteps = NewAmended("給与所得控除",
	YearRow[SpeedTable]{FromYear: 0, Value: NewSpeedTable(
		SpeedTableStep{Upto: 1_625_000, Add: SalaryDeductionFloorUntil2019},
		SpeedTableStep{Upto: 1_800_000, Rate: 40},
		SpeedTableStep{Upto: 3_600_000, Rate: 30, Add: 180_000},
		SpeedTableStep{Upto: 6_600_000, Rate: 20, Add: 540_000},
		SpeedTableStep{Upto: 10_000_000, Rate: 10, Add: 1_200_000},
		SpeedTableStep{Add: SalaryDeductionCeilingUntil2019},
	)},
	YearRow[SpeedTable]{FromYear: 2020, Value: NewSpeedTable(
		SpeedTableStep{Upto: 1_625_000, Add: SalaryDeductionFloorUntil2024},
		SpeedTableStep{Upto: 1_800_000, Rate: 40, Add: -100_000},
		SpeedTableStep{Upto: 3_600_000, Rate: 30, Add: 80_000},
		SpeedTableStep{Upto: 6_600_000, Rate: 20, Add: 440_000},
		SpeedTableStep{Upto: 8_500_000, Rate: 10, Add: 1_100_000},
		SpeedTableStep{Add: SalaryDeductionCeilingFrom2020},
	)},
	YearRow[SpeedTable]{FromYear: 2025, Value: NewSpeedTable(
		SpeedTableStep{Upto: 1_900_000, Add: SalaryDeductionFloorFrom2025},
		SpeedTableStep{Upto: 3_600_000, Rate: 30, Add: 80_000},
		SpeedTableStep{Upto: 6_600_000, Rate: 20, Add: 440_000},
		SpeedTableStep{Upto: 8_500_000, Rate: 10, Add: 1_100_000},
		SpeedTableStep{Add: SalaryDeductionCeilingFrom2020},
	)},
)

func SalaryIncomeDeduction(salary money.Yen, year date.Year) money.Yen {
	return salarySteps.At(year).At(salary)
}

const SalaryScheduleCeiling money.Yen = 6_600_000

const SalaryScheduleRewriteYear = 2025

type salaryScheduleFloors struct {
	nothing money.Yen

	bandsFrom money.Yen

	narrowFrom money.Yen

	narrowUntil money.Yen
}

var salaryScheduleBands = NewAmended("別表第五の階級",
	YearRow[salaryScheduleFloors]{Value: salaryScheduleFloors{
		nothing: 551_000, bandsFrom: 1_619_000, narrowFrom: 1_620_000, narrowUntil: 1_624_000,
	}},
	YearRow[salaryScheduleFloors]{FromYear: SalaryScheduleRewriteYear, Value: salaryScheduleFloors{
		nothing: 651_000, bandsFrom: 1_900_000, narrowFrom: 1_900_000, narrowUntil: 1_900_000,
	}},
)

func SalaryIncome(salary money.Yen, year date.Year) money.Yen {
	floors := salaryScheduleBands.At(year)
	if salary < SalaryScheduleCeiling {
		switch {
		case salary < floors.nothing:
			return 0
		case salary < floors.bandsFrom:
		case salary < floors.narrowFrom:
			salary = floors.bandsFrom
		case salary < floors.narrowUntil:
			salary -= salary % salaryScheduleNarrowStep
		default:
			salary -= salary % SalaryScheduleStep
		}
	}
	income := salary - SalaryIncomeDeduction(salary, year)
	return max(income, 0)
}

const (
	SalaryScheduleStep       money.Yen = 4_000
	salaryScheduleNarrowStep money.Yen = 2_000
)
