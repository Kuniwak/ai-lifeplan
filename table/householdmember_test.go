package table_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func TestHouseholdMembersShouldRefuseWhatOnlyOneOfTheTwoTablesKnows(t *testing.T) {
	years := []date.Year{2060, 2061}
	row := table.IncomeRow{
		Total: 3_000_000, TotalIncome: 2_000_000,
		PensionReceived: 1_000_000, PensionIncome: 400_000, OldAgePensionBenefit: 900_000,
	}
	earned := relation.New([]relation.Row[table.IncomeRow]{
		{Year: 2060, Value: row},
		{Year: 2061, Value: row},
	})
	liable := relation.Constant(years, law.ResidentTaxLiability{PerCapita: true})

	cases := map[string]struct {
		income  map[table.PersonName]relation.Table[table.IncomeRow]
		taxed   map[table.PersonName]relation.Table[law.ResidentTaxLiability]
		refused string
	}{
		"揃っていれば通る": {
			income: map[table.PersonName]relation.Table[table.IncomeRow]{"本人": earned},
			taxed:  map[table.PersonName]relation.Table[law.ResidentTaxLiability]{"本人": liable},
		},

		"収入の表が空で、課税の有無も無い": {
			income:  map[table.PersonName]relation.Table[table.IncomeRow]{"本人": relation.New([]relation.Row[table.IncomeRow]{})},
			taxed:   map[table.PersonName]relation.Table[law.ResidentTaxLiability]{},
			refused: "本人",
		},
		"収入だけがある人": {
			income:  map[table.PersonName]relation.Table[table.IncomeRow]{"本人": earned},
			taxed:   map[table.PersonName]relation.Table[law.ResidentTaxLiability]{},
			refused: "本人",
		},
		"課税の有無だけがある人": {
			income:  map[table.PersonName]relation.Table[table.IncomeRow]{},
			taxed:   map[table.PersonName]relation.Table[law.ResidentTaxLiability]{"本人": liable},
			refused: "本人",
		},
		"年が片方に足りない": {
			income: map[table.PersonName]relation.Table[table.IncomeRow]{"本人": earned},
			taxed: map[table.PersonName]relation.Table[law.ResidentTaxLiability]{
				"本人": relation.Constant([]date.Year{2060}, law.ResidentTaxLiability{PerCapita: true}),
			},
			refused: "2061",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			members, err := table.HouseholdMembersOf(oneMemberCalendar("本人", 2060, 66, 2), c.income, c.taxed)

			if c.refused == "" {
				if err != nil {
					t.Fatalf("table.HouseholdMembersOf: %v", err)
				}
				got, ok := members["本人"].At(2060)
				if !ok {
					t.Fatal("2060 が無い")
				}
				want := table.HouseholdMemberYear{
					Receipts: 3_000_000, Income: 2_000_000,
					PensionReceipts: 1_000_000, PensionIncome: 400_000, OldAgePensionBenefit: 900_000,
					Taxed: law.ResidentTaxLiability{PerCapita: true},
				}
				if got != want {
					t.Errorf("行が %+v、%+v のはず", got, want)
				}
				return
			}
			if err == nil {
				t.Fatal("片側しか無いのに黙って通った")
			}
			if !strings.Contains(err.Error(), c.refused) {
				t.Errorf("誤りの文に %q が無い: %v", c.refused, err)
			}
		})
	}
}

func TestHouseholdMembersShouldAnswerForEverybodyTheCalendarHolds(t *testing.T) {
	years := []date.Year{2060, 2061}
	adult := relation.New([]relation.Row[table.IncomeRow]{
		{Year: 2060, Value: table.IncomeRow{Total: 3_000_000, TotalIncome: 2_000_000}},
		{Year: 2061, Value: table.IncomeRow{Total: 3_000_000, TotalIncome: 2_000_000}},
	})
	liable := relation.Constant(years, law.ResidentTaxLiability{PerCapita: true})

	earning := map[table.PersonName]relation.Table[table.IncomeRow]{"本人": adult}
	levied := map[table.PersonName]relation.Table[law.ResidentTaxLiability]{"本人": liable}

	calendarOf := func(people ...table.PersonYear) relation.Table[table.CalendarRow] {
		rows := make([]relation.Row[table.CalendarRow], 0, len(years))
		for _, y := range years {
			rows = append(rows, relation.Row[table.CalendarRow]{
				Year:  y,
				Value: table.CalendarRow{Municipality: "世田谷区", Ages: people},
			})
		}
		return relation.New(rows)
	}

	cases := map[string]struct {
		calendar relation.Table[table.CalendarRow]
		income   map[table.PersonName]relation.Table[table.IncomeRow]
		taxed    map[table.PersonName]relation.Table[law.ResidentTaxLiability]
		refused  string
	}{
		"大人に収入の表がある": {
			calendar: calendarOf(table.PersonYear{Name: "本人", Age: 66}),
			income:   earning,
			taxed:    levied,
		},
		"子は就学段階が答えになる": {
			calendar: calendarOf(
				table.PersonYear{Name: "本人", Age: 66},
				table.PersonYear{Name: "子", Age: 10, Stage: "小学校", Relation: table.Child},
			),
			income: earning,
			taxed:  levied,
		},
		"大人に収入の表が無い": {
			calendar: calendarOf(
				table.PersonYear{Name: "本人", Age: 66},
				table.PersonYear{Name: "同居人", Age: 40},
			),
			income:  earning,
			taxed:   levied,
			refused: "同居人",
		},

		"大人の収入の表が暦の一部の年しか覆っていない": {
			calendar: calendarOf(table.PersonYear{Name: "本人", Age: 66}),
			income: map[table.PersonName]relation.Table[table.IncomeRow]{
				"本人": relation.New([]relation.Row[table.IncomeRow]{{Year: 2060, Value: table.IncomeRow{Total: 1}}}),
			},
			taxed: map[table.PersonName]relation.Table[law.ResidentTaxLiability]{
				"本人": relation.Constant([]date.Year{2060}, law.ResidentTaxLiability{}),
			},
			refused: "2061",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			members, err := table.HouseholdMembersOf(c.calendar, c.income, c.taxed)

			if c.refused != "" {
				if err == nil {
					t.Fatal("収入の表が無い大人が黙って通った")
				}
				if !strings.Contains(err.Error(), c.refused) {
					t.Errorf("誤りの文に %q が無い: %v", c.refused, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("table.HouseholdMembersOf: %v", err)
			}

			for _, row := range c.calendar.Rows() {
				for _, person := range row.Value.Ages {
					got, ok := members[person.Name].At(row.Year)
					if !ok {
						t.Fatalf("%d 年の %q が引けない", row.Year, person.Name)
					}
					if person.Stage != "" && got != (table.HouseholdMemberYear{}) {
						t.Errorf("子 %q の行が %+v。何も稼いでいないはず", person.Name, got)
					}
				}
			}
		})
	}
}
