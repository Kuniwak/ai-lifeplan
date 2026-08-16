package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type Pay struct {
	Salary money.Yen

	Bonus        money.Yen
	BonusesAYear int

	LeaveMonths int

	WeeklyHours int

	Workplace law.Workplace

	ExemptMonths date.Months

	MiscellaneousReceipts money.Yen

	BusinessReceipts, BusinessExpenses money.Yen

	BlueFormRecordKeeping law.BlueFormRecordKeeping
}

func (p Pay) BusinessIncome() money.Yen {
	return p.BusinessReceipts - p.BusinessExpenses
}

func (p Pay) Monthly() money.Yen {
	return (p.Salary - p.Bonus) / date.MonthsAYear
}

func (p Pay) BonusPayment() money.Yen {
	if p.BonusesAYear <= 0 {
		return 0
	}
	return p.Bonus / money.Yen(p.BonusesAYear)
}

type Pension struct {
	StartYear date.Year

	Basic, Proportional money.Yen

	Supplement money.Yen

	SupplementFrom, SupplementThrough date.Date

	Expected money.Rate
}

func (p Pension) AtLevel(level PensionLevel) Pension {
	p.Basic = p.Basic.Mul(level.Basic, money.Truncate)
	p.Proportional = p.Proportional.Mul(level.Proportional, money.Truncate)
	p.Supplement = p.Supplement.Mul(level.Proportional, money.Truncate)
	return p
}

func (p Pension) Received(y date.Year) money.Yen {
	if y < p.StartYear {
		return 0
	}
	return (p.Basic + p.Proportional + p.supplementIn(y)).Mul(p.Expected, money.Truncate)
}

func (p Pension) supplementIn(y date.Year) money.Yen {
	if p.Supplement == 0 || p.SupplementThrough.Year == 0 {
		return 0
	}
	months := date.MonthsOfYearIn(y, p.SupplementFrom, p.SupplementThrough).Count()
	if months == 0 {
		return 0
	}
	return p.Supplement.Mul(money.NewRate(int64(months), date.MonthsAYear), money.Truncate)
}

func (p Pension) PartsReceived(y date.Year) (basic, proportional, supplement money.Yen) {
	if y < p.StartYear {
		return 0, 0, 0
	}
	return p.Basic.Mul(p.Expected, money.Truncate),
		p.Proportional.Mul(p.Expected, money.Truncate),
		p.supplementIn(y).Mul(p.Expected, money.Truncate)
}

func (p Pension) OldAgeBenefit(y date.Year) money.Yen {
	basic, _, _ := p.PartsReceived(y)
	return basic
}

type IncomeRow struct {
	Salary, SalaryIncome money.Yen

	SalaryIncomeAdjustment money.Yen

	ChildcareLeaveBenefit money.Yen

	BusinessReceipts, BusinessExpenses, BusinessDeduction, BusinessIncome money.Yen

	MiscellaneousReceipts, MiscellaneousIncome money.Yen

	PensionReceived, PensionIncome money.Yen

	PensionBasic, PensionProportional, PensionSupplement money.Yen

	OldAgePensionBenefit money.Yen

	Total, TotalIncome money.Yen
}

type IncomeInput struct {
	Pay relation.Table[Pay]

	Pension Pension

	Age relation.Table[int]

	HasYoungDependant relation.Table[bool]

	ChildcareLeave law.ChildcareLeaveBenefitTable

	PriceLevels relation.Table[money.Factor]

	WageLevels relation.Table[money.Factor]
}

func (p Pay) Nominal(prices, wages money.Factor) Pay {
	earned := prices.Compose(wages)
	p.Salary = earned.Apply(p.Salary)
	p.Bonus = earned.Apply(p.Bonus)
	p.BusinessReceipts = prices.Apply(p.BusinessReceipts)
	p.BusinessExpenses = prices.Apply(p.BusinessExpenses)
	p.MiscellaneousReceipts = prices.Apply(p.MiscellaneousReceipts)
	return p
}

func (p Pension) Nominal(level money.Factor) Pension {
	p.Basic = level.Apply(p.Basic)
	p.Proportional = level.Apply(p.Proportional)
	p.Supplement = level.Apply(p.Supplement)
	return p
}

func (in IncomeInput) NominalPay() (relation.Table[Pay], error) {
	years := in.Pay.Years()
	rows := make([]relation.Row[Pay], 0, len(years))

	for _, y := range years {
		written, _ := in.Pay.At(y)

		level, ok := in.PriceLevels.At(y)
		if !ok {
			return relation.Table[Pay]{}, fmt.Errorf("table.NominalPay: %d の物価が分からない", y)
		}
		wage, ok := in.WageLevels.At(y)
		if !ok {
			return relation.Table[Pay]{}, fmt.Errorf("table.NominalPay: %d の実質賃金上昇率が分からない", y)
		}
		rows = append(rows, relation.Row[Pay]{Year: y, Value: written.Nominal(level, wage)})
	}
	return relation.New(rows), nil
}

func IncomeTable(in IncomeInput) (relation.Table[IncomeRow], error) {
	var empty relation.Table[IncomeRow]

	years := in.Pay.Years()
	rows := make([]relation.Row[IncomeRow], 0, len(years))

	nominalPay, err := in.NominalPay()
	if err != nil {
		return empty, err
	}

	for _, y := range years {
		level, ok := in.PriceLevels.At(y)
		if !ok {
			return empty, fmt.Errorf("table.IncomeTable: %d の物価が分からない", y)
		}
		pay, _ := nominalPay.At(y)
		pension := in.Pension.Nominal(level)

		age, ok := in.Age.At(y)
		if !ok {
			return empty, fmt.Errorf("table.IncomeTable: the calendar has no age for %d", y)
		}
		young, ok := in.HasYoungDependant.At(y)
		if !ok {
			return empty, fmt.Errorf("table.IncomeTable: nothing is known about the dependants of %d", y)
		}

		var row IncomeRow
		row.Salary = pay.Salary
		row.SalaryIncome = law.SalaryIncome(row.Salary, y)
		adjusted := law.SalaryIncomeAfterAdjustment(row.Salary, young, y)

		benefit, err := in.ChildcareLeave.Benefit(pay.Monthly(), pay.LeaveMonths, y)
		if err != nil {
			return empty, fmt.Errorf("table.IncomeTable: %d: %w", y, err)
		}
		row.ChildcareLeaveBenefit = benefit

		row.BusinessReceipts = pay.BusinessReceipts
		row.BusinessExpenses = pay.BusinessExpenses
		businessIncomeBeforeDeduction := pay.BusinessIncome()
		row.BusinessDeduction, err = law.BlueFormDeduction(pay.BlueFormRecordKeeping, businessIncomeBeforeDeduction)
		if err != nil {
			return empty, fmt.Errorf("table.IncomeTable: %d: %w", y, err)
		}
		row.BusinessIncome = businessIncomeBeforeDeduction - row.BusinessDeduction
		adjusted += row.BusinessIncome

		row.MiscellaneousReceipts = pay.MiscellaneousReceipts
		row.MiscellaneousIncome = pay.MiscellaneousReceipts
		adjusted += row.MiscellaneousIncome

		row.PensionReceived = pension.Received(y)
		row.PensionBasic, row.PensionProportional, row.PensionSupplement = pension.PartsReceived(y)
		row.OldAgePensionBenefit = pension.OldAgeBenefit(y)
		row.PensionIncome = law.PensionIncome(row.PensionReceived, age, adjusted, y)

		first, second := law.TotalIncomeAdjustment(row.Salary, row.PensionIncome, young, y)
		row.SalaryIncomeAdjustment = first + second
		adjusted -= second

		row.Total = row.Salary + row.BusinessReceipts + row.MiscellaneousReceipts +
			row.ChildcareLeaveBenefit + row.PensionReceived
		row.TotalIncome = adjusted + row.PensionIncome

		rows = append(rows, relation.Row[IncomeRow]{Year: y, Value: row})
	}

	return relation.New(rows), nil
}
