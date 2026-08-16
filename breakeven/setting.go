package breakeven

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

type Setting struct {
	field string

	at *big.Rat

	unit string
}

func (s Setting) Field() string { return s.field }

func (s Setting) String() string { return s.field }

func (s Setting) IsZero() bool { return s.at == nil }

func (s Setting) Cmp(t Setting) int {
	if !s.Comparable(t) {
		panic(fmt.Sprintf("breakeven: %s（%s）と %s（%s）には順序が無い", s, s.unit, t, t.unit))
	}
	return s.value().Cmp(t.value())
}

func (s Setting) Comparable(t Setting) bool { return s.unit == t.unit }

func (s Setting) Unit() string { return s.unit }

func (s Setting) value() *big.Rat {
	if s.at == nil {
		return new(big.Rat)
	}
	return s.at
}

type Kind interface {
	Parse(field string) (Setting, error)

	Of(steps int) (Setting, error)

	Unit() string
}

type Percent struct{}

func (Percent) Unit() string { return "%" }

func (Percent) Of(steps int) (Setting, error) {
	return RateSetting(money.NewRate(int64(steps), 10_000))
}

func (Percent) Parse(field string) (Setting, error) {
	rate, err := money.ParsePercent(field)
	if err != nil {
		return Setting{}, err
	}
	return RateSetting(rate)
}

type Years struct{}

func (Years) Unit() string { return "年" }

func (Years) Of(steps int) (Setting, error) {
	if steps < 0 {
		return Setting{}, fmt.Errorf("breakeven: 年数は負にできない: %d", steps)
	}
	return YearsSetting(steps), nil
}

func (Years) Parse(field string) (Setting, error) {
	n, err := strconv.Atoi(strings.TrimSpace(field))
	if err != nil {
		return Setting{}, fmt.Errorf("breakeven: %q は年数として読めない: %w", field, err)
	}
	return Years{}.Of(n)
}

func YearsSetting(years int) Setting {
	return Setting{
		field: strconv.Itoa(years), at: new(big.Rat).SetInt64(int64(years)), unit: Years{}.Unit(),
	}
}

func yearsOf(s Setting) (int, error) {
	n, err := strconv.Atoi(s.Field())
	if err != nil {
		return 0, fmt.Errorf("breakeven: %s は年数として読めない: %w", s, err)
	}
	return n, nil
}

func KindOf(slot tsv.Slot, column tsv.ColumnName) (Kind, error) {
	for _, shape := range input.Shapes() {
		if shape.Slot != slot {
			continue
		}
		for _, c := range shape.Columns {
			if c.Name != column {
				continue
			}
			switch c.Unit {
			case validate.UnitPercent:
				return Percent{}, nil
			case validate.UnitYen:
				return Yen{}, nil
			case "":
				return nil, fmt.Errorf(
					"breakeven.KindOf: %s の %q は単位が宣言されていない列である（input/shape.go）",
					slot, column)
			default:
				return nil, fmt.Errorf(
					"breakeven.KindOf: %s の %q は %s の列で、掃引できない。掃引できるのは 円 と %% の列である",
					slot, column, c.Unit)
			}
		}
		return nil, fmt.Errorf("breakeven.KindOf: %s に %q という列は無い", slot, column)
	}
	return nil, fmt.Errorf("breakeven.KindOf: %q という slot は無い", slot)
}

type Yen struct{}

func (Yen) Unit() string { return "円" }

func (Yen) Of(steps int) (Setting, error) { return YenSetting(money.Yen(steps)), nil }

func (Yen) Parse(field string) (Setting, error) {
	amount, err := money.ParseYen(field)
	if err != nil {
		return Setting{}, err
	}
	return YenSetting(amount), nil
}

func YenSetting(amount money.Yen) Setting {
	return Setting{field: amount.String(), at: new(big.Rat).SetInt64(int64(amount)), unit: Yen{}.Unit()}
}

func RateSetting(rate money.Rate) (Setting, error) {
	field := rate.Percent()

	written, err := money.ParsePercent(field)
	if err != nil || written.Cmp(rate) != 0 {
		return Setting{}, fmt.Errorf(
			"breakeven: %s は率の表に書ける刻みではない。書けるのは 0.01pt 刻みまでである", rate)
	}
	return Setting{
		field: field, at: new(big.Rat).SetFrac64(rate.Num(), rate.Den()), unit: Percent{}.Unit(),
	}, nil
}
