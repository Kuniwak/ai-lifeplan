package validate_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/panictest"
	"github.com/Kuniwak/lifeplan/validate"
)

func TestAsCountShouldAcceptAWholeNumberOfThings(t *testing.T) {
	for _, field := range []string{"0", "2", "12", "35"} {
		if err := validate.AsCount(field); err != nil {
			t.Errorf("AsCount(%q) = %v, want nil", field, err)
		}
	}
}

func TestAsCountShouldRefuseWhatCannotBeCounted(t *testing.T) {
	for _, field := range []string{"", "-1", "1.5", "2ヶ月", "two"} {
		if err := validate.AsCount(field); err == nil {
			t.Errorf("AsCount(%q) = nil, want an error", field)
		}
	}
}

func TestEveryParserShouldRefuseABlankField(t *testing.T) {
	type testCase struct {
		Parse validate.Parser
	}
	cases := map[string]testCase{
		"AsText":    {Parse: validate.AsText},
		"AsYear":    {Parse: validate.AsYear},
		"AsCount":   {Parse: validate.AsCount},
		"AsYen":     {Parse: validate.AsYen},
		"AsPercent": {Parse: validate.AsPercent},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.Parse("")

			if err == nil {
				t.Error("a blank field was accepted; blank is not a value of any kind")
			}
		})
	}
}

func TestAsOptionalShouldBeHowAColumnSaysBlankIsAllowed(t *testing.T) {
	parse := validate.AsOptional(validate.AsText)

	if err := parse(""); err != nil {
		t.Errorf("AsOptional refused a blank field: %v", err)
	}
	if err := parse("なにか"); err != nil {
		t.Errorf("AsOptional refused a written field: %v", err)
	}
}

func TestAsPercentAtLeastShouldRefuseWhatIsBelowTheFloor(t *testing.T) {
	const (
		readable   = ""
		unreadable = "読めない"
		below      = "床を下回る"
	)

	cases := map[string]struct {
		floorNum, floorDen int64
		field              string
		fault              string
	}{
		"床の上": {0, 1, "118.8%", readable},
		"床ちょうどは通る（境界値）":   {0, 1, "0.0%", readable},
		"0% と 0.0% は同じ":   {0, 1, "0%", readable},
		"床のすぐ下（境界値）":      {0, 1, "-0.1%", below},
		"はるかに下":           {0, 1, "-118.8%", below},
		"床が 0 でなくても効く":    {1, 1, "99.9%", below},
		"その床ちょうどは通る（境界値）": {1, 1, "100%", readable},
		"率として読めない":        {0, 1, "あいうえお", unreadable},
		"空欄":              {0, 1, "", unreadable},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := validate.AsPercentAtLeast(c.floorNum, c.floorDen)(c.field)

			switch c.fault {
			case readable:
				if err != nil {
					t.Errorf("%q が拒まれた: %v", c.field, err)
				}
			case unreadable:
				if err == nil {
					t.Fatalf("%q が黙って通った", c.field)
				}
				if strings.Contains(err.Error(), "下回っている") {
					t.Errorf("%q が床の違反として報告された: %v", c.field, err)
				}
			case below:
				if err == nil {
					t.Fatalf("%q が黙って通った", c.field)
				}
				if !strings.Contains(err.Error(), "下回っている") {
					t.Errorf("%q が床の違反として報告されていない: %v", c.field, err)
				}
			}
		})
	}
}

func TestAsPercentAtLeastShouldRefuseAFloorNobodyWrote(t *testing.T) {
	refused := panictest.Recovered(func() { validate.AsPercentAtLeast(0, 0) })
	if refused == nil {
		t.Fatal("分母が 0 の床が黙って作られた。その床は何も拒まない")
	}
}

func TestAsYenOnlyShouldRefuseEveryOtherAmount(t *testing.T) {
	const (
		readable   = ""
		unreadable = "読めない"
		other      = "別の額"
	)

	const because = "終わる年の欄が無いので、0 でない額は払いつづけられてしまう"

	cases := map[string]struct {
		only  int64
		field string
		fault string
	}{
		"許された額ちょうど":     {0, "0", readable},
		"桁区切りで書かれた同じ額":  {1000, "1,000", readable},
		"1 円だけ多い（境界値）":  {0, "1", other},
		"1 円だけ少ない（境界値）": {1000, "999", other},
		"符号だけ違う":        {0, "-1", other},
		"額として読めない":      {0, "あいうえお", unreadable},
		"空欄":            {0, "", unreadable},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := validate.AsYenOnly(money.Yen(c.only), because)(c.field)

			switch c.fault {
			case readable:
				if err != nil {
					t.Errorf("%q が拒まれた: %v", c.field, err)
				}
			case unreadable:
				if err == nil {
					t.Fatalf("%q が黙って通った", c.field)
				}
				if strings.Contains(err.Error(), "でなければならない") {
					t.Errorf("%q が別の額として報告された: %v", c.field, err)
				}
			case other:
				if err == nil {
					t.Fatalf("%q が黙って通った", c.field)
				}
				if !strings.Contains(err.Error(), "でなければならない") {
					t.Errorf("%q が別の額として報告されていない: %v", c.field, err)
				}
				if !strings.Contains(err.Error(), because) {
					t.Errorf("%q の指摘が理由を運んでいない: %v", c.field, err)
				}
			}
		})
	}
}
