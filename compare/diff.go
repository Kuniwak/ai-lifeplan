package compare

import (
	"fmt"
	"maps"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/sets"
	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	TableColumn    tsv.ColumnName = "表"
	ColumnColumn   tsv.ColumnName = "列"
	ProjectColumn  tsv.ColumnName = "プロジェクト"
	BaseColumn     tsv.ColumnName = "基準"
	ValueColumn    tsv.ColumnName = "値"
	DeltaColumn    tsv.ColumnName = "差"
	PresenceColumn tsv.ColumnName = "年の有無"
)

type Presence string

const (
	InBoth      Presence = "両方"
	InBaseOnly  Presence = "基準のみ"
	InOtherOnly Presence = "プロジェクトのみ"
)

func Diff(subjects []Subject) (*tsv.Table, error) {
	out := &tsv.Table{Header: []tsv.ColumnName{
		TableColumn, plan.YearColumn, ColumnColumn, ProjectColumn,
		BaseColumn, ValueColumn, DeltaColumn, PresenceColumn,
	}}
	if len(subjects) < 2 {
		return out, nil
	}
	base := subjects[0]

	for _, name := range slices.Sorted(maps.Keys(base.Tables)) {
		baseTable := base.Tables[name]
		yearAt, ok := baseTable.ColumnIndex(plan.YearColumn)
		if !ok {
			return nil, fmt.Errorf("compare.Diff: %s: %s has no %q column", base.Name, name, plan.YearColumn)
		}
		baseRows, err := rowsByYear("compare.Diff", name, base, baseTable)
		if err != nil {
			return nil, err
		}

		others := make([]map[date.Year][]string, 0, len(subjects)-1)
		years := slices.Collect(maps.Keys(baseRows))
		for _, subject := range subjects[1:] {
			other, ok := subject.Tables[name]
			if !ok {
				return nil, fmt.Errorf(
					"compare.Diff: %s has the table %s and %s has not, so there is nothing to compare it against",
					base.Name, name, subject.Name)
			}
			if err := assertSameColumns(name, base, subject, baseTable, other); err != nil {
				return nil, err
			}
			otherRows, err := rowsByYear("compare.Diff", name, subject, other)
			if err != nil {
				return nil, err
			}
			for year := range otherRows {
				if _, both := baseRows[year]; !both {
					years = append(years, year)
				}
			}
			others = append(others, otherRows)
		}

		slices.Sort(years)
		years = slices.Compact(years)

		for _, year := range years {
			was, inBase := baseRows[year]

			for i, otherRows := range others {
				is, inOther := otherRows[year]
				if inBase == inOther {
					continue
				}
				only, held := InBaseOnly, was
				if inOther {
					only, held = InOtherOnly, is
				}
				out.Rows = append(out.Rows, []string{
					string(name), held[yearAt], "", subjects[i+1].Name, "", "", "", string(only),
				})
			}

			if !inBase {
				continue
			}
			for at, column := range baseTable.Header {
				if at == yearAt {
					continue
				}
				for i, otherRows := range others {
					is, inOther := otherRows[year]
					if !inOther || was[at] == is[at] {
						continue
					}
					out.Rows = append(out.Rows, []string{
						string(name), was[yearAt], string(column), subjects[i+1].Name,
						was[at], is[at], delta(was[at], is[at]), string(InBoth),
					})
				}
			}
		}
	}
	return out, nil
}

func rowsByYear(by string, name plan.TableName, subject Subject, table *tsv.Table) (map[date.Year][]string, error) {
	byYear, err := table.RowsByYear(string(name), plan.YearColumn)
	if err != nil {
		return nil, fmt.Errorf("%s: %s: %w", by, subject.Name, err)
	}
	return byYear, nil
}

func delta(was, is string) string {
	if from, err := strconv.ParseInt(was, 10, 64); err == nil {
		if to, err := strconv.ParseInt(is, 10, 64); err == nil {
			return strconv.FormatInt(to-from, 10)
		}
	}

	from, ok := decimal(was)
	if !ok {
		return ""
	}
	to, ok := decimal(is)
	if !ok {
		return ""
	}

	return new(big.Rat).Sub(to, from).FloatString(max(places(was), places(is)))
}

func decimal(s string) (*big.Rat, bool) {
	if s == "" {
		return nil, false
	}
	body := strings.TrimPrefix(s, "-")
	if body == "" {
		return nil, false
	}
	for _, r := range body {
		if r != '.' && (r < '0' || r > '9') {
			return nil, false
		}
	}
	if strings.Count(body, ".") > 1 {
		return nil, false
	}
	return new(big.Rat).SetString(s)
}

func places(s string) int {
	at := strings.IndexByte(s, '.')
	if at < 0 {
		return 0
	}
	return len(s) - at - 1
}

func assertSameColumns(name plan.TableName, base, other Subject, was, is *tsv.Table) error {
	if !slices.Equal(was.Header, is.Header) {
		mine := make([]string, len(was.Header))
		for i, column := range was.Header {
			mine[i] = string(column)
		}
		theirs := make([]string, len(is.Header))
		for i, column := range is.Header {
			theirs[i] = string(column)
		}
		slices.Sort(mine)
		slices.Sort(theirs)

		return fmt.Errorf(
			"compare.Diff: %s: the columns differ, %s has %v and %s has %v",
			name,
			base.Name, sets.Difference(mine, theirs),
			other.Name, sets.Difference(theirs, mine))
	}
	return nil
}
