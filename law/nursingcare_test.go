package law

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/panictest"
)

func loadNursingCare(t *testing.T) NursingCarePremiumTable {
	t.Helper()

	return loadedNursingCare(t, "世田谷区").WithGrowth(NoCostGrowth())
}

func TestTheNursingCareStagesShouldBeChargedExactlyAsTheOrdinanceWritesThem(t *testing.T) {
	table := loadNursingCare(t)

	for _, c := range []struct {
		subject NursingCarePremiumSubject
		written money.Yen
	}{
		{NursingCarePremiumSubject{PensionReceipts: 826_500}, 21_478},
		{NursingCarePremiumSubject{PensionReceipts: 826_501}, 36_550},
		{NursingCarePremiumSubject{PensionReceipts: 1_200_001}, 48_984},
		{NursingCarePremiumSubject{HouseholdTaxed: true, PensionReceipts: 826_501}, 75_360},
		{NursingCarePremiumSubject{Taxed: true, TotalIncome: 3_200_000}, 120_576},
		{NursingCarePremiumSubject{Taxed: true, TotalIncome: 50_000_000}, 369_264},
	} {
		if got := table.Charge(c.subject, 2060); got.Premium != c.written {
			t.Errorf("第%d段階の保険料 %d、条例の書いた %d のはず", got.Stage, got.Premium, c.written)
		}
	}

	if got := table.Charge(NursingCarePremiumSubject{PensionReceipts: 826_500}, 2060); got.Premium%KokuhoPremiumUnit == 0 {
		t.Errorf("第%d段階が %d 円の倍数になっている。世田谷区の額は割合を掛けたままのはずで、"+
			"丸まっているなら表か Charge に丸めが入っている", got.Stage, KokuhoPremiumUnit)
	}
}

func TestNursingCareChargeShouldFollowTheStageTable(t *testing.T) {
	table := loadNursingCare(t)

	cases := []struct {
		name    string
		subject NursingCarePremiumSubject
		stage   NursingCareStage
		premium money.Yen
	}{
		{"第1段階 非課税・世帯も非課税・82.65万円以下", NursingCarePremiumSubject{PensionReceipts: 826_500}, 1, 21_478},
		{"第2段階 82.65万円超", NursingCarePremiumSubject{PensionReceipts: 826_501}, 2, 36_550},
		{"第2段階 120万円ちょうどは以下なのでまだ第2段階", NursingCarePremiumSubject{PensionReceipts: 1_200_000}, 2, 36_550},
		{"第3段階 120万円超", NursingCarePremiumSubject{PensionReceipts: 1_200_001}, 3, 48_984},

		{"第4段階 非課税・世帯に課税者・82.65万円以下", NursingCarePremiumSubject{HouseholdTaxed: true, PensionReceipts: 826_500}, 4, 64_056},
		{"第5段階 非課税・世帯に課税者・82.65万円超", NursingCarePremiumSubject{HouseholdTaxed: true, PensionReceipts: 826_501}, 5, 75_360},

		{"第6段階 課税・120万円未満", NursingCarePremiumSubject{Taxed: true, TotalIncome: 1_199_999}, 6, 86_664},
		{"第7段階 課税・120万円以上", NursingCarePremiumSubject{Taxed: true, TotalIncome: 1_200_000}, 7, 94_200},
		{"第8段階 課税・210万円以上", NursingCarePremiumSubject{Taxed: true, TotalIncome: 2_100_000}, 8, 105_504},
		{"第9段階 課税・320万円以上", NursingCarePremiumSubject{Taxed: true, TotalIncome: 3_200_000}, 9, 120_576},
		{"第10段階 課税・420万円以上", NursingCarePremiumSubject{Taxed: true, TotalIncome: 4_200_000}, 10, 143_184},
		{"第11段階 課税・520万円以上", NursingCarePremiumSubject{Taxed: true, TotalIncome: 5_200_000}, 11, 158_256},
		{"第12段階 課税・620万円以上", NursingCarePremiumSubject{Taxed: true, TotalIncome: 6_200_000}, 12, 173_328},
		{"第13段階 課税・720万円以上", NursingCarePremiumSubject{Taxed: true, TotalIncome: 7_200_000}, 13, 188_400},
		{"第14段階 課税・1,000万円以上", NursingCarePremiumSubject{Taxed: true, TotalIncome: 10_000_000}, 14, 218_544},
		{"第15段階 課税・1,500万円以上", NursingCarePremiumSubject{Taxed: true, TotalIncome: 15_000_000}, 15, 256_224},
		{"第16段階 課税・2,500万円以上", NursingCarePremiumSubject{Taxed: true, TotalIncome: 25_000_000}, 16, 293_904},
		{"第17段階 課税・3,500万円以上", NursingCarePremiumSubject{Taxed: true, TotalIncome: 35_000_000}, 17, 331_584},
		{"第18段階 課税・5,000万円以上", NursingCarePremiumSubject{Taxed: true, TotalIncome: 50_000_000}, 18, 369_264},

		{"課税なら世帯に課税者がいてもいなくても同じ", NursingCarePremiumSubject{Taxed: true, HouseholdTaxed: true, TotalIncome: 2_100_000}, 8, 105_504},

		{
			name: "非課税の判定所得は 課税年金収入額 ＋（合計所得金額 − 年金に係る所得）",
			subject: NursingCarePremiumSubject{
				HouseholdTaxed: true, PensionReceipts: 1_066_000, TotalIncome: 500_000, PensionIncome: 500_000,
			},
			stage: 5, premium: 75_360,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := table.Charge(c.subject, 2060)

			if got.Stage != c.stage {
				t.Errorf("%v、%v のはず", got.Stage, c.stage)
			}
			if got.Premium != c.premium {
				t.Errorf("介護保険料 %v、%v のはず", got.Premium, c.premium)
			}
		})
	}
}

func TestNursingCarePremiumShouldGrowWithTheCareAssumption(t *testing.T) {
	table := loadNursingCare(t)
	subject := NursingCarePremiumSubject{Taxed: true, TotalIncome: 2_477_000}

	flat := table.Charge(subject, 2060).Premium
	if flat != 105_504 {
		t.Fatalf("据え置きの保険料が %v である。この検査が空回りしている", flat)
	}

	one := GrowingSteadilyBy(2018, 2090, money.NewRate(1, 100))
	careOnly := CostGrowth{Medical: NoCostGrowthCurve(), Care: one, CarePremium: one}
	medicalOnly := CostGrowth{Medical: one, Care: NoCostGrowthCurve(), CarePremium: NoCostGrowthCurve()}

	if got := table.WithGrowth(careOnly).Charge(subject, 2060).Premium; got <= flat {
		t.Errorf("介護費上昇率を与えても %v のまま（据え置き %v）", got, flat)
	}
	if got := table.WithGrowth(medicalOnly).Charge(subject, 2060).Premium; got != flat {
		t.Errorf("医療費上昇率を与えただけで %v に動いた（据え置き %v）", got, flat)
	}
}

func TestTheNursingCarePremiumShouldRefuseAYearBeforeItsPeriod(t *testing.T) {
	loaded := loadNursingCare(t)
	period, ok := loaded.LastWrittenYear()
	if !ok {
		t.Fatal("期が読めていない")
	}

	cases := map[string]struct {
		Year        date.Year
		WantRefused bool
	}{
		"期の最初の年は答える（境界値）":   {Year: period},
		"その 1 年前は拒む（境界値）":   {Year: period - 1, WantRefused: true},
		"はるか前の年も拒む":         {Year: 1900, WantRefused: true},
		"期より後の年は答える（伸ばされる）": {Year: period + 30},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var got money.Yen

			refused := panictest.Recovered(func() {
				got = loaded.Charge(NursingCarePremiumSubject{}, c.Year).Premium
			})

			if !c.WantRefused {
				if refused != nil {
					t.Fatalf("%d 年が拒まれた: %v", c.Year, refused)
				}
				if got <= 0 {
					t.Errorf("%d 年の保険料が %d である", c.Year, got)
				}
				return
			}
			if refused == nil {
				t.Fatalf("期より前の %d 年が黙って %d と答えられた", c.Year, got)
			}
			message, _ := refused.(string)
			for _, want := range []string{NursingCarePremiumTableName, "2024"} {
				if !strings.Contains(message, want) {
					t.Errorf("panic のメッセージに %q が無い: %s", want, message)
				}
			}
		})
	}
}

func TestNursingCareCategoryMonths(t *testing.T) {
	husband := date.Date{Year: 1990, Month: 6, Day: 15}
	wife := date.Date{Year: 1992, Month: 9, Day: 20}

	for name, tc := range map[string]struct {
		born          date.Date
		year          date.Year
		second, first int
	}{
		"夫 40 歳になる年は 6 月から":      {born: husband, year: 2030, second: 7, first: 0},
		"夫 その前年は無い":              {born: husband, year: 2029, second: 0, first: 0},
		"夫 40 代は通年 第2号":          {born: husband, year: 2040, second: 12, first: 0},
		"夫 65 歳になる年は 5 か月と 7 か月": {born: husband, year: 2055, second: 5, first: 7},
		"夫 その翌年は通年 第1号":          {born: husband, year: 2056, second: 0, first: 12},

		"妻 40 歳になる年は 9 月から":      {born: wife, year: 2032, second: 4, first: 0},
		"妻 65 歳になる年は 8 か月と 4 か月": {born: wife, year: 2057, second: 8, first: 4},
	} {
		t.Run(name, func(t *testing.T) {

			second := NursingCareSecondCategoryMonthsIn(tc.year, tc.born).Count()
			first := NursingCareFirstCategoryMonthsIn(tc.year, tc.born).Count()

			if second != tc.second || first != tc.first {
				t.Errorf("%d 年: 第2号 %d か月・第1号 %d か月（%d・%d のはず）",
					tc.year, second, first, tc.second, tc.first)
			}
			if second+first > 12 {
				t.Errorf("%d 年: 第2号 %d + 第1号 %d = %d か月で 12 を超える", tc.year, second, first, second+first)
			}
		})
	}
}

func TestTheAmountAStageIsReadOffShouldDependOnWhoIsTaxed(t *testing.T) {
	pensioner := NursingCarePremiumSubject{
		PensionReceipts: 1_066_000, TotalIncome: 500_000, PensionIncome: 500_000,
	}

	cases := []struct {
		name    string
		subject NursingCarePremiumSubject
		want    money.Yen
		group   NursingCareGroup
	}{
		{
			name:    "非課税なら 課税年金収入額 ＋（合計所得金額 − 年金に係る所得）",
			subject: pensioner,
			want:    1_066_000,
			group:   NursingCareExemptHousehold,
		},
		{
			name:    "世帯に課税者がいても、判定に使う額は変わらない",
			subject: func() NursingCarePremiumSubject { s := pensioner; s.HouseholdTaxed = true; return s }(),
			want:    1_066_000,
			group:   NursingCareExemptInTaxedHousehold,
		},
		{
			name:    "課税なら 合計所得金額 だけ。年金はそこに雑所得として入っている",
			subject: func() NursingCarePremiumSubject { s := pensioner; s.Taxed = true; return s }(),
			want:    500_000,
			group:   NursingCareTaxedPerson,
		},
		{
			name: "課税されていれば世帯は問われない。第6段階以降の要件に世帯が出てこない",
			subject: func() NursingCarePremiumSubject {
				s := pensioner
				s.Taxed, s.HouseholdTaxed = true, true
				return s
			}(),
			want:  500_000,
			group: NursingCareTaxedPerson,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			income, group := c.subject.AssessedIncome(), c.subject.Group()

			if income != c.want {
				t.Errorf("判定所得 %d、%d のはず", income, c.want)
			}
			if group != c.group {
				t.Errorf("群 %v、%v のはず", group, c.group)
			}
		})
	}
}
