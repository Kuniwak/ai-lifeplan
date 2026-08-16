package law

import "testing"

func TestKanjiYenShouldReadTheAmountsStatutesAreWrittenIn(t *testing.T) {
	testCases := map[string]struct {
		Kanji    string
		Expected int64
	}{
		"二十七万":   {"二十七万", 270_000},
		"四十万":    {"四十万", 400_000},
		"七十五万":   {"七十五万", 750_000},
		"二十六万":   {"二十六万", 260_000},
		"三十万":    {"三十万", 300_000},
		"五十三万":   {"五十三万", 530_000},
		"三十八万":   {"三十八万", 380_000},
		"四十八万":   {"四十八万", 480_000},
		"五十八万":   {"五十八万", 580_000},
		"六百二十二万": {"六百二十二万", 6_220_000},
		"八百五十八万": {"八百五十八万", 8_580_000},
		"四十四万":   {"四十四万", 440_000},
		"七十八万九百": {"七十八万九百", 780_900},
		"千万":     {"千万", 10_000_000},
		"千":      {"千", 1_000},
		"一":      {"一", 1},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := ParseKanjiNumber(tc.Kanji)

			if err != nil {
				t.Fatalf("ParseKanjiNumber(%q): %v", tc.Kanji, err)
			}
			if got != tc.Expected {
				t.Errorf("ParseKanjiNumber(%q) = %d, want %d", tc.Kanji, got, tc.Expected)
			}
		})
	}
}

func TestKanjiYenShouldRefuseWhatItCannotRead(t *testing.T) {
	for _, s := range []string{"", "二十七万円", "270000", "二十七マン", "億"} {
		t.Run(s, func(t *testing.T) {
			if got, err := ParseKanjiNumber(s); err == nil {
				t.Errorf("ParseKanjiNumber(%q) = %d, want an error", s, got)
			}
		})
	}
}
