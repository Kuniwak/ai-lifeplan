package breakeven

import (
	"fmt"
	"strings"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

type Painted struct {
	Rows []relation.Row[Setting]

	Now Setting
}

func (p Painted) Flattened() []relation.Row[Setting] {
	out := make([]relation.Row[Setting], 0, len(p.Rows))
	for _, row := range p.Rows {
		if !row.Value.Comparable(p.Now) || row.Value.Cmp(p.Now) != 0 {
			out = append(out, row)
		}
	}
	return out
}

func (p Painted) Span() (first, last date.Year, ok bool) {
	if len(p.Rows) == 0 {
		return 0, 0, false
	}
	first, last = p.Rows[0].Year, p.Rows[0].Year
	for _, row := range p.Rows {
		first, last = min(first, row.Year), max(last, row.Year)
	}
	return first, last, true
}

func (d Dial) PaintedOver(written *tsv.Table, now Setting) (Painted, error) {
	if d.Correction == Postpone {
		return Painted{Now: now}, nil
	}

	yearAt, settingAt, err := d.columnsOf(written)
	if err != nil {
		return Painted{}, err
	}
	needed, err := d.companionsOf(written)
	if err != nil {
		return Painted{}, err
	}

	painted := Painted{Now: now}
	for row, fields := range written.Rows {
		year, err := date.ParseYear(fields[yearAt])
		if err != nil {
			return Painted{}, fmt.Errorf("breakeven: %s: row %d: %w", d.Slot, row+1, err)
		}
		if year <= d.From {
			continue
		}

		lacks, err := needed.lacking(fields)
		if err != nil {
			return Painted{}, fmt.Errorf("breakeven: %s: row %d: %w", d.Slot, row+1, err)
		}
		if lacks != "" {
			continue
		}
		setting, err := d.Kind.Parse(fields[settingAt])
		if err != nil {
			return Painted{}, fmt.Errorf("breakeven: %s: row %d: %q: %w", d.Slot, row+1, d.Column, err)
		}
		painted.Rows = append(painted.Rows, relation.Row[Setting]{Year: year, Value: setting})
	}
	return painted, nil
}

func (s Swept) Warning() string {
	first, last, ok := s.Painted.Span()
	if !ok {
		return s.leftAloneWarning()
	}

	flattened := s.Painted.Flattened()
	if len(flattened) == 0 {
		return fmt.Sprintf(
			"%s: %d〜%d 年の %d 行も同じ設定になる（どれも %s のままなので、表に書かれた形は失われない）%s",
			s.Dial.Name, first, last, len(s.Painted.Rows), s.Painted.Now, s.leftAloneClause())
	}

	lost := flattened[len(flattened)-1]
	return fmt.Sprintf(
		"%s: **%d〜%d 年の %d 行をこの設定で塗り潰している。うち %d 行は %s ではなく、"+
			"%d 年の %s まで書かれている。掃引はその形を消す**%s",
		s.Dial.Name, first, last, len(s.Painted.Rows),
		len(flattened), s.Painted.Now, lost.Year, lost.Value, s.leftAloneClause())
}

func (s Swept) leftAloneWarning() string {
	if len(s.LeftAlone) == 0 {
		return ""
	}
	return s.Dial.Name + ":" + s.leftAloneClause()
}

func (s Swept) leftAloneClause() string {
	if len(s.LeftAlone) == 0 {
		return ""
	}

	years := make([]string, 0, len(s.LeftAlone))
	for _, year := range s.LeftAlone {
		years = append(years, fmt.Sprint(year))
	}
	return fmt.Sprintf(
		"。**%s 年は書き換えていない**——その年は %s が 0 以下で、%s だけを立てると"+
			"被用者保険から外れる",
		strings.Join(years, "・"), s.stoppedColumn(), s.Dial.Column)
}

func (s Swept) stoppedColumn() tsv.ColumnName {
	for _, pair := range input.PositiveTogether() {
		if pair.Slot == s.Dial.Slot && pair.Positive == s.Dial.Column {
			return pair.Needed
		}
	}
	return ""
}
