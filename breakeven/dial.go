package breakeven

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/stepfn"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

type Dial struct {
	Name string

	Slot   tsv.Slot
	Column tsv.ColumnName

	Kind Kind

	YearColumn tsv.ColumnName

	TableKind input.Kind

	From date.Year

	Correction CorrectionKind
}

type CorrectionKind int

const (
	Overwrite CorrectionKind = iota

	Postpone
)

func DialOf(slot tsv.Slot, column tsv.ColumnName, from date.Year) (Dial, error) {
	for _, shape := range input.Shapes() {
		if shape.Slot != slot {
			continue
		}

		kind, err := KindOf(slot, column)
		if err != nil {
			return Dial{}, err
		}
		switch shape.Kind {
		case input.Step:
		case input.Events:
			return Dial{}, fmt.Errorf(
				"breakeven.DialOf: %s は起きたことの一覧なので、年から回すという言い方ができない。"+
					"同じ年が二度書かれることがあり、書かれていない年は「前の行が続く年」ではなく"+
					"「何も起きなかった年」である", slot)
		default:
			return Dial{}, fmt.Errorf(
				"breakeven.DialOf: %s は年で読む表ではないので、年から回すという言い方ができない", slot)
		}

		return Dial{
			Name:       string(column),
			Slot:       slot,
			Column:     column,
			Kind:       kind,
			YearColumn: shape.YearColumn,
			TableKind:  shape.Kind,
			From:       from,
		}, nil
	}
	return Dial{}, fmt.Errorf("breakeven.DialOf: %q という slot は無い", slot)
}

var postponableSlots = []tsv.Slot{
	input.IncomeHusbandSlot,
	input.IncomeWifeSlot,
}

func PostponeDialOf(slot tsv.Slot, from date.Year) (Dial, error) {
	if !slices.Contains(postponableSlots, slot) {
		return Dial{}, fmt.Errorf(
			"breakeven.PostponeDialOf: %s は延長で補正できる対象ではない。"+
				"就労延長として意味があるのは %v である", slot, postponableSlots)
	}

	for _, shape := range input.Shapes() {
		if shape.Slot != slot {
			continue
		}
		if shape.Kind != input.Step {
			return Dial{}, fmt.Errorf(
				"breakeven.PostponeDialOf: %s は区分定数の表ではないので、最後の行を延ばすという言い方ができない", slot)
		}
		return Dial{
			Name:       fmt.Sprintf("%s:延長", slot),
			Slot:       slot,
			Kind:       Years{},
			YearColumn: shape.YearColumn,
			TableKind:  shape.Kind,
			From:       from,
			Correction: Postpone,
		}, nil
	}
	return Dial{}, fmt.Errorf("breakeven.PostponeDialOf: %q という slot は無い", slot)
}

func (d Dial) LastRowYear(written *tsv.Table) (date.Year, error) {
	yearAt, ok := written.ColumnIndex(d.YearColumn)
	if !ok {
		return 0, fmt.Errorf("breakeven: %s has no %q column", d.Slot, d.YearColumn)
	}
	if len(written.Rows) == 0 {
		return 0, fmt.Errorf("breakeven: %s には行が無い", d.Slot)
	}
	last := len(written.Rows) - 1
	year, err := date.ParseYear(written.Rows[last][yearAt])
	if err != nil {
		return 0, fmt.Errorf("breakeven: %s: row %d: %w", d.Slot, last+1, err)
	}
	return year, nil
}

func (d Dial) turnPostpone(written *tsv.Table, setting Setting) (*tsv.Table, error) {
	lastYear, err := d.LastRowYear(written)
	if err != nil {
		return nil, err
	}
	if lastYear < d.From {
		return nil, fmt.Errorf(
			"breakeven: %s の最後の行は %d 年で、%d 年より前にある。まだ来ていない行しか延ばせない",
			d.Slot, lastYear, d.From)
	}

	years, err := yearsOf(setting)
	if err != nil {
		return nil, err
	}

	yearAt, _ := written.ColumnIndex(d.YearColumn)
	last := len(written.Rows) - 1
	rows := make([][]string, len(written.Rows))
	for i, fields := range written.Rows {
		rows[i] = slices.Clone(fields)
	}
	rows[last][yearAt] = fmt.Sprint(int(lastYear) + years)

	return &tsv.Table{Header: slices.Clone(written.Header), Rows: rows}, nil
}

func (d Dial) Turn(written *tsv.Table, setting Setting) (*tsv.Table, error) {
	turned, err := d.turn(written, setting)
	if err != nil {
		return nil, err
	}
	return turned.Table, nil
}

type Turned struct {
	Table *tsv.Table

	Wrote int

	LeftAlone []date.Year

	Stopped tsv.ColumnName
}

func (d Dial) turn(written *tsv.Table, setting Setting) (Turned, error) {
	if d.Correction == Postpone {
		table, err := d.turnPostpone(written, setting)
		if err != nil {
			return Turned{}, err
		}
		return Turned{Table: table, Wrote: 1}, nil
	}

	yearAt, settingAt, err := d.columnsOf(written)
	if err != nil {
		return Turned{}, err
	}

	needed, err := d.companionsOf(written)
	if err != nil {
		return Turned{}, err
	}

	out := Turned{}
	before := make([][]string, 0, len(written.Rows))
	after := make([][]string, 0, len(written.Rows))
	atFrom := false
	for row, fields := range written.Rows {
		year, err := date.ParseYear(fields[yearAt])
		if err != nil {
			return Turned{}, fmt.Errorf("breakeven: %s: row %d: %w", d.Slot, row+1, err)
		}

		turned := slices.Clone(fields)
		if year < d.From {
			before = append(before, turned)
			continue
		}
		lacks, err := needed.lacking(fields)
		if err != nil {
			return Turned{}, fmt.Errorf("breakeven: %s: row %d: %w", d.Slot, row+1, err)
		}
		if lacks == "" {
			turned[settingAt] = setting.Field()
			atFrom = atFrom || year == d.From
			out.Wrote++
		} else {
			out.LeftAlone = append(out.LeftAlone, year)
			out.Stopped = lacks
		}
		after = append(after, turned)
	}

	rows := before
	if d.TableKind == input.Step && !atFrom {
		inserted, err := d.rowInEffect(written, yearAt)
		if err != nil {
			return Turned{}, err
		}
		lacks, err := needed.lacking(inserted)
		if err != nil {
			return Turned{}, err
		}
		if lacks == "" {
			inserted = slices.Clone(inserted)
			inserted[yearAt] = fmt.Sprint(d.From)
			inserted[settingAt] = setting.Field()
			rows = append(rows, inserted)
			out.Wrote++
		} else {
			out.LeftAlone = append(out.LeftAlone, d.From)
			out.Stopped = lacks
		}
	}
	rows = append(rows, after...)

	out.Table = &tsv.Table{Header: slices.Clone(written.Header), Rows: rows}
	return out, nil
}

type companions struct{ at map[tsv.ColumnName]int }

func (c companions) lacking(fields []string) (tsv.ColumnName, error) {
	for _, column := range slices.Sorted(maps.Keys(c.at)) {
		n, err := validate.SignOf(fields[c.at[column]])
		if err != nil {
			return "", fmt.Errorf("%s: %w", column, err)
		}
		if n <= 0 {
			return column, nil
		}
	}
	return "", nil
}

func (d Dial) companionsOf(written *tsv.Table) (companions, error) {
	at := make(map[tsv.ColumnName]int)
	for _, pair := range input.PositiveTogether() {
		if pair.Slot != d.Slot || pair.Positive != d.Column {
			continue
		}
		i, ok := written.ColumnIndex(pair.Needed)
		if !ok {
			return companions{}, fmt.Errorf(
				"breakeven: %s に %q 列が無い。%q を回すにはその列が要る（%s）",
				d.Slot, pair.Needed, d.Column, pair.Why)
		}
		at[pair.Needed] = i
	}
	return companions{at: at}, nil
}

func (d Dial) Written(written *tsv.Table) (Setting, error) {
	if d.Correction == Postpone {
		return YearsSetting(0), nil
	}

	yearAt, settingAt, err := d.columnsOf(written)
	if err != nil {
		return Setting{}, err
	}

	fields, err := d.rowInEffect(written, yearAt)
	if err != nil {
		return Setting{}, err
	}
	setting, err := d.Kind.Parse(fields[settingAt])
	if err != nil {
		return Setting{}, fmt.Errorf("breakeven: %s: %q: %w", d.Slot, d.Column, err)
	}
	return setting, nil
}

func (d Dial) columnsOf(written *tsv.Table) (yearAt, settingAt int, err error) {
	yearAt, ok := written.ColumnIndex(d.YearColumn)
	if !ok {
		return 0, 0, fmt.Errorf("breakeven: %s has no %q column", d.Slot, d.YearColumn)
	}
	settingAt, ok = written.ColumnIndex(d.Column)
	if !ok {
		return 0, 0, fmt.Errorf("breakeven: %s has no %q column", d.Slot, d.Column)
	}
	return yearAt, settingAt, nil
}

func (d Dial) rowInEffect(written *tsv.Table, yearAt int) ([]string, error) {
	rows := make([]relation.Row[[]string], 0, len(written.Rows))
	for row, fields := range written.Rows {
		year, err := date.ParseYear(fields[yearAt])
		if err != nil {
			return nil, fmt.Errorf("breakeven: %s: row %d: %w", d.Slot, row+1, err)
		}
		rows = append(rows, relation.Row[[]string]{Year: year, Value: fields})
	}

	found, err := d.rowOf(rows)
	if err != nil {
		return nil, fmt.Errorf(
			"breakeven: %s は %d 年について何も言えない。掃引の起点になる設定が無い: %w",
			d.Slot, d.From, err)
	}
	return found, nil
}

func (d Dial) rowOf(rows []relation.Row[[]string]) ([]string, error) {
	if d.TableKind != input.Step {
		return nil, fmt.Errorf(
			"breakeven: %s のダイヤルの表の種類が %v である。段階の表しか回せない",
			d.Slot, d.TableKind)
	}
	return stepfn.At(rows, d.From)
}

func ParseDial(named string, from date.Year) (Dial, error) {
	slot, column, ok := strings.Cut(named, ":")
	if !ok {
		return Dial{}, fmt.Errorf("breakeven.ParseDial: %q は slot:列名 の形で書くこと", named)
	}
	return DialOf(tsv.Slot(slot), tsv.ColumnName(column), from)
}
