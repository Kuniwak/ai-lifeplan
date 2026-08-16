package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

func ExpenseInputFrom(tables map[tsv.Slot]*tsv.Table, calendar relation.Table[CalendarRow]) (ExpenseInput, error) {
	var in ExpenseInput
	in.Calendar = calendar

	years := calendar.Years()
	if len(years) == 0 {
		return in, fmt.Errorf("table.ExpenseInputFrom: the calendar is empty")
	}
	from, to := years[0], years[len(years)-1]

	step := func(slot tsv.Slot, column tsv.ColumnName) (relation.Table[money.Yen], error) {
		return ReadYenStep(tables[slot], slot, column, from, to)
	}

	var err error
	if in.CoupleLivingMonthly, err = step(input.LivingCostSlot, input.CoupleLivingColumn); err != nil {
		return in, err
	}
	if in.LifeInsurance, err = step(input.LifeInsurancePremiumSlot, input.LifeInsuranceColumn); err != nil {
		return in, err
	}
	if in.MedicalInsurance, err = step(input.LifeInsurancePremiumSlot, input.MedicalInsuranceColumn); err != nil {
		return in, err
	}
	if in.Rent, err = step(input.HousingRentSlot, input.RentColumn); err != nil {
		return in, err
	}

	if in.ChildLivingByStage, err = readByStage(tables[input.ChildLivingCostSlot], input.ChildLivingCostSlot, input.ChildLivingColumn); err != nil {
		return in, err
	}
	if in.TuitionByStage, err = readByStage(tables[input.TuitionSlot], input.TuitionSlot, input.TuitionColumn); err != nil {
		return in, err
	}

	in.AllowanceMonthly = make(map[PersonName]relation.Table[money.Yen], 2)
	for person, slot := range map[PersonName]tsv.Slot{"夫": input.AllowanceHusbandSlot, "妻": input.AllowanceWifeSlot} {
		if in.AllowanceMonthly[person], err = step(slot, input.AllowanceColumn); err != nil {
			return in, err
		}
	}

	if in.MedicalPaid, err = step(input.MedicalExpenseSlot, input.MedicalColumn); err != nil {
		return in, err
	}
	if in.MedicalRefunded, err = step(input.MedicalExpenseSlot, input.MedicalRefundColumn); err != nil {
		return in, err
	}

	if in.Extraordinary, err = readEvents(tables[input.ExtraordinaryCostSlot], input.ExtraordinaryCostSlot, input.ExtraordinaryColumn); err != nil {
		return in, err
	}
	if in.FireInsurance, err = readEvents(tables[input.PropertyInsuranceSlot], input.PropertyInsuranceSlot, input.FireInsuranceColumn); err != nil {
		return in, err
	}
	if in.EarthquakeInsurance, err = readEvents(tables[input.PropertyInsuranceSlot], input.PropertyInsuranceSlot, input.QuakeInsuranceColumn); err != nil {
		return in, err
	}
	if in.InsuranceTerm, err = readEvents(tables[input.PropertyInsuranceSlot], input.PropertyInsuranceSlot, input.InsuranceTermColumn); err != nil {
		return in, err
	}
	if in.Maintenance, err = readEvents(tables[input.HousingMaintenanceSlot], input.HousingMaintenanceSlot, input.MaintenanceColumn); err != nil {
		return in, err
	}

	if in.Deposit, err = readEventsKeyedOn(tables[input.HousingSlot], input.HousingSlot, input.BoughtInColumn, input.DepositColumn); err != nil {
		return in, err
	}

	if in.PriceLevelsByItem, err = PriceLevelsByItemFrom(tables, from, to); err != nil {
		return in, err
	}
	return in, nil
}

func readEvents(table *tsv.Table, slot tsv.Slot, column tsv.ColumnName) (map[date.Year]money.Yen, error) {
	return readEventsKeyedOn(table, slot, input.YearColumn, column)
}

func readEventsKeyedOn(table *tsv.Table, slot tsv.Slot, yearColumn, column tsv.ColumnName) (map[date.Year]money.Yen, error) {
	r, err := tsv.NewReader(table, slot, yearColumn, column)
	if err != nil {
		return nil, err
	}

	byYear := make(map[date.Year]money.Yen, r.Rows())
	for row := range r.Rows() {
		year, err := r.Year(row, yearColumn)
		if err != nil {
			return nil, err
		}
		amount, err := r.Yen(row, column)
		if err != nil {
			return nil, err
		}
		byYear[year] += amount
	}
	return byYear, nil
}

func readByStage(table *tsv.Table, slot tsv.Slot, column tsv.ColumnName) (map[Stage]money.Yen, error) {
	r, err := tsv.NewReader(table, slot, input.StageColumn, column)
	if err != nil {
		return nil, err
	}

	byStage := make(map[Stage]money.Yen, r.Rows())
	for row := range r.Rows() {
		amount, err := r.Yen(row, column)
		if err != nil {
			return nil, err
		}
		byStage[Stage(r.Field(row, input.StageColumn))] = amount
	}
	return byStage, nil
}
