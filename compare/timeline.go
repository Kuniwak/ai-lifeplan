package compare

import (
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

type Metric struct {
	Table  plan.TableName
	Column tsv.ColumnName
}

func Headline() []Metric {
	return []Metric{
		{Table: "timeline", Column: "総収入"},
		{Table: "timeline", Column: "総支出"},
		{Table: "timeline", Column: "収支"},
		{Table: plan.AssetsTable, Column: plan.AssetsTotalColumn},
		{Table: "assets", Column: "手が届く額"},
		{Table: plan.AssetsTable, Column: plan.AssetsShortfallColumn},
	}
}

func Timeline(subjects []Subject) (*tsv.Table, error) {
	if len(subjects) == 0 {
		return &tsv.Table{Header: []tsv.ColumnName{plan.YearColumn}}, nil
	}

	headline := Headline()
	out := &tsv.Table{Header: []tsv.ColumnName{plan.YearColumn}}
	columns := make([]map[date.Year]string, 0, len(headline)*len(subjects))
	seen := make(map[date.Year]struct{})
	for _, metric := range headline {
		for _, subject := range subjects {
			values, err := subject.byYear(metric)
			if err != nil {
				return nil, err
			}
			out.Header = append(out.Header, tsv.ColumnName(
				fmt.Sprintf("%s:%s", metric.Column, subject.Name)))
			columns = append(columns, values)
			for year := range values {
				seen[year] = struct{}{}
			}
		}
	}

	for _, year := range slices.Sorted(maps.Keys(seen)) {
		row := []string{strconv.Itoa(int(year))}
		for _, values := range columns {
			row = append(row, values[year])
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

func (s Subject) cells(by string, name plan.TableName, column tsv.ColumnName) (map[date.Year]string, bool, error) {
	table, ok := s.Tables[name]
	if !ok {
		return nil, false, fmt.Errorf("%s: %s has no table %s", by, s.Name, name)
	}
	rows, err := rowsByYear(by, name, s, table)
	if err != nil {
		return nil, false, err
	}

	at, ok := table.ColumnIndex(column)
	if !ok {
		return nil, false, nil
	}
	out := make(map[date.Year]string, len(rows))
	for year, row := range rows {
		out[year] = row[at]
	}
	return out, true, nil
}

func (s Subject) byYear(metric Metric) (map[date.Year]string, error) {
	out, ok, err := s.cells("compare.Timeline", metric.Table, metric.Column)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf(
			"compare.Timeline: %s: %s has no %q column", s.Name, metric.Table, metric.Column)
	}
	return out, nil
}

func (s Subject) column(metric Metric) ([]string, error) {
	table, ok := s.Tables[metric.Table]
	if !ok {
		return nil, fmt.Errorf("compare: %s has no table %s", s.Name, metric.Table)
	}
	at, ok := table.ColumnIndex(metric.Column)
	if !ok {
		return nil, fmt.Errorf("compare: %s: %s has no %q column", s.Name, metric.Table, metric.Column)
	}

	values := make([]string, len(table.Rows))
	for i, row := range table.Rows {
		values[i] = row[at]
	}
	return values, nil
}

func (s Subject) years(name plan.TableName) ([]string, error) {
	return s.column(Metric{Table: name, Column: plan.YearColumn})
}
