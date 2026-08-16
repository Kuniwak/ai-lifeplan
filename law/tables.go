package law

import (
	"fmt"
	"io/fs"
	"path"
	"slices"

	"github.com/Kuniwak/lifeplan/tsv"
)

type Shape struct {
	Name string

	BandColumn tsv.ColumnName

	DomainMin int64

	Regional bool

	KeyColumns []tsv.ColumnName

	StartColumn tsv.ColumnName
}

func (s Shape) Keys() []tsv.ColumnName {
	if len(s.KeyColumns) > 0 {
		return s.KeyColumns
	}
	if s.BandColumn != "" {
		return []tsv.ColumnName{s.Start(), s.BandColumn}
	}
	return []tsv.ColumnName{s.Start()}
}

func (s Shape) Start() tsv.ColumnName {
	if s.StartColumn != "" {
		return s.StartColumn
	}
	return LawStartYearColumn
}

func Shapes() []Shape {
	return []Shape{
		{Name: StandardRemunerationHealthTableName, BandColumn: StandardRemunerationLowerColumn},
		{Name: StandardRemunerationPensionTableName, BandColumn: StandardRemunerationLowerColumn},
		{Name: DepreciationRateTableName, BandColumn: DepreciationYearsColumn, DomainMin: 1},
		{Name: ChildAllowanceLimitsTableName, BandColumn: ChildAllowanceDependentsColumn},

		{Name: NationalPensionPremiumTableName},
		{Name: BasicPensionFullTableName},
		{Name: SupplementaryPensionTableName},
		{Name: EmploymentInsuranceRateTableName},
		{Name: HealthInsuranceRateTableName},
		{Name: DisabilityDeductionTableName, KeyColumns: []tsv.ColumnName{DisabilityCategoryColumn}},
		{Name: SpouseIncomeCeilingTableName},
		{Name: PensionRevaluationRateTableName,
			KeyColumns: []tsv.ColumnName{RevaluationTableYearColumn, LawStartYearColumn}},
		{Name: ChildcareLeaveBenefitTableName,
			KeyColumns: []tsv.ColumnName{LawStartYearColumn, ChildcareLeaveRateColumn}},
		{Name: HousingLoanCreditTableName, StartColumn: HousingLoanFromYearColumn},
		{Name: ForestEnvironmentTaxTableName},
		{Name: MunicipalitiesTableName,
			KeyColumns: []tsv.ColumnName{MunicipalityColumn, LawStartYearColumn}},

		{Name: ResidentRateTableName, Regional: true},
		{Name: ResidentPerCapitaTableName, Regional: true},
		{Name: ResidentExemptionTableName, Regional: true},
		{Name: KoukiRatesTableName, Regional: true,
			KeyColumns: []tsv.ColumnName{LawStartYearColumn, KoukiPartColumn}},
		{Name: KokuhoTableName, Regional: true,
			KeyColumns: []tsv.ColumnName{LawStartYearColumn, KokuhoPartColumn}},
		{Name: NursingCarePremiumTableName, Regional: true,
			KeyColumns: []tsv.ColumnName{LawStartYearColumn, NursingCareStageColumn}},
		{Name: PropertyTaxTableName, Regional: true},
	}
}

func LoadShape(fsys fs.FS, shape Shape, region string) (*tsv.Table, error) {
	name := shape.Name
	if shape.Regional {
		if region == "" {
			return nil, fmt.Errorf("law.LoadShape: %s is regional, so a region has to be named", shape.Name)
		}
		name = path.Join(region, shape.Name)
	}

	f, err := fsys.Open(name + ".tsv")
	if err != nil {
		return nil, fmt.Errorf("law.LoadShape: %w", err)
	}
	defer f.Close()

	table, err := tsv.Read(f)
	if err != nil {
		return nil, fmt.Errorf("law.LoadShape: %s: %w", name, err)
	}
	return table, nil
}

func RegionalTableNames() []string {
	var names []string
	for _, shape := range Shapes() {
		if shape.Regional {
			names = append(names, shape.Name)
		}
	}
	slices.Sort(names)
	return names
}

func ShapeNamed(name string) (Shape, bool) {
	for _, shape := range Shapes() {
		if shape.Name == name {
			return shape, true
		}
	}
	return Shape{}, false
}

func MustLoadTable(t testingT, root fs.FS, name string) *tsv.Table {
	t.Helper()
	return mustLoadShape(t, root, name, "")
}

func MustLoadRegionalTable(t testingT, root fs.FS, region Municipality, name string) *tsv.Table {
	t.Helper()
	if region == "" {
		t.Fatalf("law.MustLoadRegionalTable(%q): 自治体が空である", name)
		return nil
	}
	return mustLoadShape(t, root, name, string(region))
}

func mustLoadShape(t testingT, root fs.FS, name, region string) *tsv.Table {
	t.Helper()

	shape, registered := ShapeNamed(name)
	if !registered {
		t.Fatalf("law: %q という表は law.Shapes() に無い", name)
		return nil
	}
	switch {
	case shape.Regional && region == "":
		t.Fatalf("law.MustLoadTable(%q): この表は自治体ごとにある。law.MustLoadRegionalTable を使うこと", name)
		return nil
	case !shape.Regional && region != "":
		t.Fatalf("law.MustLoadRegionalTable(%q): この表は全国で 1 つである。law.MustLoadTable を使うこと", name)
		return nil
	}

	table, err := LoadShape(root, shape, region)
	if err != nil {
		t.Fatalf("law.MustLoadTable(%q): %v", name, err)
		return nil
	}
	return table
}

func mustLoad[T any](t testingT, root fs.FS, name string, parse func(*tsv.Table) (T, error)) T {
	t.Helper()

	parsed, err := parse(MustLoadTable(t, root, name))
	if err != nil {
		var zero T
		t.Fatalf("law: %s を読めない: %v", name, err)
		return zero
	}
	return parsed
}

func MustLoadDepreciationRates(t testingT, root fs.FS) DepreciationRateTable {
	t.Helper()
	return mustLoad(t, root, DepreciationRateTableName, ParseDepreciationRateTable)
}

func MustLoadChildcareLeaveBenefits(t testingT, root fs.FS) ChildcareLeaveBenefitTable {
	t.Helper()
	return mustLoad(t, root, ChildcareLeaveBenefitTableName, ParseChildcareLeaveBenefitTable)
}

func MustLoadNationalPensionPremiums(t testingT, root fs.FS) YearYenTable {
	t.Helper()
	return mustLoad(t, root, NationalPensionPremiumTableName, func(table *tsv.Table) (YearYenTable, error) {
		return ParseYearYenTable(table, NationalPensionPremiumTableName, NationalPensionPremiumColumn)
	})
}

func MustLoadBasicPensionFullAmounts(t testingT, root fs.FS) YearYenTable {
	t.Helper()
	return mustLoad(t, root, BasicPensionFullTableName, func(table *tsv.Table) (YearYenTable, error) {
		return ParseYearYenTable(table, BasicPensionFullTableName, BasicPensionFullColumn)
	})
}

func MustLoadChildAllowanceLimits(t testingT, root fs.FS) ChildAllowanceTable {
	t.Helper()
	return mustLoad(t, root, ChildAllowanceLimitsTableName, ParseChildAllowanceTable)
}

func MustLoadSpouseIncomeCeilings(t testingT, root fs.FS) SpouseIncomeCeilingTable {
	t.Helper()
	return mustLoad(t, root, SpouseIncomeCeilingTableName, ParseSpouseIncomeCeilingTable)
}

func MustLoadDisabilityDeductions(t testingT, root fs.FS) DisabilityDeductionTable {
	t.Helper()
	return mustLoad(t, root, DisabilityDeductionTableName, ParseDisabilityDeductionTable)
}

func MustLoadHousingLoanCredits(t testingT, root fs.FS) HousingLoanCreditTable {
	t.Helper()
	return mustLoad(t, root, HousingLoanCreditTableName, ParseHousingLoanCreditTable)
}

func MustLoadStandardRemunerations(t testingT, root fs.FS, name string) StandardRemunerationTable {
	t.Helper()
	return mustLoad(t, root, name, ParseStandardRemunerationTable)
}

func MustLoadPensionRevaluationRates(t testingT, root fs.FS) PensionRevaluationRates {
	t.Helper()
	return mustLoad(t, root, PensionRevaluationRateTableName, ParsePensionRevaluationRates)
}

func MustLoadSocialInsuranceRates(t testingT, root fs.FS) SocialInsuranceRates {
	t.Helper()
	return mustLoad(t, root, HealthInsuranceRateTableName, ParseSocialInsuranceRates)
}

func MustLoadEmploymentInsuranceRates(t testingT, root fs.FS) YearRateTable {
	t.Helper()
	return mustLoad(t, root, EmploymentInsuranceRateTableName, func(table *tsv.Table) (YearRateTable, error) {
		return ParseYearRateTable(table, EmploymentInsuranceRateTableName, EmploymentInsuranceRateColumn)
	})
}
