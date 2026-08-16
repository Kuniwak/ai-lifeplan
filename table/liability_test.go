package table_test

import (
	"os"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

func disabilitiesOfTheBaseProject(t *testing.T) table.Disabilities {
	t.Helper()

	built, err := table.DisabilitiesFrom(tablesOfTheBaseProject(t))
	if err != nil {
		t.Fatalf("table.DisabilitiesFrom: %v", err)
	}
	return built
}

func liabilityOfTheBaseProject(t *testing.T, taxpayer, spouse table.PersonName) relation.Table[law.ResidentTaxLiability] {
	t.Helper()
	return liabilityOf(t, taxpayer, spouse, nil)
}

func liabilityOf(
	t *testing.T, taxpayer, spouse table.PersonName, override map[table.PersonName]relation.Table[money.Yen],
) relation.Table[law.ResidentTaxLiability] {
	t.Helper()

	income := make(map[table.PersonName]relation.Table[money.Yen], 2)
	for person, slot := range map[table.PersonName]tsv.Slot{"夫": input.IncomeHusbandSlot, "妻": input.IncomeWifeSlot} {
		if replaced, ok := override[person]; ok {
			income[person] = replaced
			continue
		}
		built := incomeOfTheBaseProject(t, person, slot)
		rows := make([]relation.Row[money.Yen], 0, built.Len())
		for _, row := range built.Rows() {
			rows = append(rows, relation.Row[money.Yen]{Year: row.Year, Value: row.Value.TotalIncome})
		}
		income[person] = relation.New(rows)
	}

	spouseCeiling := law.MustLoadSpouseIncomeCeilings(t, os.DirFS("../"+law.LawDirectory))

	built, err := table.ResidentTaxLiabilityTable(table.ResidentTaxLiabilityInput{
		Calendar: calendarOfTheBaseProject(t),
		Dependents: table.DependentsInput{
			Taxpayer: taxpayer, Spouse: spouse, Income: income,
			SpouseIncomeCeiling: spouseCeiling,
			ClaimsDependents:    taxpayer == "夫",
		},
		Disabilities: disabilitiesOfTheBaseProject(t),
		Levies:       law.MustLoadResidentLevies(t, os.DirFS("../"+law.LawDirectory), municipalitiesOfTheBaseProject(t)...),
	})
	if err != nil {
		t.Fatalf("table.ResidentTaxLiabilityTable: %v", err)
	}
	return built
}

func TestEveryYearShouldHaveALiabilityRow(t *testing.T) {
	calendar := calendarOfTheBaseProject(t)

	for _, person := range []struct{ Taxpayer, Spouse table.PersonName }{{"夫", "妻"}, {"妻", "夫"}} {
		t.Run(string(person.Taxpayer), func(t *testing.T) {
			built := liabilityOfTheBaseProject(t, person.Taxpayer, person.Spouse)

			for _, year := range calendar.Years() {
				if _, ok := built.At(year); !ok {
					t.Errorf("%d: 非課税判定の行が無い。ResidentTaxTable は ok を捨てて読むので、その年は何も課されなくなる", year)
				}
			}
			if built.Len() != calendar.Len() {
				t.Errorf("行数 %d、暦は %d 年ある", built.Len(), calendar.Len())
			}
		})
	}
}

func TestTheLiabilityOfAYearShouldBeJudgedOnTheYearBefore(t *testing.T) {
	const crosses date.Year = 2050

	calendar := calendarOfTheBaseProject(t)
	rows := make([]relation.Row[money.Yen], 0, calendar.Len())
	for _, year := range calendar.Years() {
		var amount money.Yen
		if year >= crosses {
			amount = 10_000_000
		}
		rows = append(rows, relation.Row[money.Yen]{Year: year, Value: amount})
	}

	built := liabilityOf(t, "夫", "妻", map[table.PersonName]relation.Table[money.Yen]{"夫": relation.New(rows)})

	for _, c := range []struct {
		year date.Year
		want bool
	}{
		{crosses - 1, false},
		{crosses, false},
		{crosses + 1, true},
	} {
		got, ok := built.At(c.year)
		if !ok {
			t.Fatalf("%d: 行が無い", c.year)
		}
		if got.PerCapita != c.want {
			t.Errorf("課税年度%d の均等割が %v である（%v のはず。判定は %d 年の所得で行う）",
				c.year, got.PerCapita, c.want, c.year-1)
		}
	}
}

func TestThePerCapitaAmountShouldTurnOnTotalIncomeAndNotOnTaxableIncome(t *testing.T) {
	const crosses date.Year = 2050

	calendar := calendarOfTheBaseProject(t)
	rows := make([]relation.Row[money.Yen], 0, calendar.Len())
	for _, year := range calendar.Years() {
		var amount money.Yen
		if year >= crosses {
			amount = 3_000_000
		}
		rows = append(rows, relation.Row[money.Yen]{Year: year, Value: amount})
	}

	built := liabilityOf(t, "夫", "妻", map[table.PersonName]relation.Table[money.Yen]{"夫": relation.New(rows)})

	got, ok := built.At(crosses + 1)
	if !ok {
		t.Fatalf("課税年度%d の行が無い", crosses+1)
	}
	if !got.PerCapita {
		t.Errorf("合計所得 3,000,000 で均等割が課されない。限度額は合計所得で見るはずである")
	}
}

func municipalitiesOfTheBaseProject(t *testing.T) []law.Municipality {
	t.Helper()

	written := tablesOfTheBaseProject(t)[input.ResidenceSlot]
	r, err := tsv.NewReader(written, input.ResidenceSlot, input.MunicipalityColumn)
	if err != nil {
		t.Fatalf("tsv.NewReader: %v", err)
	}

	var named []law.Municipality
	for row := range r.Rows() {
		named = append(named, law.Municipality(r.Field(row, input.MunicipalityColumn)))
	}
	if len(named) == 0 {
		t.Fatalf("%s に自治体が 1 つも無い", input.ResidenceSlot)
	}
	return named
}
