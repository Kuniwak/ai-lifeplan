package law

import (
	"testing"
)

func TestTheResidentialLandFractionsShouldComeFromTheStatute(t *testing.T) {
	type testCase struct {
		Article     string
		Pattern     string
		Denominator int64
	}
	testCases := map[string]testCase{
		"固定資産税の小規模住宅用地": {
			Article:     "地方税法-第349条の3の2.xml",
			Pattern:     `当該小規模住宅用地に係る固定資産税の課税標準となるべき価格の(` + theKanjiNumeral + `)分の一の額`,
			Denominator: 6,
		},
		"都市計画税の小規模住宅用地": {
			Article:     "地方税法-第702条の3.xml",
			Pattern:     `第三百四十九条の三の二第二項の規定の適用を受ける土地に対して課する都市計画税の課税標準は、[^。]*価格の(` + theKanjiNumeral + `)分の一の額`,
			Denominator: 3,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			article := egovArticle(t, tc.Article)

			got := egovAmount(t, article, tc.Pattern)

			if got != tc.Denominator {
				t.Errorf("%s の分母が条文では %d。表は %d を持っている", name, got, tc.Denominator)
			}
		})
	}
}
