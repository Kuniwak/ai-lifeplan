package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
)

type ResidentTaxRow struct {
	Municipality law.Municipality

	JudgedOn date.Year

	TotalIncome money.Yen
	Deductions  money.Yen

	Taxable money.Yen

	PerCapita                          money.Yen
	PrefecturalIncome, MunicipalIncome money.Yen

	Adjustment money.Yen

	SpecialCredit money.Yen

	Liable law.ResidentTaxLiability

	ForestEnvironmentTax money.Yen

	Total money.Yen
}

func (r ResidentTaxRow) PerCapitaPaid() money.Yen {
	return r.PerCapita + r.ForestEnvironmentTax
}

type ResidentTaxInput struct {
	Calendar relation.Table[CalendarRow]

	IncomeTax relation.Table[IncomeTaxRow]

	Tables law.ResidentLevies

	Liable relation.Table[law.ResidentTaxLiability]
}

func ResidentTaxTable(in ResidentTaxInput) (relation.Table[ResidentTaxRow], error) {
	var empty relation.Table[ResidentTaxRow]

	years := in.Calendar.Years()
	rows := make([]relation.Row[ResidentTaxRow], 0, len(years))

	for _, y := range years {
		calendar, _ := in.Calendar.At(y)

		row := ResidentTaxRow{Municipality: calendar.Municipality, JudgedOn: y - 1}

		tables, hasMunicipality := in.Tables.TablesFor(calendar.Municipality)
		rates, hasRates := tables.Rates().At(y)
		perCapita, hasPerCapita := tables.PerCapita().At(y)
		judged, ok := in.IncomeTax.At(row.JudgedOn)
		switch {
		case calendar.Municipality == "":
			rows = append(rows, relation.Row[ResidentTaxRow]{Year: y, Value: row})
			continue

		case !hasMunicipality:
			return empty, fmt.Errorf(
				"table.ResidentTaxTable: %d: no tables for %q; add data/law/%s/{%s,%s,%s}.tsv",
				y, calendar.Municipality, calendar.Municipality,
				law.ResidentRateTableName, law.ResidentPerCapitaTableName, law.ResidentExemptionTableName)

		case !hasRates:
			return empty, fmt.Errorf(
				"table.ResidentTaxTable: %d: %q has no rates for that year; the first row of data/law/%s/%s.tsv starts later",
				y, calendar.Municipality, calendar.Municipality, law.ResidentRateTableName)

		case !hasPerCapita:
			return empty, fmt.Errorf(
				"table.ResidentTaxTable: %d: %q has no 均等割 for that year; the first row of data/law/%s/%s.tsv starts later",
				y, calendar.Municipality, calendar.Municipality, law.ResidentPerCapitaTableName)

		case !ok:
			rows = append(rows, relation.Row[ResidentTaxRow]{Year: y, Value: row})
			continue
		}

		row.TotalIncome = judged.TotalIncome
		row.Deductions = judged.Deductions.Total().Resident
		row.Taxable = max(row.TotalIncome-row.Deductions, 0).TruncateTaxableIncome()

		row.Liable, _ = in.Liable.At(y)

		if row.Liable.PerCapita {
			row.PerCapita = perCapita.Total()
			row.ForestEnvironmentTax = in.Tables.ForestEnvironmentTaxAt(y)
		}
		if row.Liable.Income {
			levy := law.ResidentTaxOf(row.Taxable, rates)
			adjustment := law.AdjustmentCredit(row.Taxable, judged.Deductions.HumanDeductionGap(), rates)

			special := law.SplitResidentSpecialTaxCredit(
				residentSpecialCredit(y, judged),
				max(levy.Prefectural-adjustment.Prefectural, 0),
				max(levy.Municipal-adjustment.Municipal, 0))

			charged := levy.Less(law.ResidentTaxCredits{
				Prefectural: adjustment.Prefectural + special.Prefectural,
				Municipal:   adjustment.Municipal + special.Municipal,
			})

			row.Adjustment = adjustment.Total()
			row.SpecialCredit = special.Total()
			row.PrefecturalIncome = charged.Prefectural
			row.MunicipalIncome = charged.Municipal
		}
		row.Total = row.PerCapita + row.ForestEnvironmentTax +
			row.PrefecturalIncome + row.MunicipalIncome

		rows = append(rows, relation.Row[ResidentTaxRow]{Year: y, Value: row})
	}

	return relation.New(rows), nil
}

func residentSpecialCredit(levyYear date.Year, judged IncomeTaxRow) money.Yen {
	incomeYear := date.Year(levyYear - 1)

	if levyYear != law.ResidentSpecialTaxCreditLevyYear &&
		levyYear != law.ResidentSpecialTaxCreditSpouseLevyYear {
		return 0
	}

	sameLivelihoodSpouse := judged.Deductions.SpouseSameLivelihood
	deductibleSpouse := sameLivelihoodSpouse &&
		law.TaxpayerMayClaimASpouse(judged.TotalIncome, incomeYear)

	deductibleDependants := judged.Deductions.RelativeCount
	if deductibleSpouse {
		deductibleDependants++
	}

	return law.ResidentSpecialTaxCredit(levyYear, judged.TotalIncome, deductibleDependants) +
		law.ResidentSpecialTaxCreditForSpouse(levyYear, judged.TotalIncome,
			sameLivelihoodSpouse && !deductibleSpouse)
}
