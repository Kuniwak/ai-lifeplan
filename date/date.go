package date

import (
	"fmt"
	"strconv"
	"strings"
)

type Date struct {
	Year  Year
	Month int
	Day   int
}

func Parse(field string) (Date, error) {
	parts := strings.Split(field, "-")
	if len(parts) != 3 {
		return Date{}, fmt.Errorf("date.Parse: %q is not a date; write it as YYYY-MM-DD", field)
	}

	numbers := make([]int, 3)
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return Date{}, fmt.Errorf("date.Parse: %q is not a date; write it as YYYY-MM-DD", field)
		}
		numbers[i] = n
	}

	date, err := New(Year(numbers[0]), numbers[1], numbers[2])
	if err != nil {
		return Date{}, fmt.Errorf("date.Parse: %q: %w", field, err)
	}
	return date, nil
}

func New(year Year, month, day int) (Date, error) {
	if month < 1 || month > MonthsAYear {
		return Date{}, fmt.Errorf("date.New: there is no month %d", month)
	}
	if day < 1 || day > daysIn(year, month) {
		return Date{}, fmt.Errorf("date.New: there is no day %d in month %d of %d", day, month, year)
	}
	return Date{Year: year, Month: month, Day: day}, nil
}

func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

func daysIn(year Year, month int) int {
	switch month {
	case 2:
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	default:
		return 31
	}
}

func (d Date) AgeInMonth(year Year, month int) int {
	age := int(year - d.Year)
	if month < d.Month {
		age--
	}
	return age
}

func (d Date) ReachesAge(n int) Date {
	year, month := d.Year+Year(n), d.Month
	if d.Day > daysIn(year, month) {
		return Date{Year: year, Month: month, Day: daysIn(year, month)}
	}
	return Date{Year: year, Month: month, Day: d.Day}.DayBefore()
}

func (d Date) Anniversary(n int) Date {
	year, month := d.Year+Year(n), d.Month
	return Date{Year: year, Month: month, Day: min(d.Day, daysIn(year, month))}
}

func (d Date) DayBefore() Date {
	if d.Day > 1 {
		return Date{Year: d.Year, Month: d.Month, Day: d.Day - 1}
	}
	if d.Month > 1 {
		return Date{Year: d.Year, Month: d.Month - 1, Day: daysIn(d.Year, d.Month-1)}
	}
	return Date{Year: d.Year - 1, Month: 12, Day: 31}
}

func (d Date) Before(other Date) bool {
	if d.Year != other.Year {
		return d.Year < other.Year
	}
	if d.Month != other.Month {
		return d.Month < other.Month
	}
	return d.Day < other.Day
}

func (d Date) SchoolYearEnd() Date {
	const march, thirtyFirst = 3, 31
	if d.Month > march || (d.Month == march && d.Day > thirtyFirst) {
		return Date{Year: d.Year + 1, Month: march, Day: thirtyFirst}
	}
	return Date{Year: d.Year, Month: march, Day: thirtyFirst}
}

func (d Date) AddMonths(n int) Date {
	months := (int(d.Year)*MonthsAYear + d.Month - 1) + n
	return Date{Year: Year(months / MonthsAYear), Month: months%MonthsAYear + 1, Day: d.Day}
}
