package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

func SocialInsuranceDeductions(
	household relation.Table[HouseholdInsuranceRow],
	employees map[PersonName]relation.Table[SocialInsuranceRow],
	earner PersonName,
) (map[PersonName]relation.Table[money.Yen], error) {
	rows := make(map[PersonName][]relation.Row[money.Yen], 2)
	for _, row := range household.Rows() {
		by := row.Value.DeductionsBy(earner)
		for person, table := range employees {
			own, ok := table.At(row.Year)
			if !ok {
				return nil, fmt.Errorf(
					"table.SocialInsuranceDeductions: 世帯の保険の表に %d 年があるが、%q の社会保険の表には無い",
					row.Year, person)
			}
			by[person] += own.Health + own.Nursing + own.Pension + own.EmploymentInsurance
		}
		for person, amount := range by {
			rows[person] = append(rows[person], relation.Row[money.Yen]{Year: row.Year, Value: amount})
		}
	}

	deductions := make(map[PersonName]relation.Table[money.Yen], len(rows))
	for person, r := range rows {
		deductions[person] = relation.New(r)
	}
	return deductions, nil
}
