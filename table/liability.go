package table

import (
	"fmt"
	"slices"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type DependentsInput struct {
	Taxpayer, Spouse PersonName

	Income map[PersonName]relation.Table[money.Yen]

	SpouseIncomeCeiling law.SpouseIncomeCeilingTable

	ClaimsDependents bool
}

type Dependents struct {
	SpouseSameLivelihood bool

	Relatives []PersonYear
}

func (d Dependents) Count() int {
	count := len(d.Relatives)
	if d.SpouseSameLivelihood {
		count++
	}
	return count
}

func (d Dependents) Includes(name PersonName) bool {
	return slices.ContainsFunc(d.Relatives, func(p PersonYear) bool { return p.Name == name })
}

func DependentsOf(in DependentsInput, calendar CalendarRow, y date.Year) Dependents {
	var d Dependents

	if in.Spouse != "" {
		spouseIncome, _ := in.Income[in.Spouse].At(y)
		d.SpouseSameLivelihood = calendar.InHousehold(in.Spouse) &&
			in.SpouseIncomeCeiling.Satisfies(spouseIncome, y)
	}

	if in.ClaimsDependents {
		for _, person := range calendar.Ages {
			if person.Name == in.Taxpayer || person.Name == in.Spouse {
				continue
			}
			if !calendar.InHousehold(person.Name) {
				continue
			}
			income, _ := in.Income[person.Name].At(y)
			if !in.SpouseIncomeCeiling.Satisfies(income, y) {
				continue
			}
			d.Relatives = append(d.Relatives, person)
		}
	}
	return d
}

type ResidentTaxLiabilityInput struct {
	Calendar relation.Table[CalendarRow]

	Dependents DependentsInput

	Disabilities Disabilities

	Levies law.ResidentLevies
}

func ResidentTaxLiabilityTable(in ResidentTaxLiabilityInput) (relation.Table[law.ResidentTaxLiability], error) {
	var empty relation.Table[law.ResidentTaxLiability]

	years := in.Calendar.Years()
	rows := make([]relation.Row[law.ResidentTaxLiability], 0, len(years))

	for _, y := range years {
		calendar, _ := in.Calendar.At(y)
		if calendar.Municipality == "" {
			rows = append(rows, relation.Row[law.ResidentTaxLiability]{Year: y})
			continue
		}

		tables, ok := in.Levies.TablesFor(calendar.Municipality)
		if !ok {
			return empty, fmt.Errorf(
				"table.ResidentTaxLiabilityTable: %d: no tables for %q; add data/law/%s/{%s,%s,%s}.tsv",
				y, calendar.Municipality, calendar.Municipality,
				law.ResidentRateTableName, law.ResidentPerCapitaTableName, law.ResidentExemptionTableName)
		}
		exemption, known := tables.Exemptions().At(y)
		if !known {
			return empty, fmt.Errorf(
				"table.ResidentTaxLiabilityTable: %d: %q has no 非課税限度額 for that year; the first row of data/law/%s/%s.tsv starts later",
				y, calendar.Municipality, calendar.Municipality, law.ResidentExemptionTableName)
		}

		judged := y - 1
		judgedCalendar, hasJudged := in.Calendar.At(judged)
		if !hasJudged {
			rows = append(rows, relation.Row[law.ResidentTaxLiability]{Year: y})
			continue
		}

		income, _ := in.Dependents.Income[in.Dependents.Taxpayer].At(judged)
		dependents := DependentsOf(in.Dependents, judgedCalendar, judged).Count()
		disabled := in.Disabilities.AppliesTo(in.Dependents.Taxpayer, judged)

		rows = append(rows, relation.Row[law.ResidentTaxLiability]{
			Year:  y,
			Value: law.ResidentTaxLiabilityOf(income, dependents, disabled, exemption, y),
		})
	}
	return relation.New(rows), nil
}
