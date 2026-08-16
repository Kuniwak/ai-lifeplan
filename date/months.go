package date

import (
	"fmt"
	"strconv"
	"strings"
)

const MonthsAYear = 12

type Months uint16

const (
	NoMonths  Months = 0
	WholeYear Months = 1<<MonthsAYear - 1
)

func MonthsOfYearIn(year Year, from, through Date) Months {
	first, last := 1, MonthsAYear
	switch {
	case from.Year > year:
		return NoMonths
	case from.Year == year:
		first = from.Month
	}
	switch {
	case through.Year < year:
		return NoMonths
	case through.Year == year:
		last = through.Month
	}
	if last < first {
		return NoMonths
	}
	return Months(1<<last-1) &^ Months(1<<(first-1)-1)
}

func MonthsOfYearFromIn(year Year, from Date) Months {
	return MonthsOfYearIn(year, from, Date{Year: year, Month: 12, Day: 31})
}

func MonthOnly(month int) Months {
	if month < 1 || month > MonthsAYear {
		return NoMonths
	}
	return Months(1) << (month - 1)
}

func ParseMonths(written string) (Months, error) {
	written = strings.TrimSpace(written)
	if written == "" {
		return NoMonths, nil
	}

	var months Months
	for _, field := range strings.Split(written, ",") {
		field = strings.TrimSpace(field)
		month, err := strconv.Atoi(field)
		if err != nil {
			return NoMonths, fmt.Errorf("date.ParseMonths: %q が月の番号でない（%q のうち）", field, written)
		}
		only := MonthOnly(month)
		if only == NoMonths {
			return NoMonths, fmt.Errorf("date.ParseMonths: %d 月は 1〜12 でない（%q のうち）", month, written)
		}
		if months.Intersect(only) != NoMonths {
			return NoMonths, fmt.Errorf("date.ParseMonths: %d 月が二度書かれている（%q）", month, written)
		}
		months = months.Union(only)
	}
	return months, nil
}

func (m Months) Count() int {
	n := 0
	for month := 1; month <= MonthsAYear; month++ {
		if m.Has(month) {
			n++
		}
	}
	return n
}

func (m Months) Has(month int) bool {
	if month < 1 || month > MonthsAYear {
		return false
	}
	return m&(1<<(month-1)) != 0
}

func (m Months) Union(other Months) Months     { return m | other }
func (m Months) Intersect(other Months) Months { return m & other }

func (m Months) String() string {
	if m == NoMonths {
		return "なし"
	}
	var b strings.Builder
	for month := 1; month <= MonthsAYear; month++ {
		if !m.Has(month) {
			continue
		}
		last := month
		for last < MonthsAYear && m.Has(last+1) {
			last++
		}
		if b.Len() > 0 {
			b.WriteString("・")
		}
		if last == month {
			fmt.Fprintf(&b, "%d月", month)
		} else {
			fmt.Fprintf(&b, "%d〜%d月", month, last)
		}
		month = last
	}
	return b.String()
}

func FirstOfMonth(d Date) Date {
	return Date{Year: d.Year, Month: d.Month, Day: 1}
}

func MonthsBetween(from, through Date) int {
	return int(through.Year-from.Year)*MonthsAYear + (through.Month - from.Month) + 1
}
