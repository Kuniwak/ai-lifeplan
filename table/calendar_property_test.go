package table_test

import (
	"slices"
	"testing"

	"pgregory.net/rapid"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/table"
)

func TestCalendarPropertiesShouldHoldForAnyHousehold(t *testing.T) {
	stages := make([]table.Stage, 0, len(schooling))
	for _, band := range schooling {
		stages = append(stages, band.Stage)
	}

	rapid.Check(t, func(t *rapid.T) {
		from := date.Year(rapid.IntRange(1900, 2100).Draw(t, "from"))
		to := from + date.Year(rapid.IntRange(0, 120).Draw(t, "span"))
		born := date.Year(rapid.IntRange(1900, 2100).Draw(t, "born"))

		built, err := table.Calendar(table.CalendarInput{
			From:      from,
			To:        to,
			People:    []table.Person{{Name: "子", BornOn: date.Date{Year: born, Month: 1, Day: 1}, Relation: table.Child}},
			Schooling: schooling,
		})
		if err != nil {
			t.Fatalf("table.Calendar: %v", err)
		}

		if got, want := built.Len(), int(to-from)+1; got != want {
			t.Fatalf("Len() = %d, want %d", got, want)
		}

		previousAge, previousStage := 0, -1
		for i, row := range built.Rows() {
			age, ok := row.Value.AgeOf("子")
			if !ok {
				t.Fatalf("%d: 子 is missing", row.Year)
			}

			if i > 0 && age != previousAge+1 {
				t.Errorf("%d: age %d does not follow %d", row.Year, age, previousAge)
			}
			if want := int(row.Year - born); age != want {
				t.Errorf("%d: age = %d, want %d", row.Year, age, want)
			}
			previousAge = age

			stage, _ := row.Value.StageOf("子")
			at := slices.Index(stages, stage)
			if stage == table.Unborn {
				if age >= 0 {
					t.Errorf("%d: a %d year old is %q", row.Year, age, table.Unborn)
				}
				continue
			}
			if at < 0 {
				t.Fatalf("%d: %q is not a stage of the table", row.Year, stage)
			}
			if at < previousStage {
				t.Errorf("%d: the stage went back to %q", row.Year, stage)
			}
			previousStage = at
		}
	})
}
