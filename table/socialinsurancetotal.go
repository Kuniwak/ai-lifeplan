package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type SocialInsuranceTotalRow struct {
	Health, NursingCare, Pension, EmploymentInsurance money.Yen

	Kokuho, Kouki, NationalPension money.Yen

	NursingCareFirstCategory money.Yen

	Total money.Yen
}

type SocialInsuranceTotalInput struct {
	Employees map[PersonName]relation.Table[SocialInsuranceRow]

	Household relation.Table[HouseholdInsuranceRow]
}

func SocialInsuranceTotalTable(in SocialInsuranceTotalInput) (relation.Table[SocialInsuranceTotalRow], error) {
	var empty relation.Table[SocialInsuranceTotalRow]

	years := in.Household.Years()
	rows := make([]relation.Row[SocialInsuranceTotalRow], 0, len(years))

	for _, y := range years {
		household, _ := in.Household.At(y)

		var row SocialInsuranceTotalRow
		for person, table := range in.Employees {
			employee, ok := table.At(y)
			if !ok {
				return empty, fmt.Errorf("table.SocialInsuranceTotalTable: %q has no premiums for %d", person, y)
			}
			row.Health += employee.Health
			row.NursingCare += employee.Nursing
			row.Pension += employee.Pension
			row.EmploymentInsurance += employee.EmploymentInsurance
		}

		row.Kokuho = household.Kokuho
		row.Kouki = household.Kouki
		row.NationalPension = household.NationalPension
		row.NursingCareFirstCategory = household.NursingCare

		row.Total = row.Health + row.NursingCare + row.Pension + row.EmploymentInsurance +
			row.Kokuho + row.Kouki + row.NationalPension + row.NursingCareFirstCategory

		rows = append(rows, relation.Row[SocialInsuranceTotalRow]{Year: y, Value: row})
	}

	return relation.New(rows), nil
}
