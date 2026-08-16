package table_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/table"
)

func TestCalendarShouldGiveEveryYearOfThePlanTheAgeOfEachPerson(t *testing.T) {
	built, err := table.Calendar(table.CalendarInput{
		From:   2018,
		To:     2020,
		People: []table.Person{{Name: "夫", BornOn: date.Date{Year: 1989, Month: 1, Day: 1}}, {Name: "妻", BornOn: date.Date{Year: 1993, Month: 1, Day: 1}}},
	})
	if err != nil {
		t.Fatalf("table.Calendar: %v", err)
	}

	if got, want := built.Len(), 3; got != want {
		t.Fatalf("Len() = %d, want %d", got, want)
	}

	row, ok := built.At(date.Year(2018))
	if !ok {
		t.Fatal("2018 is missing")
	}
	for name, want := range map[table.PersonName]int{"夫": 29, "妻": 25} {
		got, ok := row.AgeOf(name)
		if !ok {
			t.Errorf("%s is missing from the row", name)
			continue
		}
		if got != want {
			t.Errorf("%s: age = %d, want %d", name, got, want)
		}
	}
}

var schooling = []table.SchoolingBand{
	{Stage: "3歳未満", FromAge: 0},
	{Stage: "幼稚園", FromAge: 3},
	{Stage: "小学校", FromAge: 6},
	{Stage: "中学校", FromAge: 13},
	{Stage: "高校", FromAge: 16},
	{Stage: "大学", FromAge: 19},
	{Stage: "独立", FromAge: 23},
}

func TestCalendarShouldSayWhichStageOfSchoolingAChildIsIn(t *testing.T) {
	built, err := table.Calendar(table.CalendarInput{
		From:      2018,
		To:        2090,
		People:    []table.Person{{Name: "子1", BornOn: date.Date{Year: 2022, Month: 1, Day: 1}, Relation: table.Child}},
		Schooling: schooling,
	})
	if err != nil {
		t.Fatalf("table.Calendar: %v", err)
	}

	for year, want := range map[date.Year]table.Stage{
		2018: table.Unborn,
		2021: table.Unborn,
		2022: "3歳未満",
		2025: "幼稚園",
		2028: "小学校",
		2035: "中学校",
		2038: "高校",
		2041: "大学",
		2045: "独立",
		2090: "独立",
	} {
		row, ok := built.At(year)
		if !ok {
			t.Fatalf("%d is missing", year)
		}
		got, ok := row.StageOf("子1")
		if !ok {
			t.Errorf("%d: 子1 has no stage", year)
			continue
		}
		if got != want {
			t.Errorf("%d: stage = %q, want %q", year, got, want)
		}
	}
}

func TestCalendarShouldGiveNoStageToSomeoneWhoIsNotAChild(t *testing.T) {
	built, err := table.Calendar(table.CalendarInput{
		From:      2018,
		To:        2018,
		People:    []table.Person{{Name: "夫", BornOn: date.Date{Year: 1989, Month: 1, Day: 1}}, {Name: "子1", BornOn: date.Date{Year: 2022, Month: 1, Day: 1}}},
		Schooling: schooling,
	})
	if err != nil {
		t.Fatalf("table.Calendar: %v", err)
	}

	row, _ := built.At(date.Year(2018))
	if stage, ok := row.StageOf("夫"); ok {
		t.Errorf("夫 has the stage %q", stage)
	}
}

func TestCalendarShouldSayWhichMunicipalityTheRulesComeFrom(t *testing.T) {
	built, err := table.Calendar(table.CalendarInput{
		From:   2018,
		To:     2090,
		People: []table.Person{{Name: "夫", BornOn: date.Date{Year: 1989, Month: 1, Day: 1}}},
		Residence: []table.ResidenceFrom{
			{FromYear: 2018, Municipality: "世田谷区"},
			{FromYear: 2023, Municipality: "世田谷区"},
		},
	})
	if err != nil {
		t.Fatalf("table.Calendar: %v", err)
	}

	for year, want := range map[date.Year]law.Municipality{
		2018: "世田谷区",
		2022: "世田谷区",
		2023: "世田谷区",
		2090: "世田谷区",
	} {
		row, ok := built.At(year)
		if !ok {
			t.Fatalf("%d is missing", year)
		}
		if got := row.Municipality; got != want {
			t.Errorf("%d: municipality = %q, want %q", year, got, want)
		}
	}
}

func TestCalendarShouldCountTheYearsSinceTheHomeWasBought(t *testing.T) {
	bought := date.Year(2022)
	built, err := table.Calendar(table.CalendarInput{
		From:     2018,
		To:       2090,
		People:   []table.Person{{Name: "夫", BornOn: date.Date{Year: 1989, Month: 1, Day: 1}}},
		BoughtIn: &bought,
	})
	if err != nil {
		t.Fatalf("table.Calendar: %v", err)
	}

	for year, want := range map[date.Year]int{
		2018: -4,
		2022: 0,
		2023: 1,
		2090: 68,
	} {
		row, ok := built.At(year)
		if !ok {
			t.Fatalf("%d is missing", year)
		}
		got, ok := row.YearsOwned()
		if !ok {
			t.Errorf("%d: no count of years owned", year)
			continue
		}
		if got != want {
			t.Errorf("%d: years owned = %d, want %d", year, got, want)
		}
	}
}

func TestCalendarShouldCountNoYearsOwnedWhenNoHomeIsBought(t *testing.T) {
	built, err := table.Calendar(table.CalendarInput{
		From:   2018,
		To:     2018,
		People: []table.Person{{Name: "夫", BornOn: date.Date{Year: 1989, Month: 1, Day: 1}}},
	})
	if err != nil {
		t.Fatalf("table.Calendar: %v", err)
	}

	row, _ := built.At(date.Year(2018))
	if got, ok := row.YearsOwned(); ok {
		t.Errorf("years owned = %d, want none", got)
	}
}

func TestCalendarShouldRefuseAPersonWithNoBirthDate(t *testing.T) {
	in := table.CalendarInput{
		From: 2024, To: 2025,
		People: []table.Person{
			{Name: "夫", Relation: table.Self, BornOn: date.Date{Year: 1985, Month: 6, Day: 15}},
			{Name: "子", Relation: table.Child},
		},
	}

	_, err := table.Calendar(in)

	if err == nil {
		t.Fatal("生年月日の無い人を受け入れてしまった")
	}
	if !strings.Contains(err.Error(), "子") {
		t.Errorf("言われたのは %q だが、誰の生年月日が無いのかを言ってほしい", err)
	}
}
