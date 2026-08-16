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

const KoukiBasicDeduction money.Yen = 430_000

const KoukiPremiumUnit money.Yen = 10

type KoukiPartName string

const (
	KoukiMedical      KoukiPartName = "医療分"
	KoukiChildSupport KoukiPartName = "子ども分"
)

func koukiGrowthOf(g CostGrowth, part KoukiPartName) CostGrowthCurve {
	switch part {
	case KoukiMedical, KoukiChildSupport:
		return g.Medical
	default:
		panic(fmt.Sprintf("law.koukiGrowthOf: %q は後期高齢者医療の保険料の区分ではない。%v のいずれかであること", part, KoukiPartNames()))
	}
}

func KoukiPartNames() []KoukiPartName {
	return []KoukiPartName{KoukiMedical, KoukiChildSupport}
}

type KoukiPart struct {
	PerCapita money.Yen

	IncomeRate money.Rate

	Cap money.Yen
}

const (
	KoukiPartColumn tsv.ColumnName = "区分"

	KoukiPerCapitaColumn  tsv.ColumnName = "均等割額[円]"
	KoukiIncomeRateColumn tsv.ColumnName = "所得割率"

	KoukiCapColumn tsv.ColumnName = "賦課限度額[円]"
)

type KoukiRatesTable struct {
	partedTable[KoukiPartName, KoukiPart]

	growth CostGrowth
}

func (t KoukiRatesTable) WithGrowth(g CostGrowth) KoukiRatesTable {
	g.AssertStated()
	t.growth = g
	return t
}

func ParseKoukiRatesTable(table *tsv.Table) (KoukiRatesTable, error) {
	r, err := newReader(table, KoukiRatesTableName, KoukiPartColumn,
		KoukiPerCapitaColumn, KoukiIncomeRateColumn, KoukiCapColumn,
		LawStartYearColumn, LawEndYearColumn)
	if err != nil {
		return KoukiRatesTable{}, fmt.Errorf("law.ParseKoukiRatesTable: %w", err)
	}

	byPart := make(map[KoukiPartName][]relation.Period[date.Year, KoukiPart], len(KoukiPartNames()))
	for row := range r.Rows() {
		from, err := r.startBound(row)
		if err != nil {
			return KoukiRatesTable{}, fmt.Errorf("law.ParseKoukiRatesTable: %w", err)
		}
		to, err := r.endBound(row)
		if err != nil {
			return KoukiRatesTable{}, fmt.Errorf("law.ParseKoukiRatesTable: %w", err)
		}
		perCapita, err := r.Yen(row, KoukiPerCapitaColumn)
		if err != nil {
			return KoukiRatesTable{}, fmt.Errorf("law.ParseKoukiRatesTable: %w", err)
		}
		rate, err := r.Percent(row, KoukiIncomeRateColumn)
		if err != nil {
			return KoukiRatesTable{}, fmt.Errorf("law.ParseKoukiRatesTable: %w", err)
		}
		capped, err := r.Yen(row, KoukiCapColumn)
		if err != nil {
			return KoukiRatesTable{}, fmt.Errorf("law.ParseKoukiRatesTable: %w", err)
		}

		part := KoukiPartName(r.Field(row, KoukiPartColumn))
		if !slices.Contains(KoukiPartNames(), part) {
			return KoukiRatesTable{}, r.Errorf(row, KoukiPartColumn,
				"%q は後期高齢者医療の保険料の区分ではない。%v のいずれかを書くこと（law.KoukiPartNames）",
				part, KoukiPartNames())
		}
		byPart[part] = append(byPart[part], relation.NewPeriod(from, to,
			KoukiPart{PerCapita: perCapita, IncomeRate: rate, Cap: capped}))
	}

	parted, err := newPartedTable("law.ParseKoukiRatesTable", byPart, KoukiPartNames())
	if err != nil {
		return KoukiRatesTable{}, err
	}
	return KoukiRatesTable{partedTable: parted}, nil
}

const KoukiRatesTableName = "kouki-rates"

func LoadKoukiRatesTable(fsys fs.FS, prefecture Prefecture) (KoukiRatesTable, error) {
	table, err := LoadRegionalTable(fsys, string(prefecture), KoukiRatesTableName)
	if err != nil {
		return KoukiRatesTable{}, err
	}
	return ParseKoukiRatesTable(table)
}

func (t KoukiRatesTable) PartAt(name KoukiPartName, year date.Year) (KoukiPart, bool) {
	return t.At(name, year)
}

func (t KoukiRatesTable) Premium(totalIncome money.Yen, year date.Year) money.Yen {
	if first, ok := t.FirstWrittenYear(); ok && year < first {
		panic(fmt.Sprintf(
			"law.KoukiRatesTable.Premium: %s に %d 年の行が無い。この表は %d 年からしか書かれておらず、"+
				"それより前は誰も値を書いていない。その表の最初の行の適用開始年を確かめること",
			KoukiRatesTableName, year, first))
	}

	var total money.Yen
	for _, name := range t.Parts() {
		total += t.PremiumOf(name, totalIncome, year)
	}
	return total
}

func (t KoukiRatesTable) FirstWrittenYear() (date.Year, bool) {
	var earliest date.Year
	found := false
	for _, name := range t.Parts() {
		first, ok := t.parts[name].FirstWrittenYear()
		if !ok {
			continue
		}
		if !found || first < earliest {
			earliest, found = first, true
		}
	}
	return earliest, found
}

func (t KoukiRatesTable) PremiumOf(name KoukiPartName, totalIncome money.Yen, year date.Year) money.Yen {
	part, ok := t.At(name, year)
	if !ok {
		return 0
	}

	base := max(totalIncome-KoukiBasicDeduction, 0)
	premium := min(part.PerCapita+base.Mul(part.IncomeRate, money.Truncate), part.Cap)

	banded := t.yearsOf(name)
	return koukiGrowthOf(t.growth, name).GrowPremium(premium, banded.LastWrittenYear, year).Truncate(KoukiPremiumUnit)
}
