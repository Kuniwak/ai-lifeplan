package law

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

var theSchedules = map[string]struct {
	file  string
	year  date.Year
	bands int
}{
	"令和7年分以降": {"salary-income-schedule", 2025, 1175},
	"令和6年分以前": {"salary-income-schedule-until-reiwa6", 2024, 1247},
}

func TestTheSalaryScheduleShouldFollowFromTheSpeedTable(t *testing.T) {
	for name, schedule := range theSchedules {
		t.Run(name, func(t *testing.T) {
			table, err := tsv.ReadFile("../testdata/law/schedule5/" + schedule.file + ".tsv")
			if err != nil {
				t.Fatalf("tsv.ReadFile: %v", err)
			}
			fromAt, ok := table.ColumnIndex("以上[円]")
			if !ok {
				t.Fatal("no 以上[円] column")
			}
			toAt, ok := table.ColumnIndex("未満[円]")
			if !ok {
				t.Fatal("no 未満[円] column")
			}
			incomeAt, ok := table.ColumnIndex("給与所得控除後の給与等の金額[円]")
			if !ok {
				t.Fatal("no 給与所得控除後の給与等の金額[円] column")
			}

			checked := 0
			previous := money.Yen(0)
			for _, row := range table.Rows {
				from, err := money.ParseYen(row[fromAt])
				if err != nil {
					t.Fatalf("以上: %v", err)
				}
				to, err := money.ParseYen(row[toAt])
				if err != nil {
					t.Fatalf("未満: %v", err)
				}
				want, err := money.ParseYen(row[incomeAt])
				if err != nil {
					t.Fatalf("給与所得控除後: %v", err)
				}

				if previous != 0 && from != previous {
					t.Errorf("階級 %d〜%d の前が %d で終わっている。隙間がある", from, to, previous)
				}
				previous = to

				for _, salary := range []money.Yen{from, from + 1, to - 1} {
					if got := SalaryIncome(salary, schedule.year); got != want {
						t.Errorf("給与収入 %d（階級 %d〜%d）の給与所得 = %d、別表第五は %d",
							salary, from, to, got, want)
					}
				}
				checked++
			}

			if checked != schedule.bands {
				t.Errorf("別表第五の階級を %d しか読んでいない（%d のはず）", checked, schedule.bands)
			}
		})
	}
}

func TestTheScheduleShouldStateNoughtBelowItsFirstFigure(t *testing.T) {
	for name, c := range map[string]struct {
		year          date.Year
		nothing, flat money.Yen
	}{
		"令和6年分以前": {2024, 551_000, 550_000},
		"令和7年分以降": {2025, 651_000, 650_000},
	} {
		t.Run(name, func(t *testing.T) {
			if got := SalaryIncome(c.nothing-1, c.year); got != 0 {
				t.Errorf("給与収入 %d の給与所得 = %d、別表第五は 0", c.nothing-1, got)
			}
			if got, want := SalaryIncome(c.nothing, c.year), c.nothing-c.flat; got != want {
				t.Errorf("給与収入 %d の給与所得 = %d、%d のはず", c.nothing, got, want)
			}
		})
	}
}
