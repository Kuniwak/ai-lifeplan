package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/wording"
)

func LoansFrom(tables map[tsv.Slot]*tsv.Table) ([]Loan, map[LoanName]money.Rate, Settlement, *date.Year, error) {
	boughtIn, err := readBoughtIn(tables[input.HousingSlot])
	if err != nil || boughtIn == nil {
		return nil, nil, nil, nil, err
	}

	settledIn, err := LoanSettlementFrom(tables)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	r, err := tsv.NewReader(tables[input.LoanSlot], input.LoanSlot,
		input.LoanNameColumn, input.LoanDrawnInColumn, input.LoanFirstYearColumn, input.LoanFirstMonthColumn,
		input.PrincipalColumn, input.LoanYearsColumn, input.LoanRateColumn,
		input.LoanFixedYearsColumn, input.LoanFloatingColumn)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if r.Rows() == 0 {
		return nil, nil, nil, nil, fmt.Errorf("table: %s: 契約が一つも書かれていない", input.LoanSlot)
	}

	loans := make([]Loan, 0, r.Rows())
	floating := make(map[LoanName]money.Rate, r.Rows())
	seen := make(map[LoanName]bool, r.Rows())
	for row := range r.Rows() {
		name := LoanName(r.Field(row, input.LoanNameColumn))
		if seen[name] {
			return nil, nil, nil, nil, wording.DuplicateKeyError(fmt.Sprintf("table: %s", input.LoanSlot),
				string(input.LoanNameColumn), wording.Name(name), "which terms it is repaid on")
		}
		seen[name] = true

		loan := Loan{Name: name}
		if loan.DrawnIn, err = r.Year(row, input.LoanDrawnInColumn); err != nil {
			return nil, nil, nil, nil, err
		}
		if loan.FirstYear, err = r.Year(row, input.LoanFirstYearColumn); err != nil {
			return nil, nil, nil, nil, err
		}
		if loan.FirstMonth, err = r.Count(row, input.LoanFirstMonthColumn); err != nil {
			return nil, nil, nil, nil, err
		}
		if loan.FirstMonth < 1 || loan.FirstMonth > date.MonthsAYear {
			return nil, nil, nil, nil, fmt.Errorf(
				"table: %s: %q の初回返済月が %d である", input.LoanSlot, name, loan.FirstMonth)
		}
		if loan.Principal, err = r.Yen(row, input.PrincipalColumn); err != nil {
			return nil, nil, nil, nil, err
		}
		if loan.Years, err = r.Count(row, input.LoanYearsColumn); err != nil {
			return nil, nil, nil, nil, err
		}
		if loan.AnnualRate, err = r.Percent(row, input.LoanRateColumn); err != nil {
			return nil, nil, nil, nil, err
		}
		if loan.FixedYears, err = r.Count(row, input.LoanFixedYearsColumn); err != nil {
			return nil, nil, nil, nil, err
		}
		if floating[name], err = r.Percent(row, input.LoanFloatingColumn); err != nil {
			return nil, nil, nil, nil, err
		}
		loans = append(loans, loan)
	}

	return loans, floating, settledIn, boughtIn, nil
}

func LoanSettlementFrom(tables map[tsv.Slot]*tsv.Table) (*date.Year, error) {
	slot := input.LoanSettlementSlot

	r, err := tsv.NewReader(tables[slot], slot, input.LoanSettledInColumn)
	if err != nil {
		return nil, err
	}
	if r.Rows() != 1 {
		return nil, fmt.Errorf("table: %s: 一行でなければならない。%d 行ある", slot, r.Rows())
	}

	if input.Settlement(r.Field(0, input.LoanSettledInColumn)) == input.LoanNoSettlement {
		return nil, nil
	}
	year, err := r.Year(0, input.LoanSettledInColumn)
	if err != nil {
		return nil, err
	}
	return &year, nil
}

func DisabilitiesFrom(tables map[tsv.Slot]*tsv.Table) (Disabilities, error) {
	r, err := tsv.NewReader(tables[input.DisabilitySlot], input.DisabilitySlot,
		input.DisabledPersonColumn, input.DisabilityCertified, input.DisabilityCategory,
		input.DisabilityPensionColumn)
	if err != nil {
		return nil, err
	}

	byPerson := make(Disabilities, r.Rows())
	for row := range r.Rows() {
		certified, err := r.Year(row, input.DisabilityCertified)
		if err != nil {
			return nil, err
		}
		pension := law.DisabilityPensionEligible(r.Field(row, input.DisabilityPensionColumn))
		if _, err := pension.Eligible(); err != nil {
			return nil, r.Errorf(row, input.DisabilityPensionColumn, "%v", err)
		}
		byPerson[PersonName(r.Field(row, input.DisabledPersonColumn))] = Disability{
			Category:          law.DisabilityCategoryValue(r.Field(row, input.DisabilityCategory)),
			CertifiedIn:       certified,
			DisabilityPension: pension,
		}
	}
	return byPerson, nil
}

func ReturnsFrom(tables map[tsv.Slot]*tsv.Table, from, to date.Year) (relation.Table[money.Rate], error) {
	return readRateStep(tables[input.InvestmentReturnSlot], input.InvestmentReturnSlot, input.ReturnColumn, from, to)
}

func CrashesFrom(tables map[tsv.Slot]*tsv.Table) (map[date.Year]money.Rate, error) {
	r, err := tsv.NewReader(tables[input.FinancialCrisisSlot], input.FinancialCrisisSlot,
		input.YearColumn, input.CrashColumn)
	if err != nil {
		return nil, err
	}

	byYear := make(map[date.Year]money.Rate, r.Rows())
	for row := range r.Rows() {
		year, err := r.Year(row, input.YearColumn)
		if err != nil {
			return nil, err
		}
		fall, err := r.Percent(row, input.CrashColumn)
		if err != nil {
			return nil, err
		}
		byYear[year] = fall
	}
	return byYear, nil
}

func PropertyTaxFrom(
	tables map[tsv.Slot]*tsv.Table,
	calendar relation.Table[CalendarRow],
	depreciation law.DepreciationRateTable,
	rates law.PropertyTaxTable,
	construction relation.Table[money.Factor],
) (relation.Table[PropertyTaxRow], error) {
	var empty relation.Table[PropertyTaxRow]

	boughtIn, err := readBoughtIn(tables[input.HousingSlot])
	if err != nil {
		return empty, err
	}
	if boughtIn == nil {
		return relation.Constant(calendar.Years(), PropertyTaxRow{}), nil
	}

	r, err := tsv.NewReader(tables[input.PropertyAssessmentSlot], input.PropertyAssessmentSlot,
		input.AssessedYearColumn, input.LandValueColumn, input.HouseBaseColumn)
	if err != nil {
		return empty, err
	}
	if r.Rows() != 1 {
		return empty, fmt.Errorf(
			"table.PropertyTaxFrom: want exactly one assessment to anchor on, got %d", r.Rows())
	}

	assessedIn, err := r.Year(0, input.AssessedYearColumn)
	if err != nil {
		return empty, err
	}
	landValue, err := r.Yen(0, input.LandValueColumn)
	if err != nil {
		return empty, err
	}
	houseBase, err := r.Yen(0, input.HouseBaseColumn)
	if err != nil {
		return empty, err
	}

	return PropertyTaxTable(PropertyTaxInput{
		Calendar:          calendar,
		BuiltIn:           *boughtIn,
		LandValue:         landValue,
		HouseBaseAt:       houseBase,
		AssessedIn:        assessedIn,
		Depreciation:      depreciation,
		Table:             rates,
		ConstructionLevel: construction,
	})
}

func MunicipalitiesFrom(tables map[tsv.Slot]*tsv.Table) ([]law.Municipality, error) {
	r, err := tsv.NewReader(tables[input.ResidenceSlot], input.ResidenceSlot,
		input.YearColumn, input.MunicipalityColumn)
	if err != nil {
		return nil, err
	}
	if r.Rows() == 0 {
		return nil, fmt.Errorf("table.MunicipalitiesFrom: nowhere is written, so no local rules can be read")
	}

	lived := make([]law.Municipality, 0, r.Rows())
	for row := range r.Rows() {
		lived = append(lived, law.Municipality(r.Field(row, input.MunicipalityColumn)))
	}
	return lived, nil
}

func SellNISAFirstFrom(tables map[tsv.Slot]*tsv.Table) (bool, error) {
	slot := input.InvestmentSlot

	r, err := tsv.NewReader(tables[slot], slot, input.SellNISAFirstColumn)
	if err != nil {
		return false, err
	}
	if r.Rows() == 0 {
		return false, fmt.Errorf("table: %s: 行が無い", slot)
	}

	switch written := input.SellFirst(r.Field(r.Rows()-1, input.SellNISAFirstColumn)); written {
	case input.SellNISA:
		return true, nil
	case input.SellTaxable:
		return false, nil
	default:
		return false, fmt.Errorf(
			"table: %s: %s が %q である。%q か %q でなければならない",
			slot, input.SellNISAFirstColumn, written, input.SellNISA, input.SellTaxable)
	}
}
