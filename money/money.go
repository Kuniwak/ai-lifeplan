package money

import (
	"fmt"
	"strconv"
	"strings"
)

type Yen int64

func (y Yen) String() string {
	return strconv.FormatInt(int64(y), 10)
}

func ParseYen(s string) (Yen, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return 0, fmt.Errorf("money.ParseYen: empty amount (an unrecorded value is not zero)")
	}

	digits := strings.NewReplacer(",", "", "_", "").Replace(trimmed)

	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("money.ParseYen: invalid amount %q: only digits, a sign and the separators \",\" and \"_\" are allowed", s)
	}

	return Yen(n), nil
}

func (y Yen) Times(n int) Yen { return y * Yen(n) }
