package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type ExpenseRow struct {
	Living        money.Yen
	CoupleLiving  money.Yen
	ChildLiving   money.Yen
	ChildLivingOf []PersonPremium
	Medical       money.Yen
	AllowanceOf   []PersonPremium

	MedicalPaid, MedicalRefunded money.Yen

	Allowance     money.Yen
	Extraordinary money.Yen

	Education money.Yen
	TuitionOf []PersonPremium

	Insurance money.Yen

	Life         money.Yen
	MedicalCover money.Yen

	Earthquake           money.Yen
	EarthquakeDeductible money.Yen

	Fire money.Yen

	Housing     money.Yen
	Rent        money.Yen
	Deposit     money.Yen
	LoanPaid    money.Yen
	Maintenance money.Yen

	Total money.Yen

	Recurring money.Yen
}

type ExpenseInput struct {
	Calendar relation.Table[CalendarRow]

	CoupleLivingMonthly relation.Table[money.Yen]

	ChildLivingByStage, TuitionByStage map[Stage]money.Yen

	AllowanceMonthly map[PersonName]relation.Table[money.Yen]

	MedicalPaid, MedicalRefunded relation.Table[money.Yen]

	Extraordinary       map[date.Year]money.Yen
	FireInsurance       map[date.Year]money.Yen
	EarthquakeInsurance map[date.Year]money.Yen

	InsuranceTerm map[date.Year]money.Yen
	Maintenance   map[date.Year]money.Yen
	Deposit       map[date.Year]money.Yen

	LifeInsurance, MedicalInsurance, Rent relation.Table[money.Yen]

	Loan relation.Table[LoanYear]

	PriceLevelsByItem map[input.PricedItem]relation.Table[money.Factor]
}

type Inflator func(input.PricedItem, money.Yen) money.Yen

func InflatorAt(levels map[input.PricedItem]relation.Table[money.Factor], y date.Year) (Inflator, error) {
	for item, byYear := range levels {
		if _, ok := byYear.At(y); !ok {
			return nil, fmt.Errorf(
				"table.InflatorAt: %d の %q の物価が分からない。インフレ率の表がこの年に届いていない", y, item)
		}
	}
	return func(item input.PricedItem, amount money.Yen) money.Yen {
		level, ok := levels[item].At(y)
		if !ok {
			return amount
		}
		return level.Apply(amount)
	}, nil
}

func earthquakeDeductibleByYear(paid, term map[date.Year]money.Yen) (map[date.Year]money.Yen, error) {
	spread := make(map[date.Year]money.Yen, len(paid)*5)
	for year, lump := range paid {
		if lump == 0 {
			continue
		}
		years := int(term[year])
		if years <= 0 {
			return nil, fmt.Errorf(
				"%d 年に地震保険料 %d 円を払っているのに保険期間が書かれていない。"+
					"**控除はその年ぶんだけ**（所得税法施行令第二百十四条第二項）なので、"+
					"割る年数が要る", year, lump)
		}
		share := lump / money.Yen(years)
		for i := 0; i < years; i++ {
			spread[year+date.Year(i)] += share
		}
	}
	return spread, nil
}

func ExpenseTable(in ExpenseInput) (relation.Table[ExpenseRow], error) {
	var empty relation.Table[ExpenseRow]

	levels := in.PriceLevelsByItem

	deductible, err := earthquakeDeductibleByYear(in.EarthquakeInsurance, in.InsuranceTerm)
	if err != nil {
		return empty, fmt.Errorf("table.ExpenseTable: %w", err)
	}

	years := in.Calendar.Years()
	rows := make([]relation.Row[ExpenseRow], 0, len(years))

	for _, y := range years {
		calendar, _ := in.Calendar.At(y)

		into, err := InflatorAt(levels, y)
		if err != nil {
			return empty, err
		}

		var row ExpenseRow

		monthly, ok := in.CoupleLivingMonthly.At(y)
		if !ok {
			return empty, fmt.Errorf("table.ExpenseTable: no living cost for %d", y)
		}
		row.CoupleLiving = into(input.CoupleLivingItem, monthly*date.MonthsAYear)

		for _, person := range calendar.Ages {
			if allowance, ok := in.AllowanceMonthly[person.Name].At(y); ok {
				paid := into(input.AllowanceItem, allowance*date.MonthsAYear)
				row.AllowanceOf = append(row.AllowanceOf, PersonPremium{Name: person.Name, Premium: paid})
				row.Allowance += paid
			}

			if !person.IsChild() {
				continue
			}
			living := into(input.ChildLivingItem, in.ChildLivingByStage[person.Stage])
			row.ChildLivingOf = append(row.ChildLivingOf, PersonPremium{Name: person.Name, Premium: living})
			row.ChildLiving += living

			tuition := into(input.EducationItem, in.TuitionByStage[person.Stage])
			row.TuitionOf = append(row.TuitionOf, PersonPremium{Name: person.Name, Premium: tuition})
			row.Education += tuition
		}

		if paid, ok := in.MedicalPaid.At(y); ok {
			row.MedicalPaid = into(input.MedicalItem, paid)
		}
		if refunded, ok := in.MedicalRefunded.At(y); ok {
			row.MedicalRefunded = into(input.MedicalItem, refunded)
		}
		row.Medical = max(row.MedicalPaid-row.MedicalRefunded, 0)
		row.Extraordinary = into(input.ExtraordinaryItem, in.Extraordinary[y])
		row.Living = row.CoupleLiving + row.ChildLiving + row.Medical + row.Allowance + row.Extraordinary

		if life, ok := in.LifeInsurance.At(y); ok {
			row.Life = into(input.LifeInsuranceItem, life)
		}
		if medical, ok := in.MedicalInsurance.At(y); ok {
			row.MedicalCover = into(input.LifeInsuranceItem, medical)
		}
		row.Earthquake = into(input.QuakeInsuranceItem, in.EarthquakeInsurance[y])
		row.EarthquakeDeductible = into(input.QuakeInsuranceItem, deductible[y])
		row.Fire = into(input.FireInsuranceItem, in.FireInsurance[y])
		row.Insurance = row.Life + row.MedicalCover + row.Earthquake + row.Fire

		if rent, ok := in.Rent.At(y); ok {
			row.Rent = into(input.RentItem, rent)
		}
		row.Deposit = into(input.DepositItem, in.Deposit[y])
		row.Maintenance = into(input.MaintenanceItem, in.Maintenance[y])
		if loan, ok := in.Loan.At(y); ok {
			row.LoanPaid = into(input.LoanPaidItem, loan.Paid)
		}
		row.Housing = row.Rent + row.Deposit + row.LoanPaid + row.Maintenance

		row.Total = row.Living + row.Education + row.Insurance + row.Housing
		row.Recurring = row.Total - row.Deposit

		rows = append(rows, relation.Row[ExpenseRow]{Year: y, Value: row})
	}

	return relation.New(rows), nil
}
