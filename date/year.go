package date

import (
	"fmt"
	"strconv"
)

type Year int

func ParseYear(field string) (Year, error) {
	n, err := strconv.Atoi(field)
	if err != nil {
		return 0, fmt.Errorf("ParseYear: %q is not a year", field)
	}
	return Year(n), nil
}
