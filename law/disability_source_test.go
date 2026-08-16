package law

import "testing"

func TestTheDisabilityDeductionShouldAgreeWithTheStatutes(t *testing.T) {
	incomeTaxArticle := egovArticle(t, "所得税法-第79条.xml")
	residentArticle := egovArticle(t, "地方税法-第34条.xml")

	type testCase struct {
		IncomeTax string
		Resident  string
	}
	testCases := map[DisabilityCategoryValue]testCase{
		OrdinaryDisability: {
			IncomeTax: `その障害者一人につき(` + theKanjiNumeral + `)円`,
			Resident:  `各障害者につき(` + theKanjiNumeral + `)円`,
		},
		SpecialDisability: {
			IncomeTax: `その障害者一人につき` + theKanjiNumeral + `円（その者が特別障害者である場合には、(` + theKanjiNumeral + `)円）`,
			Resident: `各障害者につき` + theKanjiNumeral + `円（その者が特別障害者（[^）]*）である場合には、(` +
				theKanjiNumeral + `)円）`,
		},
		CohabitingSpecialDisability: {
			IncomeTax: `その特別障害者一人につき(` + theKanjiNumeral + `)円を控除する`,
			Resident:  `第一項第六号の金額は、(` + theKanjiNumeral + `)円とする`,
		},
	}

	loaded := disabilityTable(t)

	for category, tc := range testCases {
		t.Run(string(category), func(t *testing.T) {
			got, err := loaded.Lookup(category)
			if err != nil {
				t.Fatalf("Lookup(%q): %v", category, err)
			}

			if want := egovAmount(t, incomeTaxArticle, tc.IncomeTax); int64(got.IncomeTax) != want {
				t.Errorf("%s の所得税控除額が %d 円。所得税法第79条は %d 円と定めている",
					category, got.IncomeTax, want)
			}
			if want := egovAmount(t, residentArticle, tc.Resident); int64(got.Resident) != want {
				t.Errorf("%s の住民税控除額が %d 円。地方税法第34条は %d 円と定めている",
					category, got.Resident, want)
			}
		})
	}

	if got := loaded.Categories(); len(got) != len(testCases) {
		t.Errorf("表の区分が %v。条文が定める 3 区分のはず", got)
	}
}
