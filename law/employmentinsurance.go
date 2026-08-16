package law

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const EmploymentInsuranceRateTableName = "national/employment-insurance-rate"

const EmploymentInsuranceRateColumn tsv.ColumnName = "雇用保険料率"

type EmploymentInsuranceTable struct {
	YearRateTable
}

func (t EmploymentInsuranceTable) Premium(salary money.Yen, year date.Year) money.Yen {
	if salary <= 0 {
		return 0
	}
	return salary.Mul(t.Rate(year), money.Truncate)
}
