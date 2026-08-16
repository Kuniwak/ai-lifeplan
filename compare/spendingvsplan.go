package compare

import (
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	SpendingVsPlanYearColumn    tsv.ColumnName = "西暦"
	SpendingVsPlanProjectColumn tsv.ColumnName = "プロジェクト"
	SpendingVsPlanPlannedColumn tsv.ColumnName = "計画の総支出"
	SpendingVsPlanActualColumn  tsv.ColumnName = "支出−運用損益"
	SpendingVsPlanColumn        tsv.ColumnName = "計画との差"
)

var outgoingMetric = Metric{Table: "outturn", Column: SpendingVsPlanActualColumn}

func SpendingVsPlan(subjects []Subject) (*tsv.Table, error) {
	out := &tsv.Table{Header: []tsv.ColumnName{
		SpendingVsPlanYearColumn, SpendingVsPlanProjectColumn,
		SpendingVsPlanPlannedColumn, SpendingVsPlanActualColumn, SpendingVsPlanColumn,
	}}

	for _, subject := range subjects {
		if _, ok := subject.Tables[outgoingMetric.Table]; !ok {
			continue
		}
		actual, ok, err := subject.cells("compare.SpendingVsPlan", outgoingMetric.Table, outgoingMetric.Column)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		planned, ok, err := subject.cells("compare.SpendingVsPlan", spendingMetric.Table, spendingMetric.Column)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}

		for _, year := range slices.Sorted(maps.Keys(actual)) {
			plannedWritten, ok := planned[year]
			if !ok {
				continue
			}
			actualWritten := actual[year]

			plannedYen, err := strconv.ParseInt(plannedWritten, 10, 64)
			if err != nil {
				return nil, fmt.Errorf(
					"compare.SpendingVsPlan: %s: %d: %q is not an amount", subject.Name, year, plannedWritten)
			}
			actualYen, err := strconv.ParseInt(actualWritten, 10, 64)
			if err != nil {
				return nil, fmt.Errorf(
					"compare.SpendingVsPlan: %s: %d: %q is not an amount", subject.Name, year, actualWritten)
			}

			out.Rows = append(out.Rows, []string{
				strconv.Itoa(int(year)), subject.Name,
				strconv.FormatInt(plannedYen, 10), strconv.FormatInt(actualYen, 10),
				strconv.FormatInt(actualYen-plannedYen, 10),
			})
		}
	}
	return out, nil
}
