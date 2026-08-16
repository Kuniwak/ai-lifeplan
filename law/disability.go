package law

import (
	"fmt"
	"maps"
	"slices"

	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/wording"
)

const DisabilityDeductionTableName = "national/disability-deduction"

const (
	DisabilityCategoryColumn  tsv.ColumnName = "区分"
	DisabilityIncomeTaxColumn tsv.ColumnName = "所得税控除額[円]"
	DisabilityResidentColumn  tsv.ColumnName = "住民税控除額[円]"
)

type DisabilityCategoryValue string

const (
	OrdinaryDisability DisabilityCategoryValue = "障害者"

	SpecialDisability DisabilityCategoryValue = "特別障害者"

	CohabitingSpecialDisability DisabilityCategoryValue = "同居特別障害者"
)

type DisabilityDeductionTable struct {
	categories map[DisabilityCategoryValue]Deduction
}

func ParseDisabilityDeductionTable(table *tsv.Table) (DisabilityDeductionTable, error) {
	r, err := newReader(table, DisabilityDeductionTableName, DisabilityCategoryColumn, DisabilityIncomeTaxColumn, DisabilityResidentColumn)
	if err != nil {
		return DisabilityDeductionTable{}, fmt.Errorf("law.ParseDisabilityDeductionTable: %w", err)
	}

	categories := make(map[DisabilityCategoryValue]Deduction, r.Rows())
	for row := range r.Rows() {
		category := DisabilityCategoryValue(r.Field(row, DisabilityCategoryColumn))
		if _, duplicate := categories[category]; duplicate {
			return DisabilityDeductionTable{}, wording.DuplicateKeyError(
				fmt.Sprintf("law.ParseDisabilityDeductionTable: row %d", row+1),
				string(DisabilityCategoryColumn), wording.Name(category), "which deduction applies")
		}

		incomeTax, err := r.Yen(row, DisabilityIncomeTaxColumn)
		if err != nil {
			return DisabilityDeductionTable{}, fmt.Errorf("law.ParseDisabilityDeductionTable: %w", err)
		}
		resident, err := r.Yen(row, DisabilityResidentColumn)
		if err != nil {
			return DisabilityDeductionTable{}, fmt.Errorf("law.ParseDisabilityDeductionTable: %w", err)
		}

		categories[category] = Deduction{IncomeTax: incomeTax, Resident: resident}
	}

	if len(categories) == 0 {
		return DisabilityDeductionTable{}, fmt.Errorf("law.ParseDisabilityDeductionTable: the table has no categories, so every lookup would miss")
	}
	return DisabilityDeductionTable{categories: categories}, nil
}

func (t DisabilityDeductionTable) Categories() []DisabilityCategoryValue {
	return slices.Sorted(maps.Keys(t.categories))
}

func (t DisabilityDeductionTable) Lookup(category DisabilityCategoryValue) (Deduction, error) {
	deduction, ok := t.categories[category]
	if !ok {
		return Deduction{}, fmt.Errorf(
			"law.DisabilityDeductionTable.Lookup: no 障害者控除 written down for %q; the categories with an amount are %v",
			category, t.Categories())
	}
	return deduction, nil
}
