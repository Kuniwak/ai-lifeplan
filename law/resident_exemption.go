package law

import (
	"fmt"
	"io/fs"
	"slices"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	ResidentExemptionPerPersonColumn tsv.ColumnName = "均等割非課税限度額の一人あたり[円]"

	ResidentExemptionAdditionColumn tsv.ColumnName = "均等割非課税限度額の加算額[円]"
)

type ResidentExemption struct {
	PerPerson, Addition money.Yen
}

type ResidentExemptions = YearTable[ResidentExemption]

const ResidentExemptionTableName = "resident-tax-exemption"

func ParseResidentExemptions(table *tsv.Table) (ResidentExemptions, error) {
	r, err := newReader(table, ResidentExemptionTableName,
		LawStartYearColumn, LawEndYearColumn,
		ResidentExemptionPerPersonColumn, ResidentExemptionAdditionColumn)
	if err != nil {
		return ResidentExemptions{}, fmt.Errorf("law.ParseResidentExemptions: %w", err)
	}

	periods, err := readYearPeriods(r, func(row int) (ResidentExemption, error) {
		var parsed ResidentExemption
		if parsed.PerPerson, err = r.Yen(row, ResidentExemptionPerPersonColumn); err != nil {
			return parsed, err
		}
		if parsed.Addition, err = r.Yen(row, ResidentExemptionAdditionColumn); err != nil {
			return parsed, err
		}
		return parsed, nil
	})
	if err != nil {
		return ResidentExemptions{}, fmt.Errorf("law.ParseResidentExemptions: %w", err)
	}

	built, err := NewYearTableOfPeriods(periods)
	if err != nil {
		return ResidentExemptions{}, fmt.Errorf("law.ParseResidentExemptions: %w", err)
	}
	return built, nil
}

func LoadResidentExemptions(fsys fs.FS, municipality Municipality) (ResidentExemptions, error) {
	table, err := LoadRegionalTable(fsys, string(municipality), ResidentExemptionTableName)
	if err != nil {
		return ResidentExemptions{}, err
	}
	return ParseResidentExemptions(table)
}

func MustLoadResidentExemptions(t testingT, fsys fs.FS, municipality Municipality) ResidentExemptions {
	t.Helper()

	exemptions, err := LoadResidentExemptions(fsys, municipality)
	if err != nil {
		t.Fatalf("law.LoadResidentExemptions: %v", err)
	}
	return exemptions
}

var residentExemptionBase = NewAmendedFrom("均等割非課税限度額の基本額に足す額", 2017,
	YearRow[money.Yen]{FromYear: 2017, Value: 0},
	YearRow[money.Yen]{FromYear: 2021, Value: 100_000},
)

var residentIncomeExemption = NewAmendedFrom("所得割の非課税限度額", 2017,
	YearRow[residentIncomeExemptionRow]{FromYear: 2017, Value: residentIncomeExemptionRow{
		PerPerson: 350_000, Addition: 320_000, Base: 0,
	}},
	YearRow[residentIncomeExemptionRow]{FromYear: 2021, Value: residentIncomeExemptionRow{
		PerPerson: 350_000, Addition: 320_000, Base: 100_000,
	}},
)

type residentIncomeExemptionRow struct {
	PerPerson, Addition, Base money.Yen
}

var residentDisabilityExemptionCeiling = NewAmendedFrom("障害者等の非課税限度額", 2017,
	YearRow[money.Yen]{FromYear: 2017, Value: 1_250_000},
	YearRow[money.Yen]{FromYear: 2021, Value: 1_350_000},
)

type RecordFloor struct {
	What         string
	FirstWritten FirstWrittenYear
}

func ResidentExemptionRecordFloors() []RecordFloor {
	return []RecordFloor{
		{residentDisabilityExemptionCeiling.name, residentDisabilityExemptionCeiling.FirstWrittenYear},
		{residentExemptionBase.name, residentExemptionBase.FirstWrittenYear},
		{residentIncomeExemption.name, residentIncomeExemption.FirstWrittenYear},
	}
}

func RecordFloors() []RecordFloor {
	return slices.Concat(ResidentExemptionRecordFloors(), SpouseDeductionRecordFloors())
}

type ResidentTaxLiability struct {
	PerCapita bool

	Income bool
}

func ResidentTaxLiabilityOf(totalIncome money.Yen, dependents int, disabled bool, e ResidentExemption, taxYear date.Year) ResidentTaxLiability {
	if disabled && totalIncome <= residentDisabilityExemptionCeiling.At(taxYear) {
		return ResidentTaxLiability{}
	}

	heads := dependents + 1

	base := residentExemptionBase.At(taxYear)

	income := residentIncomeExemption.At(taxYear)

	perCapitaLimit := e.PerPerson.Times(heads) + base
	incomeLimit := income.PerPerson.Times(heads) + income.Base
	if dependents > 0 {
		perCapitaLimit += e.Addition
		incomeLimit += income.Addition
	}

	return ResidentTaxLiability{
		PerCapita: totalIncome > perCapitaLimit,
		Income:    totalIncome > incomeLimit,
	}
}
