package actuals

import (
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	ResidentTaxYearColumn         tsv.ColumnName = "課税年度"
	ResidentTaxMunicipalityColumn tsv.ColumnName = "自治体"

	ResidentTaxTotalIncomeColumn tsv.ColumnName = "合計所得金額[円]"
	ResidentTaxDeductionsColumn  tsv.ColumnName = "所得控除合計[円]"
	ResidentTaxTaxableColumn     tsv.ColumnName = "課税標準額[円]"

	ResidentTaxMunicipalGrossColumn      tsv.ColumnName = "市税額控除前所得割額[円]"
	ResidentTaxMunicipalAdjustmentColumn tsv.ColumnName = "市調整控除額[円]"
	ResidentTaxMunicipalDonationColumn   tsv.ColumnName = "市寄附金税額控除額[円]"
	ResidentTaxMunicipalFlatCutColumn    tsv.ColumnName = "市定額減税額[円]"
)

type ResidentTaxRecord struct {
	TaxYear, IncomeYear date.Year

	Municipality string

	TotalIncome, Deductions, Taxable money.Yen

	MunicipalGross money.Yen

	MunicipalAdjustment, MunicipalDonation, MunicipalFlatCut money.Yen

	MunicipalLevied money.Yen

	PrefecturalLevied money.Yen

	MunicipalPoll, PrefecturalPoll money.Yen
}

func (r ResidentTaxRecord) PerCapita() money.Yen { return r.MunicipalPoll + r.PrefecturalPoll }

func (r ResidentTaxRecord) Charged() money.Yen {
	return r.MunicipalLevied + r.PrefecturalLevied + r.PerCapita()
}

func ResidentTaxRecords(residentTax *tsv.Table) ([]ResidentTaxRecord, error) {
	r, err := tsv.NewReader(residentTax, ResidentTaxPath,
		ResidentTaxYearColumn, ResidentTaxIncomeYearColumn, ResidentTaxMunicipalityColumn,
		ResidentTaxTotalIncomeColumn, ResidentTaxDeductionsColumn, ResidentTaxTaxableColumn,
		ResidentTaxMunicipalGrossColumn, ResidentTaxMunicipalAdjustmentColumn,
		ResidentTaxMunicipalDonationColumn, ResidentTaxMunicipalFlatCutColumn,
		ResidentTaxMunicipalIncomeColumn, ResidentTaxMunicipalPollColumn,
		ResidentTaxPrefecturalIncomeColumn, ResidentTaxPrefecturalPollColumn)
	if err != nil {
		return nil, err
	}

	records := make([]ResidentTaxRecord, 0, r.Rows())
	for row := range r.Rows() {
		year := func(column tsv.ColumnName) (date.Year, error) {
			y, err := date.ParseYear(r.Field(row, column))
			if err != nil {
				return 0, r.Errorf(row, column, "%v", err)
			}
			return y, nil
		}
		yen := func(column tsv.ColumnName) (money.Yen, error) {
			v, err := money.ParseYen(r.Field(row, column))
			if err != nil {
				return 0, r.Errorf(row, column, "%v", err)
			}
			return v, nil
		}

		var record ResidentTaxRecord
		record.Municipality = r.Field(row, ResidentTaxMunicipalityColumn)

		for _, field := range []struct {
			into   *date.Year
			column tsv.ColumnName
		}{
			{&record.TaxYear, ResidentTaxYearColumn},
			{&record.IncomeYear, ResidentTaxIncomeYearColumn},
		} {
			if *field.into, err = year(field.column); err != nil {
				return nil, err
			}
		}

		for _, field := range []struct {
			into   *money.Yen
			column tsv.ColumnName
		}{
			{&record.TotalIncome, ResidentTaxTotalIncomeColumn},
			{&record.Deductions, ResidentTaxDeductionsColumn},
			{&record.Taxable, ResidentTaxTaxableColumn},
			{&record.MunicipalGross, ResidentTaxMunicipalGrossColumn},
			{&record.MunicipalAdjustment, ResidentTaxMunicipalAdjustmentColumn},
			{&record.MunicipalDonation, ResidentTaxMunicipalDonationColumn},
			{&record.MunicipalFlatCut, ResidentTaxMunicipalFlatCutColumn},
			{&record.MunicipalLevied, ResidentTaxMunicipalIncomeColumn},
			{&record.MunicipalPoll, ResidentTaxMunicipalPollColumn},
			{&record.PrefecturalLevied, ResidentTaxPrefecturalIncomeColumn},
			{&record.PrefecturalPoll, ResidentTaxPrefecturalPollColumn},
		} {
			if *field.into, err = yen(field.column); err != nil {
				return nil, err
			}
		}

		records = append(records, record)
	}
	return records, nil
}

func ResidentTaxRecordsByIncomeYear(residentTax *tsv.Table) (map[date.Year]ResidentTaxRecord, error) {
	records, err := ResidentTaxRecords(residentTax)
	if err != nil {
		return nil, err
	}

	byYear := make(map[date.Year]ResidentTaxRecord, len(records))
	for _, record := range records {
		byYear[record.IncomeYear] = record
	}
	return byYear, nil
}
