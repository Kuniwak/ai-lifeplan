package money

import (
	"fmt"
	"strings"
)

type PriceMove struct {
	rate Rate

	difference bool
}

func RatioMove(r Rate) PriceMove { return PriceMove{rate: r} }

func DifferenceMove(r Rate) PriceMove { return PriceMove{rate: r, difference: true} }

func (m PriceMove) Applied(general Rate) Rate {
	if m.difference {
		return general.Plus(m.rate)
	}
	return general.Times(m.rate)
}

func (m PriceMove) IsDifference() bool { return m.difference }

func (m PriceMove) Rate() Rate { return m.rate }

const PointsSuffix = "pt"

func ParsePriceMove(s string) (PriceMove, error) {
	trimmed := strings.TrimSpace(s)

	body, isDifference := strings.CutSuffix(trimmed, PointsSuffix)
	if !isDifference {
		if strings.HasPrefix(trimmed, "+") {
			return PriceMove{}, fmt.Errorf(
				"money.ParsePriceMove: %q は比なのに符号が付いている。差なら %s で終わること", s, PointsSuffix)
		}
		rate, err := ParsePercent(trimmed)
		if err != nil {
			return PriceMove{}, fmt.Errorf(
				"money.ParsePriceMove: %q は比（%%）でも差（%s）でもない: %w", s, PointsSuffix, err)
		}
		return RatioMove(rate), nil
	}

	negative := strings.HasPrefix(body, "-")
	if !negative && !strings.HasPrefix(body, "+") {
		return PriceMove{}, fmt.Errorf(
			"money.ParsePriceMove: %q に符号が無い。差は一般物価より速いか遅いかを言うものなので、"+
				"+0.541%s のように向きを書くこと", s, PointsSuffix)
	}
	magnitude := body[len("+"):]
	if strings.ContainsAny(magnitude, "+-") || strings.TrimSpace(magnitude) != magnitude {
		return PriceMove{}, fmt.Errorf(
			"money.ParsePriceMove: %q の符号のあとが %q である。符号は 1 つで、そのあとは数だけである",
			s, magnitude)
	}
	rate, err := ParsePercent(magnitude + "%")
	if err != nil {
		return PriceMove{}, fmt.Errorf("money.ParsePriceMove: %q が差として読めない: %w", s, err)
	}
	if negative {
		rate = NewRate(-rate.Num(), rate.Den())
	}

	if floor := NewRate(-1, 1); rate.Cmp(floor) <= 0 {
		return PriceMove{}, fmt.Errorf(
			"money.ParsePriceMove: %q は一般物価に足すと物価を負にする。差は %s より大きくなければならない",
			s, floor.Percent())
	}
	return DifferenceMove(rate), nil
}

func (m PriceMove) String() string {
	if !m.difference {
		return m.rate.Percent()
	}
	percent := m.rate.Percent()
	if strings.HasPrefix(percent, "-") {
		return strings.TrimSuffix(percent, "%") + PointsSuffix
	}
	return "+" + strings.TrimSuffix(percent, "%") + PointsSuffix
}
