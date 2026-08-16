package plan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/config"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	Earner table.PersonName = "夫"
	Spouse table.PersonName = "妻"
)

type Plan struct {
	Calendar        relation.Table[table.CalendarRow]
	Loan            relation.Table[table.LoanYear]
	Income          map[table.PersonName]relation.Table[table.IncomeRow]
	SocialInsurance relation.Table[table.SocialInsuranceTotalRow]

	HouseholdInsurance relation.Table[table.HouseholdInsuranceRow]
	ChildAllowance     relation.Table[table.ChildAllowanceRow]
	Expense            relation.Table[table.ExpenseRow]
	IncomeTax          map[table.PersonName]relation.Table[table.IncomeTaxRow]
	ResidentTax        map[table.PersonName]relation.Table[table.ResidentTaxRow]
	PropertyTax        relation.Table[table.PropertyTaxRow]
	Tax                relation.Table[table.TaxTotalRow]
	Timeline           relation.Table[table.TimelineRow]
	Assets             relation.Table[table.AssetsRow]

	PriceLevels relation.Table[money.Factor]

	levelsByItem map[input.PricedItem]relation.Table[money.Factor]
	mutualAid    relation.Table[money.Yen]

	LastResort table.LastResort

	Outturn relation.Table[actuals.Outturn]

	Uncompared []date.Year
}

type Sources struct {
	Root string

	ProjectPath string

	SlotOverrides map[tsv.Slot]string
}

func Build(sources Sources) (*Plan, error) {
	loaded, err := Load(sources)
	if err != nil {
		return nil, err
	}
	return loaded.Build()
}

type Input struct {
	sources Sources

	paths map[tsv.Slot]string

	tables   map[tsv.Slot]*tsv.Table
	statutes statutes

	cash relation.Table[actuals.CashTakeHomeRow]

	payslips []actuals.Payslip

	remuneration []actuals.Remuneration

	pensionRecord []actuals.PensionRecordEntry

	balance actuals.BalanceTable
}

func (in *Input) Root() string { return in.sources.Root }

func (in *Input) SlotPaths() map[tsv.Slot]string {
	paths := make(map[tsv.Slot]string, len(in.paths))
	for slot, path := range in.paths {
		paths[slot] = path
	}
	return paths
}

func (in *Input) UnreadColumns() map[tsv.Slot][]tsv.ColumnName {
	return input.UnreadColumnsOf(in.tables)
}

// everySlotWasRead refuses a manifest that names a table which is not there.
//
// input.Load skips a slot whose file does not exist, because the check
// (-allow-missing) has to run on input that is still being written. **A plan
// has no such licence**: a missing table would be read as nil further down and
// the failure would surface as a panic, far from the manifest that named it.
func everySlotWasRead(paths map[tsv.Slot]string, tables map[tsv.Slot]*tsv.Table) error {
	slots := make([]tsv.Slot, 0, len(paths))
	for slot := range paths {
		if _, ok := tables[slot]; !ok {
			slots = append(slots, slot)
		}
	}
	if len(slots) == 0 {
		return nil
	}
	slices.Sort(slots)

	var said []string
	for _, slot := range slots {
		said = append(said, fmt.Sprintf("%s = %s", slot, paths[slot]))
	}
	return fmt.Errorf(
		"plan.Load: マニフェストが名指した表がそこに無い: %s。"+
			"**検査は書きかけの入力を許すが、計画は許さない**——"+
			"無い表はこの先で nil として読まれ、マニフェストから遠いところで落ちる",
		strings.Join(said, "、"))
}

func Load(sources Sources) (*Input, error) {
	paths, err := config.SlotPaths(sources.ProjectPath, sources.SlotOverrides)
	if err != nil {
		return nil, err
	}
	tables, err := input.Load(sources.Root, paths)
	if err != nil {
		return nil, err
	}
	if err := everySlotWasRead(paths, tables); err != nil {
		return nil, err
	}

	lawFS := os.DirFS(filepath.Join(sources.Root, filepath.FromSlash(law.LawDirectory)))
	statutes, err := loadStatutes(lawFS)
	if err != nil {
		return nil, err
	}
	if err := statutes.loadRegional(lawFS, tables); err != nil {
		return nil, err
	}

	read, ok := tables[input.BalanceSlot]
	if !ok {
		return nil, fmt.Errorf(
			"plan: nothing fills the slot %q, and a projection has nowhere to start from without it",
			input.BalanceSlot)
	}
	balance, err := actuals.ParseBalanceTable(read)
	if err != nil {
		return nil, err
	}

	cash, err := actuals.CashTakeHomeUnder(sources.Root)
	if err != nil {
		return nil, err
	}
	payslips, err := actuals.PayslipsUnder(os.DirFS(sources.Root))
	if err != nil {
		return nil, err
	}
	remuneration, err := actuals.RemunerationRecord(os.DirFS(sources.Root))
	if err != nil {
		return nil, err
	}
	pensionRecord, err := actuals.PensionRecordOf(os.DirFS(sources.Root))
	if err != nil {
		return nil, err
	}

	return &Input{
		sources: sources, paths: paths, tables: tables, statutes: statutes,
		balance: balance, cash: cash, payslips: payslips, remuneration: remuneration,
		pensionRecord: pensionRecord,
	}, nil
}

func (in *Input) StartsAfter() (date.Year, error) {
	year, _, ok := in.balance.Latest()
	if !ok {
		return 0, fmt.Errorf("plan: the actuals carry no year to start from")
	}
	return year, nil
}

func (in *Input) With(slot tsv.Slot, table *tsv.Table) *Input {
	tables := make(map[tsv.Slot]*tsv.Table, len(in.tables))
	for name, t := range in.tables {
		tables[name] = t
	}
	tables[slot] = table

	next := *in
	next.tables = tables
	return &next
}

func (in *Input) Table(slot tsv.Slot) (*tsv.Table, error) {
	table, ok := in.tables[slot]
	if !ok {
		return nil, fmt.Errorf("plan: no table fills the slot %q", slot)
	}
	return table, nil
}

func (in *Input) Build() (*Plan, error) {
	return build(in)
}

type statutes struct {
	healthGrades, pensionGrades law.StandardRemunerationTable
	socialRates                 law.SocialInsuranceRates
	employment                  law.EmploymentInsuranceTable
	childcareLeave              law.ChildcareLeaveBenefitTable
	childAllowance              law.ChildAllowanceTable
	disability                  law.DisabilityDeductionTable
	spouseCeiling               law.SpouseIncomeCeilingTable
	pensionRevaluation          law.PensionRevaluationRates
	housingLoan                 law.HousingLoanCreditTable
	depreciation                law.DepreciationRateTable
	kokuho                      law.KokuhoTable
	kouki                       law.KoukiRatesTable
	nationalPension             law.NationalPensionPremiumTable
	basicPensionFull            law.YearYenTable
	supplementaryPension        money.Yen
	supplementarySpecial        money.Yen
	nursingCare                 law.NursingCarePremiumTable
	residentTax                 law.ResidentLevies
	propertyTax                 law.PropertyTaxTable
}

func loadStatutes(fsys fs.FS) (statutes, error) {
	var s statutes

	read := func(name string) (*tsv.Table, error) {
		f, err := fsys.Open(name + ".tsv")
		if err != nil {
			return nil, fmt.Errorf("plan: %w", err)
		}
		defer f.Close()
		return tsv.Read(f)
	}

	grades := func(name string) (law.StandardRemunerationTable, error) {
		table, err := read(name)
		if err != nil {
			return law.StandardRemunerationTable{}, err
		}
		return law.ParseStandardRemunerationTable(table)
	}

	var err error
	if s.healthGrades, err = grades(law.StandardRemunerationHealthTableName); err != nil {
		return s, err
	}
	if s.pensionGrades, err = grades(law.StandardRemunerationPensionTableName); err != nil {
		return s, err
	}

	for _, load := range []struct {
		name string
		into func(*tsv.Table) error
	}{
		{law.HealthInsuranceRateTableName, func(t *tsv.Table) (err error) {
			s.socialRates, err = law.ParseSocialInsuranceRates(t)
			return
		}},
		{law.EmploymentInsuranceRateTableName, func(t *tsv.Table) error {
			rates, err := law.ParseYearRateTable(t, law.EmploymentInsuranceRateTableName, law.EmploymentInsuranceRateColumn)
			s.employment = law.EmploymentInsuranceTable{YearRateTable: rates}
			return err
		}},
		{law.ChildcareLeaveBenefitTableName, func(t *tsv.Table) (err error) {
			s.childcareLeave, err = law.ParseChildcareLeaveBenefitTable(t)
			return
		}},
		{law.ChildAllowanceLimitsTableName, func(t *tsv.Table) (err error) {
			s.childAllowance, err = law.ParseChildAllowanceTable(t)
			return
		}},
		{law.DisabilityDeductionTableName, func(t *tsv.Table) (err error) {
			s.disability, err = law.ParseDisabilityDeductionTable(t)
			return
		}},
		{law.PensionRevaluationRateTableName, func(t *tsv.Table) (err error) {
			s.pensionRevaluation, err = law.ParsePensionRevaluationRates(t)
			return
		}},
		{law.SpouseIncomeCeilingTableName, func(t *tsv.Table) (err error) {
			s.spouseCeiling, err = law.ParseSpouseIncomeCeilingTable(t)
			return
		}},
		{law.HousingLoanCreditTableName, func(t *tsv.Table) (err error) {
			s.housingLoan, err = law.ParseHousingLoanCreditTable(t)
			return
		}},
		{law.DepreciationRateTableName, func(t *tsv.Table) (err error) {
			s.depreciation, err = law.ParseDepreciationRateTable(t)
			return
		}},
		{law.NationalPensionPremiumTableName, func(t *tsv.Table) error {
			monthly, err := law.ParseYearYenTable(t, law.NationalPensionPremiumTableName, law.NationalPensionPremiumColumn)
			s.nationalPension = law.NationalPensionPremiumTable{YearYenTable: monthly}
			return err
		}},
		{law.BasicPensionFullTableName, func(t *tsv.Table) (err error) {
			s.basicPensionFull, err = law.ParseYearYenTable(t, law.BasicPensionFullTableName, law.BasicPensionFullColumn)
			return
		}},
		{law.SupplementaryPensionTableName, func(t *tsv.Table) error {
			amount, err := law.ParseYearYenTable(t, law.SupplementaryPensionTableName, law.SpouseSupplementColumn)
			if err != nil {
				return err
			}
			special, err := law.ParseYearYenTable(t, law.SupplementaryPensionTableName, law.SpouseSupplementSpecialColumn)
			if err != nil {
				return err
			}
			s.supplementaryPension = amount.Amount(pensionTableYear)
			s.supplementarySpecial = special.Amount(pensionTableYear)
			return nil
		}},
	} {
		table, err := read(load.name)
		if err != nil {
			return s, err
		}
		if err := load.into(table); err != nil {
			return s, fmt.Errorf("plan: %s: %w", load.name, err)
		}
	}

	return s, nil
}

func (s *statutes) loadRegional(fsys fs.FS, tables map[tsv.Slot]*tsv.Table) error {
	lived, err := table.MunicipalitiesFrom(tables)
	if err != nil {
		return err
	}
	if len(lived) == 0 {
		return fmt.Errorf("plan: 住所が 1 年も書かれていないので、どの自治体の規則によるかが決まらない")
	}

	if s.residentTax, err = law.LoadResidentLevies(fsys, lived...); err != nil {
		return err
	}

	last := lived[len(lived)-1]
	if s.kokuho, err = law.LoadKokuhoTable(fsys, last); err != nil {
		return err
	}
	if s.propertyTax, err = law.LoadPropertyTaxTable(fsys, last); err != nil {
		return err
	}

	if s.nursingCare, err = law.LoadNursingCarePremiumTable(fsys, last); err != nil {
		return err
	}

	municipalities, err := law.LoadMunicipalities(fsys)
	if err != nil {
		return err
	}
	from, to, err := input.PlanSpan(tables[input.PlanSlot])
	if err != nil {
		return err
	}
	prefecture, err := municipalities.PrefectureOf(last, from)
	if err != nil {
		return err
	}

	if until, err := municipalities.PrefectureOf(last, to); err != nil {
		return err
	} else if until != prefecture {
		return fmt.Errorf(
			"plan: %s は %d 年に %s、%d 年に %s である。後期高齢者医療は県ごとの広域連合が課すので、"+
				"計画の間で県が変わる住所は 1 つの料率表では読めない",
			last, from, prefecture, to, until)
	}

	s.kouki, err = law.LoadKoukiRatesTable(fsys, prefecture)
	return err
}
