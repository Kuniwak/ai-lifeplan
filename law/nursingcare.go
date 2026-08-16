package law

import (
	"fmt"
	"io/fs"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

const NursingCareFirstCategoryAge = 65

const NursingCarePremiumTableName = "nursing-care-premium"

const (
	NursingCareStageColumn          tsv.ColumnName = "段階"
	NursingCareTaxedColumn          tsv.ColumnName = "本人の市民税"
	NursingCareHouseholdTaxedColumn tsv.ColumnName = "世帯の市民税課税者"
	NursingCareIncomeFromColumn     tsv.ColumnName = "判定所得の下限[円]"
	NursingCarePremiumColumn        tsv.ColumnName = "保険料[円/年]"
)

type (
	NursingCareTaxedField string

	NursingCareHouseholdField string
)

const (
	NursingCareTaxed  NursingCareTaxedField = "課税"
	NursingCareExempt NursingCareTaxedField = "非課税"

	NursingCareHouseholdTaxedYes   NursingCareHouseholdField = "いる"
	NursingCareHouseholdTaxedNo    NursingCareHouseholdField = "いない"
	NursingCareHouseholdIrrelevant NursingCareHouseholdField = "問わない"
)

type NursingCarePremiumTable struct {
	byGroup map[NursingCareGroup]relation.Bands[money.Yen, NursingCareCharge]

	lastWritten date.Year
	written     bool

	growth CostGrowth
}

type NursingCareGroup int

const (
	NursingCareExemptHousehold NursingCareGroup = iota

	NursingCareExemptInTaxedHousehold

	NursingCareTaxedPerson
)

func NursingCareGroups() []NursingCareGroup {
	return []NursingCareGroup{
		NursingCareExemptHousehold, NursingCareExemptInTaxedHousehold, NursingCareTaxedPerson,
	}
}

func (g NursingCareGroup) String() string {
	switch g {
	case NursingCareExemptHousehold:
		return "本人非課税・世帯に課税者なし"
	case NursingCareExemptInTaxedHousehold:
		return "本人非課税・世帯に課税者あり"
	case NursingCareTaxedPerson:
		return "本人課税"
	}
	return fmt.Sprintf("NursingCareGroup(%d)", int(g))
}

type NursingCareStage int

func (s NursingCareStage) String() string {
	switch {
	case s == 0:
		return "段階なし"
	case s < 0:
		return fmt.Sprintf("段階でない値(%d)", int(s))
	}
	return fmt.Sprintf("第%d段階", int(s))
}

type NursingCareCharge struct {
	Stage   NursingCareStage
	Premium money.Yen
}

type NursingCarePremiumSubject struct {
	Taxed bool

	HouseholdTaxed bool

	TotalIncome money.Yen

	PensionReceipts money.Yen

	PensionIncome money.Yen
}

func (s NursingCarePremiumSubject) AssessedIncome() money.Yen {
	if s.Taxed {
		return s.TotalIncome
	}
	return s.PensionReceipts + max(s.TotalIncome-s.PensionIncome, 0)
}

func (s NursingCarePremiumSubject) Group() NursingCareGroup {
	switch {
	case s.Taxed:
		return NursingCareTaxedPerson
	case s.HouseholdTaxed:
		return NursingCareExemptInTaxedHousehold
	}
	return NursingCareExemptHousehold
}

func (t NursingCarePremiumTable) Charge(s NursingCarePremiumSubject, year date.Year) NursingCareCharge {
	if t.written && year < t.lastWritten {
		panic(fmt.Sprintf(
			"law.NursingCarePremiumTable.Charge: %s に %d 年の行が無い。この表は %d 年度から始まる期しか"+
				"持っておらず、それより前の期の段階は読み捨てられている",
			NursingCarePremiumTableName, year, t.lastWritten))
	}

	bands, ok := t.byGroup[s.Group()]
	if !ok {
		panic(fmt.Sprintf(
			"law.NursingCarePremiumTable.Charge: %v の段階が表に無い", s.Group()))
	}
	charge := bands.Lookup(s.AssessedIncome())
	charge.Premium = t.growth.CarePremium.GrowPremium(charge.Premium, t.LastWrittenYear, year)
	return charge
}

func (t NursingCarePremiumTable) LastWrittenYear() (date.Year, bool) { return t.lastWritten, t.written }

func (t NursingCarePremiumTable) WithGrowth(g CostGrowth) NursingCarePremiumTable {
	g.AssertStated()
	t.growth = g
	return t
}

func LoadNursingCarePremiumTable(fsys fs.FS, municipality Municipality) (NursingCarePremiumTable, error) {
	table, err := LoadRegionalTable(fsys, string(municipality), NursingCarePremiumTableName)
	if err != nil {
		return NursingCarePremiumTable{}, fmt.Errorf("law.LoadNursingCarePremiumTable: %w", err)
	}

	r, err := newReader(table, NursingCarePremiumTableName, NursingCareStageColumn,
		NursingCareTaxedColumn, NursingCareHouseholdTaxedColumn,
		NursingCareIncomeFromColumn, NursingCarePremiumColumn, LawStartYearColumn)
	if err != nil {
		return NursingCarePremiumTable{}, fmt.Errorf("law.LoadNursingCarePremiumTable: %w", err)
	}

	byGroup := make(map[NursingCareGroup][]relation.Band[money.Yen, NursingCareCharge], 3)
	var latest date.Year
	written := false
	for row := range r.Rows() {
		startInt, err := r.startYear(row)
		if err != nil {
			return NursingCarePremiumTable{}, fmt.Errorf("law.LoadNursingCarePremiumTable: %w", err)
		}
		start := date.Year(startInt)
		stage, err := nursingCareStageOf(r, row)
		if err != nil {
			return NursingCarePremiumTable{}, fmt.Errorf("law.LoadNursingCarePremiumTable: %w", err)
		}
		group, err := nursingCareGroupOf(
			NursingCareTaxedField(r.Field(row, NursingCareTaxedColumn)),
			NursingCareHouseholdField(r.Field(row, NursingCareHouseholdTaxedColumn)))
		if err != nil {
			return NursingCarePremiumTable{}, fmt.Errorf("law.LoadNursingCarePremiumTable: row %d: %w", row+1, err)
		}
		from, err := r.Yen(row, NursingCareIncomeFromColumn)
		if err != nil {
			return NursingCarePremiumTable{}, fmt.Errorf("law.LoadNursingCarePremiumTable: %w", err)
		}
		premium, err := r.Yen(row, NursingCarePremiumColumn)
		if err != nil {
			return NursingCarePremiumTable{}, fmt.Errorf("law.LoadNursingCarePremiumTable: %w", err)
		}

		if !written || start > latest {
			latest, written = start, true
			byGroup = make(map[NursingCareGroup][]relation.Band[money.Yen, NursingCareCharge], 3)
		}
		if start < latest {
			continue
		}
		byGroup[group] = append(byGroup[group], relation.Band[money.Yen, NursingCareCharge]{
			Lower: from, Value: NursingCareCharge{Stage: stage, Premium: premium}})
	}

	for _, want := range NursingCareGroups() {
		if len(byGroup[want]) == 0 {
			return NursingCarePremiumTable{}, fmt.Errorf(
				"law.LoadNursingCarePremiumTable: %s に %v の段階が無い。その人たちの保険料が引けない",
				municipality, want)
		}
	}

	banded := make(map[NursingCareGroup]relation.Bands[money.Yen, NursingCareCharge], len(byGroup))
	for group, rows := range byGroup {
		bands := relation.NewBands(rows)
		lowest, ok := bands.Min()
		if !ok || lowest != 0 {
			return NursingCarePremiumTable{}, fmt.Errorf(
				"law.LoadNursingCarePremiumTable: %s: %v の最も低い段階が %d 円から始まっており、それより低い所得を引けない",
				municipality, group, lowest)
		}
		banded[group] = bands
	}
	return NursingCarePremiumTable{byGroup: banded, lastWritten: latest, written: written}, nil
}

func nursingCareStageOf(r reader, row int) (NursingCareStage, error) {
	n, err := r.Count(row, NursingCareStageColumn)
	if err != nil {
		return 0, err
	}
	if n < 1 {
		return 0, r.Errorf(row, NursingCareStageColumn, "段階が %d である。1 以上でなければならない", n)
	}
	return NursingCareStage(n), nil
}

func nursingCareGroupOf(taxed NursingCareTaxedField, householdTaxed NursingCareHouseholdField) (NursingCareGroup, error) {
	switch taxed {
	case NursingCareTaxed:
		if householdTaxed != NursingCareHouseholdIrrelevant {
			return 0, fmt.Errorf(
				"%q の段階に 世帯の市民税課税者=%q と書いてあるが、その段階の要件に世帯は出てこないので %q でなければならない",
				taxed, householdTaxed, NursingCareHouseholdIrrelevant)
		}
		return NursingCareTaxedPerson, nil

	case NursingCareExempt:
		switch householdTaxed {
		case NursingCareHouseholdTaxedYes:
			return NursingCareExemptInTaxedHousehold, nil
		case NursingCareHouseholdTaxedNo:
			return NursingCareExemptHousehold, nil
		}
		return 0, fmt.Errorf("世帯の市民税課税者が %q である。%q か %q のはず",
			householdTaxed, NursingCareHouseholdTaxedYes, NursingCareHouseholdTaxedNo)
	}
	return 0, fmt.Errorf("本人の市民税が %q である。%q か %q のはず",
		taxed, NursingCareTaxed, NursingCareExempt)
}

func NursingCareSecondCategoryFrom(born date.Date) date.Date {
	return born.ReachesAge(NursingCareAgeMin)
}

func NursingCareFirstCategoryFrom(born date.Date) date.Date {
	return born.ReachesAge(NursingCareFirstCategoryAge)
}

func NursingCareSecondCategoryMonthsIn(year date.Year, born date.Date) date.Months {
	first := NursingCareFirstCategoryFrom(born)
	through := date.Date{Year: first.Year, Month: first.Month, Day: 1}.DayBefore()
	return date.MonthsOfYearIn(year, NursingCareSecondCategoryFrom(born), through)
}

func NursingCareFirstCategoryMonthsIn(year date.Year, born date.Date) date.Months {
	return date.MonthsOfYearFromIn(year, NursingCareFirstCategoryFrom(born))
}
