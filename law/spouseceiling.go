package law

import (
	"io/fs"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const SpouseIncomeCeilingTableName = "national/spouse-income-ceiling"

const SpouseIncomeCeilingColumn tsv.ColumnName = "所得要件[円]"

type SpouseIncomeCeilingTable struct {
	YearYenTable
}

func (t SpouseIncomeCeilingTable) Satisfies(income money.Yen, incomeYear date.Year) bool {
	return income <= t.Ceiling(incomeYear)
}

func ParseSpouseIncomeCeilingTable(table *tsv.Table) (SpouseIncomeCeilingTable, error) {
	amounts, err := ParseYearYenTable(table, SpouseIncomeCeilingTableName, SpouseIncomeCeilingColumn)
	if err != nil {
		return SpouseIncomeCeilingTable{}, err
	}
	return SpouseIncomeCeilingTable{YearYenTable: amounts}, nil
}

func LoadSpouseIncomeCeilingTable(fsys fs.FS) (SpouseIncomeCeilingTable, error) {
	table, err := LoadShape(fsys, Shape{Name: SpouseIncomeCeilingTableName}, "")
	if err != nil {
		return SpouseIncomeCeilingTable{}, err
	}
	return ParseSpouseIncomeCeilingTable(table)
}

func (t SpouseIncomeCeilingTable) Ceiling(incomeYear date.Year) money.Yen {
	return t.Amount(incomeYear)
}
