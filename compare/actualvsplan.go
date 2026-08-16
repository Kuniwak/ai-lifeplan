package compare

import (
	"fmt"
	"strconv"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	GapYearColumn    tsv.ColumnName = "西暦"
	GapProjectColumn tsv.ColumnName = "プロジェクト"
	GapBucketColumn  tsv.ColumnName = "区分"
	GapPlannedColumn tsv.ColumnName = "計画"
	GapActualColumn  tsv.ColumnName = "実績"
	GapColumn        tsv.ColumnName = "乖離"

	GapBeforeOriginColumn tsv.ColumnName = "起点以前"

	GapPartialColumn tsv.ColumnName = "一部未記録"
)

const beforeOriginYes = "はい"

var buckets = []struct {
	name   tsv.ColumnName
	holdOf func(actuals.Balance) money.Yen
}{
	{"貯蓄", func(b actuals.Balance) money.Yen { return b.Cash }},
	{"金融資産", func(b actuals.Balance) money.Yen { return b.Invested }},
	{"年金資産", func(b actuals.Balance) money.Yen { return b.Locked }},
	{"資産合計", actuals.Balance.Total},
}

func ActualVsPlan(subjects []Subject) (*tsv.Table, error) {
	out := &tsv.Table{Header: []tsv.ColumnName{
		GapYearColumn, GapProjectColumn, GapBucketColumn,
		GapPlannedColumn, GapActualColumn, GapColumn,
		GapBeforeOriginColumn, GapPartialColumn,
	}}

	held := RecordOf(subjects)
	for _, subject := range subjects {
		for _, year := range held.Years() {
			balance, ok := held.At(year)
			if !ok {
				continue
			}
			partial := ""
			if balance.Partial {
				partial = string(actuals.PartialYes)
			}
			beforeOrigin := ""
			if year <= subject.StartsAfter {
				beforeOrigin = beforeOriginYes
			}

			for _, bucket := range buckets {
				planned, ok, err := subject.plannedHold(year, bucket.name)
				if err != nil {
					return nil, err
				}
				if !ok {
					continue
				}
				actual := int64(bucket.holdOf(balance))
				out.Rows = append(out.Rows, []string{
					strconv.Itoa(int(year)), subject.Name, string(bucket.name),
					strconv.FormatInt(planned, 10),
					strconv.FormatInt(actual, 10),
					strconv.FormatInt(actual-planned, 10),
					beforeOrigin, partial,
				})
			}
		}
	}
	return out, nil
}

func (s Subject) plannedHold(year date.Year, bucket tsv.ColumnName) (int64, bool, error) {
	cells, ok, err := s.cells("compare.ActualVsPlan", plan.AssetsTable, bucket)
	if err != nil || !ok {
		return 0, false, err
	}
	written, ok := cells[year]
	if !ok {
		return 0, false, nil
	}

	held, err := strconv.ParseInt(written, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf(
			"compare.ActualVsPlan: %s: %s: %q is not an amount", s.Name, bucket, written)
	}
	return held, true, nil
}
