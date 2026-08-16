package money

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

type Rate struct {
	num int64
	den int64
}

func NewRate(num, den int64) Rate {
	if den == 0 {
		panic("money: NewRate needs a non-zero denominator")
	}
	return Rate{num: num, den: den}
}

func NewPercent(n int64) Rate {
	return NewRate(n, 100)
}

func ParsePercent(s string) (Rate, error) {
	trimmed := strings.TrimSpace(s)

	body, found := strings.CutSuffix(trimmed, "%")
	if !found {
		return Rate{}, fmt.Errorf("money.ParsePercent: %q has no %% sign (a bare number is ambiguous between a percentage and a multiplier)", s)
	}

	intPart, fracPart, _ := strings.Cut(body, ".")

	digits := intPart + fracPart
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return Rate{}, fmt.Errorf("money.ParsePercent: invalid percentage %q", s)
	}

	den := int64(100)
	for range fracPart {
		den *= 10
	}

	return NewRate(n, den), nil
}

func (r Rate) Percent() string {
	n, d := r.num, r.den
	if d == 0 {
		panic("money.Rate.Percent: the zero Rate is not a rate; it has no denominator")
	}
	if d < 0 {
		n, d = -n, -d
	}

	hundredths := n * 10_000 / d
	sign := ""
	if hundredths < 0 {
		sign, hundredths = "-", -hundredths
	}
	return fmt.Sprintf("%s%d.%02d%%", sign, hundredths/100, hundredths%100)
}

func (r Rate) String() string {
	return fmt.Sprintf("%d/%d", r.num, r.den)
}

type Rounding func(n, d int64) int64

func Truncate(n, d int64) int64 {
	q := n / d
	if n%d != 0 && (n < 0) != (d < 0) {
		q--
	}
	return q
}

func Ceil(n, d int64) int64 {
	q := n / d
	if n%d != 0 && (n < 0) == (d < 0) {
		q++
	}
	return q
}

func HalfUp(n, d int64) int64 {
	if d < 0 {
		n, d = -n, -d
	}
	if n >= 0 {
		return (2*n + d) / (2 * d)
	}
	return -((-2*n + d) / (2 * d))
}

func (y Yen) Mul(r Rate, round Rounding) Yen {
	if round == nil {
		panic("money: Mul needs an explicit Rounding; the statute decides how a rate is rounded")
	}
	return Yen(round(int64(y)*r.num, r.den))
}

func ShareOf(amount, part, whole Yen) Yen {
	if whole <= 0 {
		return 0
	}
	out := new(big.Int).SetInt64(int64(amount))
	out.Mul(out, big.NewInt(int64(part)))
	return Yen(out.Div(out, big.NewInt(int64(whole))).Int64())
}

func (r Rate) Compound(s Rate) Rate {
	num, den := r.num*s.den+s.num*r.den+r.num*s.num, r.den*s.den

	if g := gcd(num, den); g > 1 {
		num, den = num/g, den/g
	}
	return NewRate(num, den)
}

func (r Rate) Times(s Rate) Rate {
	num, den := r.num*s.num, r.den*s.den
	if g := gcd(num, den); g > 1 {
		num, den = num/g, den/g
	}
	return NewRate(num, den)
}

func gcd(a, b int64) int64 {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func (r Rate) Cmp(s Rate) int {
	left, right := r.num*s.den, s.num*r.den
	if (r.den < 0) != (s.den < 0) {
		left, right = right, left
	}
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func (r Rate) Div(n int64) Rate {
	if n == 0 {
		panic("money.Rate.Div: a rate spread over no periods is not a rate")
	}
	return NewRate(r.num, r.den*n)
}

func (r Rate) Float64() float64 {
	return float64(r.num) / float64(r.den)
}

func (r Rate) Num() int64 { return r.num }
func (r Rate) Den() int64 { return r.den }

func (r Rate) Plus(s Rate) Rate {
	num, den := r.num*s.den+s.num*r.den, r.den*s.den
	if g := gcd(num, den); g > 1 {
		num, den = num/g, den/g
	}
	return NewRate(num, den)
}
