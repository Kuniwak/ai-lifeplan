package law

import "fmt"

var (
	theKanjiDigits = map[rune]int64{
		'一': 1, '二': 2, '三': 3, '四': 4, '五': 5,
		'六': 6, '七': 7, '八': 8, '九': 9,
	}
	theKanjiSmallUnits = map[rune]int64{'十': 10, '百': 100, '千': 1_000}
	theKanjiBigUnits   = map[rune]int64{'万': 10_000, '億': 100_000_000}
)

func ParseKanjiNumber(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("law.ParseKanjiNumber: empty")
	}

	var total, section, current int64
	seen := false
	for _, r := range s {
		switch {
		case theKanjiDigits[r] != 0:
			current = theKanjiDigits[r]
			seen = true
		case theKanjiSmallUnits[r] != 0:
			if current == 0 {
				current = 1
			}
			section += current * theKanjiSmallUnits[r]
			current = 0
			seen = true
		case theKanjiBigUnits[r] != 0:
			section += current
			if section == 0 {
				return 0, fmt.Errorf("law.ParseKanjiNumber: %q has %q with nothing before it", s, string(r))
			}
			total += section * theKanjiBigUnits[r]
			section, current = 0, 0
		default:
			return 0, fmt.Errorf("law.ParseKanjiNumber: %q has %q, which is not a numeral", s, string(r))
		}
	}
	if !seen {
		return 0, fmt.Errorf("law.ParseKanjiNumber: %q has no numeral", s)
	}
	return total + section + current, nil
}
