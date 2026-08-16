package law

import (
	"fmt"
	"io/fs"

	"github.com/Kuniwak/lifeplan/tsv"
)

const ForestEnvironmentTaxColumn tsv.ColumnName = "森林環境税[円]"

const ForestEnvironmentTaxTableName = "national/forest-environment-tax"

func LoadForestEnvironmentTaxTable(fsys fs.FS) (YearYenTable, error) {
	f, err := fsys.Open(ForestEnvironmentTaxTableName + ".tsv")
	if err != nil {
		return YearYenTable{}, fmt.Errorf("law.LoadForestEnvironmentTaxTable: %w", err)
	}
	defer f.Close()

	table, err := tsv.Read(f)
	if err != nil {
		return YearYenTable{}, fmt.Errorf("law.LoadForestEnvironmentTaxTable: %w", err)
	}
	return ParseYearYenTable(table, ForestEnvironmentTaxTableName, ForestEnvironmentTaxColumn)
}
