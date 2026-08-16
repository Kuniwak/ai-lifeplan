package stepfn

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/wording"
)

func At[T any](written []relation.Row[T], at date.Year) (T, error) {
	var found *relation.Row[T]
	twice := false
	for i := range written {
		row := written[i]
		if row.Year > at {
			continue
		}
		switch {
		case found == nil || row.Year > found.Year:
			found, twice = &row, false
		case row.Year == found.Year:
			twice = true
		}
	}

	var empty T
	if found == nil {
		if len(written) == 0 {
			return empty, fmt.Errorf(
				"stepfn.At: nothing is written, so year %d has no value (an unrecorded value is not zero)", at)
		}
		return empty, fmt.Errorf(
			"stepfn.At: year %d comes before the first written year %d, so it has no value (an unrecorded value is not zero)",
			at, earliestYear(written))
	}
	if twice {
		return empty, wording.DuplicateKeyError("stepfn.At", "year", wording.Number(found.Year),
			wording.Undecided(fmt.Sprintf("which row applies in %d", at)))
	}
	return found.Value, nil
}

func earliestYear[T any](written []relation.Row[T]) date.Year {
	earliest := written[0].Year
	for _, row := range written[1:] {
		earliest = min(earliest, row.Year)
	}
	return earliest
}
