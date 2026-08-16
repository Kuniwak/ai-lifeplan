package validate

import (
	"fmt"
	"strconv"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

type Parser func(field string) error

func AsText(field string) error {
	if isBlank(field) {
		return fmt.Errorf("the field is blank (a blank is not a name; leave the row out instead)")
	}
	return nil
}

func isBlank(field string) bool { return field == "" }

func AsYear(field string) error {
	if _, err := strconv.Atoi(field); err != nil {
		return fmt.Errorf("%q is not a year", field)
	}
	return nil
}

func AsDate(field string) error {
	if _, err := date.Parse(field); err != nil {
		return fmt.Errorf("%q is not a date; write it as YYYY-MM-DD", field)
	}
	return nil
}

func AsCount(field string) error {
	n, err := strconv.Atoi(field)
	if err != nil {
		return fmt.Errorf("%q is not a count", field)
	}
	if n < 0 {
		return fmt.Errorf("%q counts less than nothing", field)
	}
	return nil
}

func AsMonths(field string) error {
	if _, err := date.ParseMonths(field); err != nil {
		return err
	}
	return nil
}

func AsYen(field string) error {
	if _, err := money.ParseYen(field); err != nil {
		return err
	}
	return nil
}

func AsPercent(field string) error {
	if _, err := money.ParsePercent(field); err != nil {
		return err
	}
	return nil
}

func AsPercentAtLeast(num, den int64) Parser {
	floor := money.NewRate(num, den)
	return func(field string) error {
		rate, err := money.ParsePercent(field)
		if err != nil {
			return err
		}
		if rate.Cmp(floor) < 0 {
			return fmt.Errorf("%s は %s を下回っている", rate.Percent(), floor.Percent())
		}
		return nil
	}
}

func AsYenOnly(only money.Yen, because string) Parser {
	return func(field string) error {
		yen, err := money.ParseYen(field)
		if err != nil {
			return err
		}
		if yen != only {
			return fmt.Errorf("%d 円は %d 円でなければならない。%s", yen, only, because)
		}
		return nil
	}
}

func AsPriceMove(field string) error {
	if _, err := money.ParsePriceMove(field); err != nil {
		return err
	}
	return nil
}

func AsYenAtMost(ceiling money.Yen) Parser {
	return func(field string) error {
		yen, err := money.ParseYen(field)
		if err != nil {
			return err
		}
		if yen < 0 {
			return fmt.Errorf("%d 円は 0 円を下回っている", yen)
		}
		if yen > ceiling {
			return fmt.Errorf("%d 円は上限の %d 円を超えている", yen, ceiling)
		}
		return nil
	}
}

func AsOneOf(words ...string) Parser {
	allowed := OneOf(words...)
	return func(field string) error {
		if !allowed(field) {
			return fmt.Errorf("%q は %v のどれでもない", field, words)
		}
		return nil
	}
}

func AsOptional(parse Parser) Parser {
	return func(field string) error {
		if isBlank(field) {
			return nil
		}
		return parse(field)
	}
}

type Unit string

const (
	UnitYen     Unit = "円"
	UnitPercent Unit = "%"
	UnitYear    Unit = "年"
	UnitCount   Unit = "個"
	UnitDate    Unit = "年月日"

	UnitMonths Unit = "月の集合"

	UnitPriceMove Unit = "比または差"

	UnitText Unit = "テキスト"
)

type Column struct {
	Name tsv.ColumnName

	Unit Unit

	Parse Parser
}
