package law

import (
	"fmt"
	"math"
	"strconv"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const PensionRevaluationRateTableName = "national/pension-revaluation-rate"

const (
	RevaluationTableYearColumn tsv.ColumnName = "表の年度"

	RevaluationRateColumn tsv.ColumnName = "再評価率"

	RevaluationBirthColumn tsv.ColumnName = "生年度の区分"
)

var EarningsRelatedMultiplierAfterTotalRemuneration = money.NewRate(5_481, 1_000_000)

const TotalRemunerationFrom date.Year = 2003

type PensionRevaluationRates struct {
	table map[date.Year]map[date.Year]money.Rate
}

func ParsePensionRevaluationRates(t *tsv.Table) (PensionRevaluationRates, error) {
	var empty PensionRevaluationRates

	r, err := tsv.NewReader(t, tsv.Slot(PensionRevaluationRateTableName),
		RevaluationTableYearColumn, LawStartYearColumn, LawEndYearColumn,
		RevaluationRateColumn, RevaluationBirthColumn, LawSourceColumn)
	if err != nil {
		return empty, err
	}

	table := make(map[date.Year]map[date.Year]money.Rate, 2)
	for row := range r.Rows() {
		published, err := r.Year(row, RevaluationTableYearColumn)
		if err != nil {
			return empty, err
		}
		fiscal, err := r.Year(row, LawStartYearColumn)
		if err != nil {
			return empty, err
		}
		written := r.Field(row, RevaluationRateColumn)
		thousandths, err := strconv.ParseFloat(written, 64)
		if err != nil {
			return empty, r.Errorf(row, RevaluationRateColumn, "%q が小数でない", written)
		}
		rate := money.NewRate(int64(math.Round(thousandths*1_000_000)), 1_000_000)
		if table[published] == nil {
			table[published] = make(map[date.Year]money.Rate, 16)
		}
		if _, twice := table[published][fiscal]; twice {
			return empty, r.Errorf(row, LawStartYearColumn,
				"%d 年度の再評価率が %d 年度の表に二度書かれている", fiscal, published)
		}
		table[published][fiscal] = rate
	}
	return PensionRevaluationRates{table: table}, nil
}

type Remuneration struct {
	Year   date.Year
	Month  int
	Amount money.Yen
}

func (r Remuneration) FiscalYear() date.Year {
	if r.Month >= 4 {
		return r.Year
	}
	return r.Year - 1
}

func (t PensionRevaluationRates) EarningsRelatedPension(published date.Year, months []Remuneration) (money.Yen, error) {
	rates, ok := t.table[published]
	if !ok {
		return 0, fmt.Errorf(
			"law.EarningsRelatedPension: %d 年度の再評価率の表が無い。表は年度ごとに作り直される（%s）",
			published, PensionRevaluationRateTableName)
	}

	var revalued money.Yen
	for _, m := range months {
		fiscal := m.FiscalYear()
		if fiscal < TotalRemunerationFrom {
			return 0, fmt.Errorf(
				"law.EarningsRelatedPension: %d年%d月は総報酬制（平成15年度）より前である。"+
					"その期間は平均標準報酬**月額**に 7.125/1000 を掛ける別の式で、賞与は入らない。"+
					"この表はその式を持っていない", m.Year, m.Month)
		}
		rate, ok := rates[fiscal]
		if !ok {
			return 0, fmt.Errorf(
				"law.EarningsRelatedPension: %d 年度の再評価率が %d 年度の表に無い（%d年%d月の分）。"+
					"表に行を足すこと（%s）",
				fiscal, published, m.Year, m.Month, PensionRevaluationRateTableName)
		}
		revalued += m.Amount.Mul(rate, money.HalfUp)
	}
	return revalued.Mul(EarningsRelatedMultiplierAfterTotalRemuneration, money.HalfUp), nil
}

var PensionDeferralRatePerMonth = money.NewRate(7, 1_000)

var PensionEarlyRatePerMonth = money.NewRate(4, 1_000)

const PensionEarlyMaxMonths = 60

const PensionDeferralMaxMonths = 120

func PensionStartAdjustment(months int) (money.Rate, error) {
	switch {
	case months < -PensionEarlyMaxMonths:
		return money.Rate{}, fmt.Errorf(
			"law.PensionStartAdjustment: %d か月は繰上げの上限の %d か月（60 歳）を超えている",
			months, PensionEarlyMaxMonths)
	case months > PensionDeferralMaxMonths:
		return money.Rate{}, fmt.Errorf(
			"law.PensionStartAdjustment: %d か月は繰下げの上限の %d か月（75 歳）を超えている",
			months, PensionDeferralMaxMonths)
	case months < 0:
		return money.NewRate(1_000-int64(-months)*4, 1_000), nil
	}
	return money.NewRate(1_000+int64(months)*7, 1_000), nil
}
