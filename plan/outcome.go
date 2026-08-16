package plan

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

const AssetsTable TableName = "assets"

const (
	AssetsTotalColumn     tsv.ColumnName = "資産合計"
	AssetsShortfallColumn tsv.ColumnName = "不足"
)

type Outcome struct {
	ShortFrom date.Year

	Shortfall money.Yen

	LastYear date.Year
	Final    money.Yen
}

func (p *Plan) LastHeld() (table.AssetsRow, bool) {
	rows := p.Assets.Rows()
	if len(rows) == 0 {
		return table.AssetsRow{}, false
	}
	return rows[len(rows)-1].Value, true
}

func (o Outcome) Fails() bool { return o.Shortfall > 0 }

func (o Outcome) ShortFromField() string {
	if o.ShortFrom == 0 {
		return ""
	}
	return strconv.Itoa(int(o.ShortFrom))
}

func (p *Plan) Outcome() (Outcome, error) {
	rows := p.Assets.Rows()
	held := make([]yearHeld, len(rows))
	for i, row := range rows {
		held[i] = yearHeld{
			year:      row.Year,
			total:     row.Value.Total,
			shortfall: row.Value.Shortfall,
		}
	}
	return outcomeOf("plan.Plan.Outcome", held)
}

func OutcomeOf(assets *tsv.Table) (Outcome, error) {
	const by = "plan.OutcomeOf"

	read, err := tsv.NewReader(assets, tsv.Slot(AssetsTable),
		YearColumn, AssetsTotalColumn, AssetsShortfallColumn)
	if err != nil {
		return Outcome{}, err
	}

	held := make([]yearHeld, read.Rows())
	for i := range held {
		year, err := date.ParseYear(read.Field(i, YearColumn))
		if err != nil {
			return Outcome{}, read.Errorf(i, YearColumn, "%s", err)
		}
		total, err := money.ParseYen(read.Field(i, AssetsTotalColumn))
		if err != nil {
			return Outcome{}, read.Errorf(i, AssetsTotalColumn, "%s", err)
		}
		shortfall, err := money.ParseYen(read.Field(i, AssetsShortfallColumn))
		if err != nil {
			return Outcome{}, read.Errorf(i, AssetsShortfallColumn, "%s", err)
		}
		held[i] = yearHeld{year: year, total: total, shortfall: shortfall}
	}
	return outcomeOf(by, held)
}

type yearHeld struct {
	year      date.Year
	total     money.Yen
	shortfall money.Yen
}

func outcomeOf(by string, held []yearHeld) (Outcome, error) {
	if len(held) == 0 {
		return Outcome{}, fmt.Errorf(
			"%s: %s has no rows, so there is no year the plan ends in", by, AssetsTable)
	}
	slices.SortFunc(held, func(a, b yearHeld) int { return int(a.year - b.year) })

	var out Outcome
	for _, year := range held {
		if year.shortfall < 0 {
			return Outcome{}, fmt.Errorf(
				"%s: %s: %d の %q が %d で、負である。負の不足は「余った」ではなく、意味が決まっていない",
				by, AssetsTable, year.year, AssetsShortfallColumn, year.shortfall)
		}
		if year.shortfall > 0 && out.ShortFrom == 0 {
			out.ShortFrom = year.year
		}
		out.Shortfall += year.shortfall
	}

	last := held[len(held)-1]
	out.LastYear, out.Final = last.year, last.total
	return out, nil
}
