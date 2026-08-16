package law

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/panictest"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

func TestAYearBelowTheRecord(t *testing.T) {
	const name = "national/例の表"

	type testCase struct {
		FirstStartYear string
		Year           date.Year

		WantRefused bool
		WantAmount  money.Yen
	}

	testCases := map[string]testCase{
		"年が書いてあり、その年を引く（境界値）": {
			FirstStartYear: "2005", Year: 2005, WantAmount: 13_580,
		},
		"年が書いてあり、その 1 年前を引く（境界値）": {
			FirstStartYear: "2005", Year: 2004, WantRefused: true,
		},
		"年が書いてあり、はるか前を引く": {
			FirstStartYear: "2005", Year: 1900, WantRefused: true,
		},
		"不明 と書いてあれば、下限として答える": {
			FirstStartYear: string(validate.Unknown), Year: 1900, WantAmount: 13_580,
		},
		"表の後ろは最後の行が立つ": {
			FirstStartYear: "2005", Year: 2100, WantAmount: 16_520,
		},
	}

	for caseName, tc := range testCases {
		t.Run(caseName, func(t *testing.T) {
			amounts, err := ParseYearYenTable(&tsv.Table{
				Header: []tsv.ColumnName{LawStartYearColumn, NationalPensionPremiumColumn, LawEndYearColumn},
				Rows: [][]string{
					{tc.FirstStartYear, "13580", "2022"},
					{"2023", "16520", string(validate.Indefinite)},
				},
			}, name, NationalPensionPremiumColumn)
			if err != nil {
				t.Fatalf("ParseYearYenTable: %v", err)
			}

			var got money.Yen

			refused := panictest.Recovered(func() { got = amounts.Amount(tc.Year) })

			if !tc.WantRefused {
				if refused != nil {
					t.Fatalf("%d 年が拒まれた: %v", tc.Year, refused)
				}
				if got != tc.WantAmount {
					t.Errorf("Amount(%d) = %d, want %d", tc.Year, got, tc.WantAmount)
				}
				return
			}

			if refused == nil {
				t.Fatalf("記録より前の %d 年が黙って %d と答えられた", tc.Year, got)
			}
			message, ok := refused.(string)
			if !ok {
				t.Fatalf("panic の中身が文字列でない: %#v", refused)
			}
			for _, want := range []string{name, "2004", "2005"} {
				if tc.Year != 2004 && want == "2004" {
					continue
				}
				if !strings.Contains(message, want) {
					t.Errorf("panic のメッセージに %q が無い: %s", want, message)
				}
			}
		})
	}
}

func TestAssertRecordReaches(t *testing.T) {
	type testCase struct {
		FirstWritten date.Year
		HasRows      bool
		From         date.Year
		WantRefused  bool
	}

	testCases := map[string]testCase{
		"記録の始まりと最初に引く年が同じ（境界値）": {FirstWritten: 2016, HasRows: true, From: 2016},
		"計画が記録より後から引く":          {FirstWritten: 2016, HasRows: true, From: 2018},
		"計画が記録の 1 年前から引く（境界値）":  {FirstWritten: 2016, HasRows: true, From: 2015, WantRefused: true},
		"行の無い表は何も言わない":          {HasRows: false, From: 1900},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			first := func() (date.Year, bool) { return tc.FirstWritten, tc.HasRows }

			err := AssertRecordReaches("健康保険料率", first, tc.From)

			if tc.WantRefused {
				if err == nil {
					t.Fatal("記録より前に始まる計画が通った")
				}
				for _, want := range []string{"健康保険料率", "2016", "2015"} {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("エラーに %q が無い: %v", want, err)
					}
				}
				return
			}
			if err != nil {
				t.Errorf("通るはずの計画が拒まれた: %v", err)
			}
		})
	}
}
