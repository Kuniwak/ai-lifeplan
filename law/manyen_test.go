package law

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Kuniwak/lifeplan/money"
)

func parseManYen(field string) (money.Yen, error) {
	whole, fraction, _ := strings.Cut(strings.TrimSpace(field), ".")

	negative := strings.HasPrefix(whole, "-")
	if negative {
		whole = whole[1:]
	}

	man, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parseManYen: %q: %w", field, err)
	}

	yen := man * 10_000
	if fraction != "" {
		scale := int64(10_000)
		for _, digit := range fraction {
			if digit < '0' || digit > '9' {
				return 0, fmt.Errorf("parseManYen: %q has a non-digit after the point", field)
			}
			scale /= 10
			yen += int64(digit-'0') * scale
		}
	}

	if negative {
		yen = -yen
	}
	return money.Yen(yen), nil
}
