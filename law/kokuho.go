package law

import (
	"fmt"
	"io/fs"
	"slices"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	KokuhoPartColumn      tsv.ColumnName = "区分"
	KokuhoIncomeRate      tsv.ColumnName = "所得割率"
	KokuhoPerPersonColumn tsv.ColumnName = "均等割[円/人]"
	KokuhoPerHouseColumn  tsv.ColumnName = "平等割[円/世帯]"
	KokuhoCapColumn       tsv.ColumnName = "課税限度額[円]"
)

type KokuhoPartName string

const KokuhoPremiumUnit money.Yen = 100

const (
	KokuhoMedical        KokuhoPartName = "医療分"
	KokuhoElderlySupport KokuhoPartName = "後期支援分"
	KokuhoNursingCare    KokuhoPartName = "介護分"
	KokuhoChildSupport   KokuhoPartName = "子ども子育て支援分"
)

func KokuhoPartNames() []KokuhoPartName {
	return []KokuhoPartName{
		KokuhoMedical, KokuhoElderlySupport, KokuhoNursingCare, KokuhoChildSupport,
	}
}

const KokuhoTableName = "kokuho"

type KokuhoPart struct {
	Rate      money.Rate
	PerPerson money.Yen
	PerHouse  money.Yen
	Cap       money.Yen
}

type KokuhoTable struct {
	partedTable[KokuhoPartName, KokuhoPart]

	growth CostGrowth
}

func (t KokuhoTable) WithGrowth(g CostGrowth) KokuhoTable {
	g.AssertStated()
	t.growth = g
	return t
}

func LoadKokuhoTable(fsys fs.FS, municipality Municipality) (KokuhoTable, error) {
	table, err := LoadRegionalTable(fsys, string(municipality), KokuhoTableName)
	if err != nil {
		return KokuhoTable{}, fmt.Errorf("law.LoadKokuhoTable: %w", err)
	}
	return ParseKokuhoTable(table)
}

func ParseKokuhoTable(table *tsv.Table) (KokuhoTable, error) {
	r, err := newReader(table, KokuhoTableName, KokuhoPartColumn, KokuhoIncomeRate,
		KokuhoPerPersonColumn, KokuhoPerHouseColumn, KokuhoCapColumn,
		LawStartYearColumn, LawEndYearColumn)
	if err != nil {
		return KokuhoTable{}, fmt.Errorf("law.ParseKokuhoTable: %w", err)
	}

	byPart := make(map[KokuhoPartName][]relation.Period[date.Year, KokuhoPart], 4)
	for row := range r.Rows() {
		from, err := r.startBound(row)
		if err != nil {
			return KokuhoTable{}, fmt.Errorf("law.ParseKokuhoTable: %w", err)
		}
		to, err := r.endBound(row)
		if err != nil {
			return KokuhoTable{}, fmt.Errorf("law.ParseKokuhoTable: %w", err)
		}
		rate, err := r.Percent(row, KokuhoIncomeRate)
		if err != nil {
			return KokuhoTable{}, fmt.Errorf("law.ParseKokuhoTable: %w", err)
		}
		perPerson, err := r.Yen(row, KokuhoPerPersonColumn)
		if err != nil {
			return KokuhoTable{}, fmt.Errorf("law.ParseKokuhoTable: %w", err)
		}
		perHouse, err := r.Yen(row, KokuhoPerHouseColumn)
		if err != nil {
			return KokuhoTable{}, fmt.Errorf("law.ParseKokuhoTable: %w", err)
		}
		cap, err := r.Yen(row, KokuhoCapColumn)
		if err != nil {
			return KokuhoTable{}, fmt.Errorf("law.ParseKokuhoTable: %w", err)
		}

		part := KokuhoPartName(r.Field(row, KokuhoPartColumn))
		if !slices.Contains(KokuhoPartNames(), part) {
			return KokuhoTable{}, r.Errorf(row, KokuhoPartColumn,
				"%q は国民健康保険税の区分ではない。%v のいずれかを書くこと（law.KokuhoPartNames）",
				part, KokuhoPartNames())
		}
		byPart[part] = append(byPart[part], relation.NewPeriod(from, to,
			KokuhoPart{Rate: rate, PerPerson: perPerson, PerHouse: perHouse, Cap: cap}))
	}

	parted, err := newPartedTable("law.ParseKokuhoTable", byPart, KokuhoPartNames())
	if err != nil {
		return KokuhoTable{}, err
	}
	return KokuhoTable{partedTable: parted}, nil
}

type KokuhoMember struct {
	Base money.Yen

	Months, NursingCareMonths date.Months
}

type KokuhoHousehold struct {
	Members []KokuhoMember
}

func (h KokuhoHousehold) baseOf(monthsOf func(KokuhoMember) date.Months) money.Yen {
	var base money.Yen
	for _, member := range h.Members {
		if monthsOf(member).Count() > 0 {
			base += max(member.Base, 0)
		}
	}
	return base
}

func (h KokuhoHousehold) countOf(monthsOf func(KokuhoMember) date.Months) int {
	count := 0
	for _, member := range h.Members {
		if monthsOf(member).Count() > 0 {
			count++
		}
	}
	return count
}

const KokuhoBasicDeduction = KoukiBasicDeduction

func (t KokuhoTable) Premium(household KokuhoHousehold, year date.Year) (money.Yen, error) {
	if len(t.parts) == 0 {
		return 0, fmt.Errorf("law.KokuhoTable.Premium: the table carries no parts")
	}

	var total money.Yen
	for _, name := range t.Parts() {
		monthsOf := kokuhoMonthsOf(name)

		var baseMonths money.Yen
		var memberMonths int
		householdMonths := date.NoMonths
		for _, member := range household.Members {
			months := monthsOf(member).Count()
			if months <= 0 {
				continue
			}
			baseMonths += max(member.Base, 0) * money.Yen(months)
			memberMonths += months
			householdMonths = householdMonths.Union(monthsOf(member))
		}
		if memberMonths <= 0 {
			continue
		}

		part, ok := t.At(name, year)
		if !ok {
			continue
		}

		amount := (baseMonths/date.MonthsAYear).Mul(part.Rate, money.Truncate) +
			part.PerPerson*money.Yen(memberMonths)/date.MonthsAYear +
			part.PerHouse.ForMonths(householdMonths.Count())

		yearly := household.baseOf(monthsOf).Mul(part.Rate, money.Truncate) +
			part.PerPerson*money.Yen(household.countOf(monthsOf)) +
			part.PerHouse
		capped := amount
		if yearly > part.Cap {
			capped = part.Cap * amount / yearly
		}

		banded := t.yearsOf(name)
		total += t.growth.of(name).GrowPremium(capped, banded.LastWrittenYear, year).Truncate(KokuhoPremiumUnit)
	}

	return total, nil
}

func kokuhoMonthsOf(name KokuhoPartName) func(KokuhoMember) date.Months {
	switch name {
	case KokuhoNursingCare:
		return func(m KokuhoMember) date.Months { return m.NursingCareMonths }
	case KokuhoMedical, KokuhoElderlySupport, KokuhoChildSupport:
		return func(m KokuhoMember) date.Months { return m.Months }
	default:
		panic(fmt.Sprintf("law.kokuhoMonthsOf: %q は国民健康保険税の区分ではない。%v のいずれかであること", name, KokuhoPartNames()))
	}
}

func (g CostGrowth) of(part KokuhoPartName) CostGrowthCurve {
	switch part {
	case KokuhoNursingCare:
		return g.Care
	case KokuhoMedical, KokuhoElderlySupport, KokuhoChildSupport:
		return g.Medical
	default:
		panic(fmt.Sprintf("law.CostGrowth.of: %q は国民健康保険税の区分ではない。%v のいずれかであること", part, KokuhoPartNames()))
	}
}
