package stepfn

import (
	"errors"
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/wording"
)

func Expand[T any](written []relation.Row[T], from, to date.Year) (relation.Table[T], error) {
	var empty relation.Table[T]

	if from > to {
		return empty, fmt.Errorf("stepfn.Expand: the span runs backwards: from %d to %d", from, to)
	}

	if err := assertAscending(written); err != nil {
		return empty, err
	}

	years := make([]date.Year, 0, len(written))
	for _, row := range written {
		years = append(years, row.Year)
	}
	if err := NoValueYet(from, years); err != nil {
		return empty, fmt.Errorf("stepfn.Expand: %w", err)
	}

	next := 0
	var current T
	return relation.Over(relation.Span(from, to), func(y date.Year) T {
		for next < len(written) && written[next].Year <= y {
			current = written[next].Value
			next++
		}
		return current
	}), nil
}

func OutOfOrder(previous, current date.Year) error {
	switch {
	case current == previous:
		return errors.New(wording.DuplicateKeyFinding(yearKind, wording.Number(current),
			wording.WhichRowAppliesInTheYear))
	case current < previous:
		return errors.New(wording.OutOfAscendingOrderFinding(yearKind,
			wording.Number(previous), wording.Number(current)))
	default:
		return nil
	}
}

func NoValueYet(wanted date.Year, written []date.Year) error {
	if len(written) == 0 {
		return fmt.Errorf("nothing is written, so year %d has no value (an unrecorded value is not zero)", wanted)
	}

	earliest := written[0]
	for _, year := range written {
		earliest = min(earliest, year)
	}
	if earliest > wanted {
		return fmt.Errorf(
			"the first year written is %d, after the plan starts in %d, so the years between have no value (an unrecorded value is not zero)",
			earliest, wanted)
	}
	return nil
}

func assertAscending[T any](written []relation.Row[T]) error {
	for i := 1; i < len(written); i++ {
		if err := OutOfOrder(written[i-1].Year, written[i].Year); err != nil {
			return err
		}
	}
	return nil
}

const yearKind = "西暦"
