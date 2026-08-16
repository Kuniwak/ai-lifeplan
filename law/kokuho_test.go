package law_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

const setagaya = "世田谷区"

func kokuhoTable(t *testing.T) law.KokuhoTable {
	t.Helper()

	table, err := law.LoadKokuhoTable(os.DirFS("../"+law.LawDirectory), setagaya)
	if err != nil {
		t.Fatalf("law.LoadKokuhoTable: %v", err)
	}

	return table.WithGrowth(law.NoCostGrowth())
}

func TestTheKokuhoTableShouldCarryEveryPartTheMunicipalityCharges(t *testing.T) {
	got := kokuhoTable(t).Parts()

	for _, want := range []law.KokuhoPartName{
		law.KokuhoMedical, law.KokuhoElderlySupport, law.KokuhoNursingCare, law.KokuhoChildSupport,
	} {
		found := false
		for _, name := range got {
			found = found || name == want
		}
		if !found {
			t.Errorf("no %q in %v", want, got)
		}
	}
}

func TestTheKokuhoPremiumShouldChargeEachPartAndAddThemUp(t *testing.T) {
	got, err := kokuhoTable(t).Premium(law.KokuhoHousehold{Members: []law.KokuhoMember{{Months: date.WholeYear}}}, 2026)
	if err != nil {
		t.Fatalf("Premium: %v", err)
	}

	want := money.Yen(47_600 + 17_600 + 1_873).Truncate(100)
	if got != want {
		t.Errorf("premium = %d, want %d", got, want)
	}
}

func TestTheKokuhoPremiumShouldChargeNursingCareOnlyForThoseBetweenFortyAndSixtyFour(t *testing.T) {
	table := kokuhoTable(t)

	without, err := table.Premium(law.KokuhoHousehold{Members: []law.KokuhoMember{{Months: date.WholeYear}, {Months: date.WholeYear}}}, 2026)
	if err != nil {
		t.Fatalf("Premium: %v", err)
	}
	with, err := table.Premium(law.KokuhoHousehold{Members: []law.KokuhoMember{{Months: date.WholeYear, NursingCareMonths: date.WholeYear}, {Months: date.WholeYear}}}, 2026)
	if err != nil {
		t.Fatalf("Premium: %v", err)
	}

	if with <= without {
		t.Errorf("a household with somebody insured for 介護 pays %d, no more than the %d of one without", with, without)
	}
}

func TestEachPartOfTheKokuhoPremiumShouldBeCappedOnItsOwn(t *testing.T) {
	got, err := kokuhoTable(t).Premium(law.KokuhoHousehold{Members: []law.KokuhoMember{
		{Base: 100_000_000, Months: date.WholeYear, NursingCareMonths: date.WholeYear},
		{Months: date.WholeYear, NursingCareMonths: date.WholeYear},
	}}, 2026)
	if err != nil {
		t.Fatalf("Premium: %v", err)
	}

	if want := money.Yen(670_000 + 260_000 + 170_000 + 30_000); got != want {
		t.Errorf("premium = %d, want the sum of the ceilings %d", got, want)
	}
}

func TestTheKokuhoPerHouseholdChargeShouldCoverEveryMonthAnybodyIsInTheScheme(t *testing.T) {
	table := kokuhoTable(t)

	turnsSixtyFive := date.MonthsOfYearIn(2058,
		date.Date{Year: 2000, Month: 1, Day: 1}, date.Date{Year: 2058, Month: 2, Day: 28})
	turnsForty := date.MonthsOfYearIn(2058,
		date.Date{Year: 2058, Month: 3, Day: 1}, date.Date{Year: 2090, Month: 12, Day: 31})

	split := law.KokuhoHousehold{Members: []law.KokuhoMember{
		{Months: date.WholeYear, NursingCareMonths: turnsSixtyFive},
		{Months: date.WholeYear, NursingCareMonths: turnsForty},
	}}
	whole := law.KokuhoHousehold{Members: []law.KokuhoMember{
		{Months: date.WholeYear, NursingCareMonths: date.WholeYear},
		{Months: date.WholeYear, NursingCareMonths: date.WholeYear},
	}}

	got, err := table.Premium(split, 2026)
	if err != nil {
		t.Fatalf("Premium: %v", err)
	}
	full, err := table.Premium(whole, 2026)
	if err != nil {
		t.Fatalf("Premium: %v", err)
	}

	if want := full - 17_800; got != want {
		t.Errorf("窓が割れた世帯の保険税 %d、%d のはず（差は均等割だけ。平等割はどの自治体も課していない）", got, want)
	}
}

func TestTheKokuhoCapShouldBeSetAgainstTheWholeYear(t *testing.T) {
	table := kokuhoTable(t)
	const year = 2026

	whole := law.KokuhoHousehold{Members: []law.KokuhoMember{
		{Base: 30_000_000, Months: date.WholeYear},
	}}
	half := law.KokuhoHousehold{Members: []law.KokuhoMember{
		{Base: 30_000_000, Months: date.Months(1<<6 - 1)},
	}}

	full, err := table.Premium(whole, year)
	if err != nil {
		t.Fatalf("Premium: %v", err)
	}
	part, err := table.Premium(half, year)
	if err != nil {
		t.Fatalf("Premium: %v", err)
	}

	if part >= full {
		t.Errorf("6 か月 %d が 12 か月 %d を下回らない。月割の後に限度額を当てている", part, full)
	}
	if part*2 < full-1_000 || part*2 > full+1_000 {
		t.Errorf("6 か月 %d の 2 倍が 12 か月 %d から千円以上離れている", part, full)
	}
}

func TestParseKokuhoTableShouldRefuseAPartItDoesNotKnow(t *testing.T) {
	read := law.MustLoadRegionalTable(t, os.DirFS("../"+law.LawDirectory), "世田谷区", law.KokuhoTableName)
	at, ok := read.ColumnIndex(law.KokuhoPartColumn)
	if !ok {
		t.Fatalf("no %s column", law.KokuhoPartColumn)
	}
	read.Rows[0][at] = "保健事業分"

	_, err := law.ParseKokuhoTable(read)

	if err == nil {
		t.Fatal("知らない区分が通ってしまった")
	}
	if !strings.Contains(err.Error(), "保健事業分") {
		t.Errorf("どの区分が拒まれたのか分からない: %v", err)
	}
}
