package compare

import (
	"fmt"
	"strconv"

	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	SummaryProjectColumn   tsv.ColumnName = "プロジェクト"
	SummaryStartsColumn    tsv.ColumnName = "起点"
	SummaryShortFromColumn tsv.ColumnName = "資産が尽きる年"
	SummaryShortfallColumn tsv.ColumnName = "不足の累計"
	SummaryReceiptsColumn  tsv.ColumnName = "生涯総収入"
	SummarySpendingColumn  tsv.ColumnName = "生涯総支出"
	SummaryTaxColumn       tsv.ColumnName = "生涯総税額"
	SummaryLastYearColumn  tsv.ColumnName = "最終年"
	SummaryFinalColumn     tsv.ColumnName = "最終年の資産合計"
)

var (
	finalMetric    = Metric{Table: plan.AssetsTable, Column: plan.AssetsTotalColumn}
	receiptsMetric = Metric{Table: "timeline", Column: "総収入"}
	spendingMetric = Metric{Table: "timeline", Column: "総支出"}
	taxMetric      = Metric{Table: "tax", Column: "合計"}
)

func Summary(subjects []Subject) (*tsv.Table, error) {
	out := &tsv.Table{Header: []tsv.ColumnName{
		SummaryProjectColumn, SummaryStartsColumn,
		SummaryShortFromColumn, SummaryShortfallColumn,
		SummaryReceiptsColumn, SummarySpendingColumn, SummaryTaxColumn,
		SummaryLastYearColumn, SummaryFinalColumn,
	}}

	for _, subject := range subjects {
		came, err := subject.outcome()
		if err != nil {
			return nil, err
		}
		receipts, err := subject.total(receiptsMetric)
		if err != nil {
			return nil, err
		}
		spending, err := subject.total(spendingMetric)
		if err != nil {
			return nil, err
		}
		tax, err := subject.total(taxMetric)
		if err != nil {
			return nil, err
		}
		out.Rows = append(out.Rows, []string{
			subject.Name,
			strconv.Itoa(int(subject.StartsAfter)),
			came.ShortFromField(),
			strconv.FormatInt(int64(came.Shortfall), 10),
			strconv.FormatInt(receipts, 10),
			strconv.FormatInt(spending, 10),
			strconv.FormatInt(tax, 10),
			strconv.Itoa(int(came.LastYear)),
			strconv.FormatInt(int64(came.Final), 10),
		})
	}
	return out, nil
}

func (s Subject) outcome() (plan.Outcome, error) {
	table, ok := s.Tables[plan.AssetsTable]
	if !ok {
		return plan.Outcome{}, fmt.Errorf(
			"compare: %s has no table %s", s.Name, plan.AssetsTable)
	}
	came, err := plan.OutcomeOf(table)
	if err != nil {
		return plan.Outcome{}, fmt.Errorf("compare: %s: %w", s.Name, err)
	}
	return came, nil
}

func (s Subject) total(metric Metric) (int64, error) {
	values, err := s.column(metric)
	if err != nil {
		return 0, err
	}

	var sum int64
	for _, value := range values {
		amount, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, fmt.Errorf(
				"compare.Summary: %s: %s: %q is not an amount", s.Name, metric.Column, value)
		}
		sum += amount
	}
	return sum, nil
}
