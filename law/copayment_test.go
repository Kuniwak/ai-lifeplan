package law_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

func TestMedicalCopaymentShareShouldFollowTheStatutes(t *testing.T) {
	pensionerCouple := law.CopaymentIncome{
		HighestTaxableIncome:  781_000,
		PensionAndOtherIncome: 4_736_000,
		Revenue:               4_736_000,
	}

	for _, c := range []struct {
		name   string
		age    int
		income law.CopaymentIncome
		want   money.Rate
	}{
		{"就学前は 2割", 0, law.CopaymentIncome{}, money.NewRate(2, 10)},
		{"就学前の上端も 2割", 5, law.CopaymentIncome{}, money.NewRate(2, 10)},
		{"就学後は 3割", 6, law.CopaymentIncome{}, money.NewRate(3, 10)},
		{"働いているあいだも 3割", 36, law.CopaymentIncome{}, money.NewRate(3, 10)},
		{"70 歳の前日まで 3割", 69, law.CopaymentIncome{}, money.NewRate(3, 10)},

		{"70 歳で 2割に下がる", 70, pensionerCouple, money.NewRate(2, 10)},
		{"74 歳も 2割", 74, pensionerCouple, money.NewRate(2, 10)},
		{"75 歳で後期高齢者医療に移っても 2割のまま", 75, pensionerCouple, money.NewRate(2, 10)},
		{"計画の最終年も 2割", 104, pensionerCouple, money.NewRate(2, 10)},

		{
			name:   "課税所得も収入も現役並みなら 3割",
			age:    70,
			income: law.CopaymentIncome{HighestTaxableIncome: 1_450_000, Revenue: 5_200_000},
			want:   money.NewRate(3, 10),
		},
		{
			name:   "課税所得は 145万でも収入が 520万に届かなければ 3割にならない",
			age:    70,
			income: law.CopaymentIncome{HighestTaxableIncome: 1_450_000, Revenue: 5_199_999},
			want:   money.NewRate(2, 10),
		},
		{
			name:   "国保に他の被保険者がいなければ収入の敷居は 383万",
			age:    70,
			income: law.CopaymentIncome{HighestTaxableIncome: 1_450_000, Revenue: 3_830_000, AloneInNationalHealth: true},
			want:   money.NewRate(3, 10),
		},
		{
			name:   "75歳以上で課税所得が 28万に届かなければ 1割",
			age:    75,
			income: law.CopaymentIncome{HighestTaxableIncome: 279_999, PensionAndOtherIncome: 4_736_000},
			want:   money.NewRate(1, 10),
		},
		{
			name:   "75歳以上で年金収入等が 320万に届かなければ 1割",
			age:    75,
			income: law.CopaymentIncome{HighestTaxableIncome: 680_000, PensionAndOtherIncome: 3_199_999},
			want:   money.NewRate(1, 10),
		},
		{
			name:   "後期高齢者医療に他の被保険者がいなければ年金収入等の敷居は 200万",
			age:    75,
			income: law.CopaymentIncome{HighestTaxableIncome: 680_000, PensionAndOtherIncome: 2_000_000, AloneInLateElderly: true},
			want:   money.NewRate(2, 10),
		},
		{
			name:   "70〜74 歳は所得が低くても 1割にならない",
			age:    74,
			income: law.CopaymentIncome{},
			want:   money.NewRate(2, 10),
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := law.MedicalCopaymentShare(c.age, c.income); got.Cmp(c.want) != 0 {
				t.Errorf("%d 歳の窓口負担 %s、%s のはず", c.age, got, c.want)
			}
		})
	}
}

func TestCopaymentShareInMonth(t *testing.T) {
	husband := date.Date{Year: 1990, Month: 6, Day: 15}
	wife := date.Date{Year: 1992, Month: 9, Day: 20}
	income := law.CopaymentIncome{}

	for name, tc := range map[string]struct {
		born        date.Date
		year, month int
		want        int
	}{
		"夫 2060 年 1 月はまだ 3 割": {born: husband, year: 2060, month: 1, want: 3},
		"夫 2060 年 6 月もまだ 3 割": {born: husband, year: 2060, month: 6, want: 3},
		"夫 2060 年 7 月から 2 割":  {born: husband, year: 2060, month: 7, want: 2},
		"夫 2061 年は通年 2 割":     {born: husband, year: 2061, month: 1, want: 2},
		"妻 2062 年 9 月はまだ 3 割": {born: wife, year: 2062, month: 9, want: 3},
		"妻 2062 年 10 月から 2 割": {born: wife, year: 2062, month: 10, want: 2},
	} {
		t.Run(name, func(t *testing.T) {

			got := law.CopaymentShareInMonth(date.Year(tc.year), tc.month, tc.born, income)

			if want := money.NewRate(int64(tc.want), 10); got.Cmp(want) != 0 {
				t.Errorf("%d 年 %d 月 = %v, want %v", tc.year, tc.month, got, want)
			}
		})
	}
}

func TestTheTwoAloneFlagsShouldCountDifferentPeople(t *testing.T) {
	const taxable = money.Yen(1_450_000)

	both := law.CopaymentIncome{
		HighestTaxableIncome:  taxable,
		Revenue:               3_830_000,
		PensionAndOtherIncome: 2_000_000,
		AloneInNationalHealth: true,
		AloneInLateElderly:    true,
	}
	onlyLateElderly := both
	onlyLateElderly.AloneInNationalHealth = false
	onlyNationalHealth := both
	onlyNationalHealth.AloneInLateElderly = false

	for name, c := range map[string]struct {
		age    int
		income law.CopaymentIncome
		want   money.Rate
		why    string
	}{
		"どちらもひとり・70〜74 は国保の 383万 で判定": {
			72, both, money.NewRate(3, 10), "",
		},
		"どちらもひとり・75 以上は後期の 383万 で判定": {
			76, both, money.NewRate(3, 10), "",
		},
		"後期だけひとり・70〜74 には効かない": {
			72, onlyLateElderly, money.NewRate(2, 10),
			"収入 383万 は世帯の敷居 520万 に届かない",
		},
		"後期だけひとり・75 以上には効く": {
			76, onlyLateElderly, money.NewRate(3, 10), "",
		},
		"国保だけひとり・75 以上には効かない": {
			76, onlyNationalHealth, money.NewRate(1, 10),
			"収入 383万 は 520万 に、年金収入等 200万 は 320万 に届かない",
		},
		"国保だけひとり・70〜74 には効く": {
			72, onlyNationalHealth, money.NewRate(3, 10), "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := law.MedicalCopaymentShare(c.age, c.income); got.Cmp(c.want) != 0 {
				t.Errorf("%d 歳が %s。%s のはず（%s）", c.age, got.Percent(), c.want.Percent(), c.why)
			}
		})
	}
}
