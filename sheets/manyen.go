package sheets

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Kuniwak/lifeplan/money"
)

const manYenDigits = 4

func ManYen(s string) (money.Yen, error) {
	s = strings.ReplaceAll(s, ",", "")
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")

	whole, frac, _ := strings.Cut(s, ".")
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("sheets.ManYen: %q: %w", s, err)
	}

	v := money.Yen(w) * 10_000
	if frac != "" {
		if len(frac) > manYenDigits {
			return 0, fmt.Errorf("sheets.ManYen: %q: more than %d decimal places of 万円 cannot be held in whole yen", s, manYenDigits)
		}
		f, err := strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("sheets.ManYen: %q: %w", s, err)
		}
		for i := len(frac); i < manYenDigits; i++ {
			f *= 10
		}
		v += money.Yen(f)
	}
	if neg {
		v = -v
	}
	return v, nil
}
