package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/yeartest"
)

func TestShortTimeWorkAboveTheFloorShouldNameASpecifiedWorkplace(t *testing.T) {
	tables := tablesOfTheBaseProject(t)

	checked := 0
	for _, slot := range []tsv.Slot{input.IncomeHusbandSlot, input.IncomeWifeSlot} {
		table, ok := tables[slot]
		if !ok {
			t.Fatalf("%s の表が無い", slot)
		}

		hoursAt := yeartest.ColumnIndex(t, table, input.WeeklyHoursColumn)
		normalAt := yeartest.ColumnIndex(t, table, input.NormalWeeklyHoursColumn)
		salaryAt := yeartest.ColumnIndex(t, table, input.AnnualSalaryColumn)
		workplaceAt := yeartest.ColumnIndex(t, table, input.SpecifiedWorkplaceColumn)

		yeartest.EachYear(t, table, func(year date.Year, fields []string) {
			hours, err := parseCount(fields[hoursAt])
			if err != nil {
				t.Fatalf("%s %d: %s: %v", slot, year, input.WeeklyHoursColumn, err)
			}
			normal, err := parseCount(fields[normalAt])
			if err != nil {
				t.Fatalf("%s %d: %s: %v", slot, year, input.NormalWeeklyHoursColumn, err)
			}
			salary, err := money.ParseYen(fields[salaryAt])
			if err != nil {
				t.Fatalf("%s %d: %s: %v", slot, year, input.AnnualSalaryColumn, err)
			}
			checked++

			if hours <= 0 {
				return
			}
			if hours*4 >= normal*3 {
				return
			}
			if monthly := salary / 12; monthly < law.ShortTimeRemunerationFloor {
				return
			}

			if fields[workplaceAt] == input.SpecifiedWorkplace {
				return
			}

			t.Errorf("%s の %d 年（週 %d 時間・年 %d 円）は短時間の帯に入っているのに、"+
				"%s が %q である。**この帯では被用者保険に入るかどうかが事業所の規模で決まり、"+
				"%q は保険料を立てない側、すなわち支出を過小に見る側に外れる**——"+
				"実際に非該当なら、そう書いた上でその向きを承知しておくこと",
				slot, year, hours, salary,
				input.SpecifiedWorkplaceColumn, fields[workplaceAt], input.NotSpecifiedWorkplace)
		})
	}
	if checked == 0 {
		t.Fatal("1 行も見ていない")
	}
}

func parseCount(field string) (int, error) {
	amount, err := money.ParseYen(field)
	return int(amount), err
}
