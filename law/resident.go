package law

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	ResidentPrefecturalRateColumn tsv.ColumnName = "県民税率"
	ResidentMunicipalRateColumn   tsv.ColumnName = "市民税率"

	ResidentDesignatedCityColumn tsv.ColumnName = "指定都市"
)

const (
	ResidentPrefecturalPerCapitaColumn tsv.ColumnName = "県民税均等割[円]"
	ResidentMunicipalPerCapitaColumn   tsv.ColumnName = "市民税均等割[円]"
)

type DesignatedCityField string

const DesignatedCityYes DesignatedCityField = "はい"

type ResidentRates struct {
	PrefecturalRate money.Rate
	MunicipalRate   money.Rate

	DesignatedCity bool
}

type ResidentRateTables = YearTable[ResidentRates]

type ResidentPerCapitaLevy struct {
	Prefectural money.Yen
	Municipal   money.Yen
}

func (l ResidentPerCapitaLevy) Total() money.Yen {
	return l.Prefectural + l.Municipal
}

type ResidentPerCapitaTables = YearTable[ResidentPerCapitaLevy]

const (
	ResidentRateTableName      = "resident-tax-rate"
	ResidentPerCapitaTableName = "resident-tax-per-capita"
)

func ResidentTableNames() []string {
	return []string{ResidentRateTableName, ResidentPerCapitaTableName, ResidentExemptionTableName}
}

func ParseResidentRateTables(table *tsv.Table) (ResidentRateTables, error) {
	r, err := newReader(table, ResidentRateTableName,
		ResidentPrefecturalRateColumn, ResidentMunicipalRateColumn,
		ResidentDesignatedCityColumn, LawStartYearColumn, LawEndYearColumn)
	if err != nil {
		return ResidentRateTables{}, fmt.Errorf("law.ParseResidentRateTables: %w", err)
	}

	periods, err := readYearPeriods(r, func(row int) (ResidentRates, error) {
		var parsed ResidentRates
		if parsed.PrefecturalRate, err = r.Percent(row, ResidentPrefecturalRateColumn); err != nil {
			return parsed, err
		}
		if parsed.MunicipalRate, err = r.Percent(row, ResidentMunicipalRateColumn); err != nil {
			return parsed, err
		}
		parsed.DesignatedCity = DesignatedCityField(r.Field(row, ResidentDesignatedCityColumn)) == DesignatedCityYes
		return parsed, nil
	})
	if err != nil {
		return ResidentRateTables{}, fmt.Errorf("law.ParseResidentRateTables: %w", err)
	}

	built, err := NewYearTableOfPeriods(periods)
	if err != nil {
		return ResidentRateTables{}, fmt.Errorf("law.ParseResidentRateTables: %w", err)
	}
	return built, nil
}

func ParseResidentPerCapitaTables(table *tsv.Table) (ResidentPerCapitaTables, error) {
	r, err := newReader(table, ResidentPerCapitaTableName,
		ResidentPrefecturalPerCapitaColumn, ResidentMunicipalPerCapitaColumn,
		LawStartYearColumn, LawEndYearColumn)
	if err != nil {
		return ResidentPerCapitaTables{}, fmt.Errorf("law.ParseResidentPerCapitaTables: %w", err)
	}

	periods, err := readYearPeriods(r, func(row int) (ResidentPerCapitaLevy, error) {
		var parsed ResidentPerCapitaLevy
		if parsed.Prefectural, err = r.Yen(row, ResidentPrefecturalPerCapitaColumn); err != nil {
			return parsed, err
		}
		if parsed.Municipal, err = r.Yen(row, ResidentMunicipalPerCapitaColumn); err != nil {
			return parsed, err
		}
		return parsed, nil
	})
	if err != nil {
		return ResidentPerCapitaTables{}, fmt.Errorf("law.ParseResidentPerCapitaTables: %w", err)
	}

	built, err := NewYearTableOfPeriods(periods)
	if err != nil {
		return ResidentPerCapitaTables{}, fmt.Errorf("law.ParseResidentPerCapitaTables: %w", err)
	}
	return built, nil
}

func LoadResidentRateTables(fsys fs.FS, municipality Municipality) (ResidentRateTables, error) {
	table, err := LoadRegionalTable(fsys, string(municipality), ResidentRateTableName)
	if err != nil {
		return ResidentRateTables{}, err
	}
	return ParseResidentRateTables(table)
}

func LoadResidentPerCapitaTables(fsys fs.FS, municipality Municipality) (ResidentPerCapitaTables, error) {
	table, err := LoadRegionalTable(fsys, string(municipality), ResidentPerCapitaTableName)
	if err != nil {
		return ResidentPerCapitaTables{}, err
	}
	return ParseResidentPerCapitaTables(table)
}

type ResidentTax struct {
	PerCapita money.Yen

	Prefectural money.Yen
	Municipal   money.Yen
}

func (r ResidentTax) Total() money.Yen {
	return r.PerCapita + r.Prefectural + r.Municipal
}

func ResidentTaxOf(taxableIncome money.Yen, m ResidentRates) ResidentTax {
	taxable := taxableIncome.TruncateTaxableIncome()
	if taxable <= 0 {
		return ResidentTax{}
	}
	return ResidentTax{
		Prefectural: taxable.Mul(m.PrefecturalRate, money.Truncate),
		Municipal:   taxable.Mul(m.MunicipalRate, money.Truncate),
	}
}

type ResidentTaxCredits struct {
	Prefectural, Municipal money.Yen
}

func (c ResidentTaxCredits) Total() money.Yen {
	return c.Prefectural + c.Municipal
}

func (r ResidentTax) Less(credits ResidentTaxCredits) ResidentTax {
	r.Prefectural = max(r.Prefectural-credits.Prefectural, 0).TruncateIncomeTax()
	r.Municipal = max(r.Municipal-credits.Municipal, 0).TruncateIncomeTax()
	return r
}

const BasicHumanDeductionGap money.Yen = 50_000

const AdjustmentCreditFloor money.Yen = 2_500

const AdjustmentCreditThreshold money.Yen = 2_000_000

func AdjustmentCredit(taxableIncome, humanDeductionGap money.Yen, m ResidentRates) ResidentTaxCredits {
	taxable := taxableIncome.TruncateTaxableIncome()
	if taxable <= 0 {
		return ResidentTaxCredits{}
	}

	base := min(humanDeductionGap, taxable)
	if taxable > AdjustmentCreditThreshold {
		base = humanDeductionGap - (taxable - AdjustmentCreditThreshold)
	}

	total := base.Mul(money.NewRate(5, 100), money.Truncate)
	if taxable > AdjustmentCreditThreshold {
		total = max(total, AdjustmentCreditFloor)
	}
	if total <= 0 {
		return ResidentTaxCredits{}
	}

	share := money.NewRate(3, 5)
	if m.DesignatedCity {
		share = money.NewRate(4, 5)
	}

	municipal := total.Mul(share, money.Truncate)
	return ResidentTaxCredits{Municipal: municipal, Prefectural: total - municipal}
}

type ResidentLevies struct {
	byMunicipality map[Municipality]ResidentMunicipality
	forest         YearYenTable
}

type ResidentMunicipality struct {
	name Municipality

	rates      ResidentRateTables
	perCapita  ResidentPerCapitaTables
	exemptions ResidentExemptions
}

func (m ResidentMunicipality) Rates() ResidentRateTables { return m.rates }

func (m ResidentMunicipality) PerCapita() ResidentPerCapitaTables { return m.perCapita }

func (m ResidentMunicipality) Exemptions() ResidentExemptions { return m.exemptions }

func (s ResidentLevies) TablesFor(municipality Municipality) (ResidentMunicipality, bool) {
	tables, ok := s.byMunicipality[municipality]
	return tables, ok
}

func (s ResidentLevies) ForestEnvironmentTaxAt(year date.Year) money.Yen {
	return s.forest.Amount(year)
}

func LoadResidentLevies(fsys fs.FS, municipalities ...Municipality) (ResidentLevies, error) {
	if len(municipalities) == 0 {
		return ResidentLevies{}, fmt.Errorf("law.LoadResidentLevies: 自治体が 1 つも指定されていない")
	}

	levies := ResidentLevies{byMunicipality: make(map[Municipality]ResidentMunicipality, len(municipalities))}
	for _, municipality := range municipalities {
		rates, err := LoadResidentRateTables(fsys, municipality)
		if err != nil {
			return ResidentLevies{}, err
		}
		perCapita, err := LoadResidentPerCapitaTables(fsys, municipality)
		if err != nil {
			return ResidentLevies{}, err
		}
		exemptions, err := LoadResidentExemptions(fsys, municipality)
		if err != nil {
			return ResidentLevies{}, err
		}
		levies.byMunicipality[municipality] = ResidentMunicipality{
			name: municipality, rates: rates, perCapita: perCapita, exemptions: exemptions,
		}
	}

	forest, err := LoadForestEnvironmentTaxTable(fsys)
	if err != nil {
		return ResidentLevies{}, err
	}
	levies.forest = forest
	return levies, nil
}

func MustLoadResidentLevies(t testingT, fsys fs.FS, municipalities ...Municipality) ResidentLevies {
	t.Helper()

	levies, err := LoadResidentLevies(fsys, municipalities...)
	if err != nil {
		t.Fatalf("law.LoadResidentLevies: %v", err)
	}
	return levies
}

func (s ResidentLevies) MustTablesFor(t testingT, municipality Municipality) ResidentMunicipality {
	t.Helper()

	tables, ok := s.TablesFor(municipality)
	if !ok {
		t.Fatalf("law: %s の住民税の表が読み込まれていない", municipality)
	}
	return tables
}

func (m ResidentMunicipality) MustAt(t testingT, year date.Year) (ResidentRates, ResidentPerCapitaLevy, ResidentExemption) {
	t.Helper()

	rates, hasRates := m.rates.At(year)
	perCapita, hasPerCapita := m.perCapita.At(year)
	exemption, hasExemption := m.exemptions.At(year)

	var missing []string
	for _, table := range []struct {
		name string
		ok   bool
	}{
		{ResidentRateTableName, hasRates},
		{ResidentPerCapitaTableName, hasPerCapita},
		{ResidentExemptionTableName, hasExemption},
	} {
		if !table.ok {
			missing = append(missing, table.name)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("law: %s の %s に 課税年度%d の行が無い", m.name, strings.Join(missing, "・"), year)
	}
	return rates, perCapita, exemption
}

type testingT interface {
	Helper()
	Fatalf(format string, args ...any)
}
