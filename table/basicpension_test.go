package table_test

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/table"
)

const theFullBasicPension money.Yen = 847_300

func theRecordsOfTheHousehold(t *testing.T) ([]actuals.PensionRecordEntry, []actuals.Remuneration) {
	t.Helper()

	record, err := actuals.PensionRecordOf(os.DirFS(".."))
	if err != nil {
		t.Fatalf("actuals.PensionRecordOf: %v", err)
	}
	monthly, err := actuals.RemunerationRecord(os.DirFS(".."))
	if err != nil {
		t.Fatalf("actuals.RemunerationRecord: %v", err)
	}
	return record, monthly
}

func TestTheBasicPensionShouldNotExceedTheStatutesMonths(t *testing.T) {
	_, monthly := theRecordsOfTheHousehold(t)

	got, err := table.NationalPensionPaidMonths(table.NationalPensionPaidMonthsInput{
		PaidInTheRecord: law.BasicPensionFullMonths,
		RecordedThrough: date.Date{Year: 2026, Month: 8, Day: 1},
		Monthly:         monthly,
		Calendar:        calendarOfTheBaseProject(t),
		Person:          "妻",
	})
	if err != nil {
		t.Fatalf("table.NationalPensionPaidMonths: %v", err)
	}
	if got != law.BasicPensionFullMonths {
		t.Errorf("納付済月数 = %d で、上限の %d 月と違う", got, law.BasicPensionFullMonths)
	}
}
