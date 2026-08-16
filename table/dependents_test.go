package table_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestADependantEarningOverTheCeilingShouldNotBeOne(t *testing.T) {
	const ceiling money.Yen = 580_000
	const year date.Year = 2025

	for _, c := range []struct {
		name          string
		spouseIncome  money.Yen
		childIncome   money.Yen
		wantSpouse    bool
		wantRelatives []table.PersonName
	}{
		{
			name: "収入の無い二人はどちらも数える", spouseIncome: 0, childIncome: 0,
			wantSpouse: true, wantRelatives: []table.PersonName{"子1"},
		},
		{
			name: "ちょうど所得要件なら数える", spouseIncome: ceiling, childIncome: ceiling,
			wantSpouse: true, wantRelatives: []table.PersonName{"子1"},
		},
		{
			name: "所得要件を 1 円超える子は扶養親族ではない", spouseIncome: 0, childIncome: ceiling + 1,
			wantSpouse: true, wantRelatives: nil,
		},
		{
			name: "所得要件を 1 円超える配偶者は同一生計配偶者ではない", spouseIncome: ceiling + 1, childIncome: 0,
			wantSpouse: false, wantRelatives: []table.PersonName{"子1"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			calendar := table.CalendarRow{Ages: []table.PersonYear{
				{Name: "夫", Age: 40},
				{Name: "妻", Age: 38},
				{Name: "子1", Age: 19, Stage: "大学", Relation: table.Child},
			}}
			in := table.DependentsInput{
				Taxpayer: "夫", Spouse: "妻", ClaimsDependents: true,
				Income: map[table.PersonName]relation.Table[money.Yen]{
					"妻":  relation.Constant([]date.Year{year}, c.spouseIncome),
					"子1": relation.Constant([]date.Year{year}, c.childIncome),
				},
				SpouseIncomeCeiling: ceilingOf(t, ceiling),
			}

			got := table.DependentsOf(in, calendar, year)

			if got.SpouseSameLivelihood != c.wantSpouse {
				t.Errorf("同一生計配偶者が %v である（%v のはず）。妻の所得 %d 円",
					got.SpouseSameLivelihood, c.wantSpouse, c.spouseIncome)
			}

			var names []table.PersonName
			for _, person := range got.Relatives {
				names = append(names, person.Name)
			}
			if diff := cmp.Diff(c.wantRelatives, names); diff != "" {
				t.Errorf("扶養親族が違う (-want +got):\n%s", diff)
			}

			for _, name := range []table.PersonName{"夫", "妻", "子1"} {
				want := slices.Contains(c.wantRelatives, name)
				if got.Includes(name) != want {
					t.Errorf("Includes(%q) が %v である（%v のはず）", name, got.Includes(name), want)
				}
			}

			if want := len(c.wantRelatives) + map[bool]int{true: 1}[c.wantSpouse]; got.Count() != want {
				t.Errorf("頭数が %d である（%d のはず）", got.Count(), want)
			}
		})
	}
}

func ceilingOf(t *testing.T, amount money.Yen) law.SpouseIncomeCeilingTable {
	t.Helper()

	built, err := law.ParseSpouseIncomeCeilingTable(&tsv.Table{
		Header: []tsv.ColumnName{law.LawStartYearColumn, law.SpouseIncomeCeilingColumn, law.LawEndYearColumn},
		Rows:   [][]string{{"不明", fmt.Sprint(int64(amount)), "無期限"}},
	})
	if err != nil {
		t.Fatalf("law.ParseSpouseIncomeCeilingTable: %v", err)
	}
	return built
}

func TestAChildEarningOverTheCeilingShouldBringNoDependentDeduction(t *testing.T) {
	const ceiling money.Yen = 580_000
	const year date.Year = 2025

	for _, c := range []struct {
		name        string
		childIncome money.Yen
		want        law.Deduction
	}{
		{name: "収入の無い 19 歳は特定扶養親族", childIncome: 0,
			want: law.Deduction{IncomeTax: 630_000, Resident: 450_000}},
		{name: "所得要件を超える 19 歳は扶養親族ではない", childIncome: ceiling + 1,
			want: law.Deduction{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			built, err := table.IncomeTaxTable(oneChildIncomeTaxInput(t, year, ceiling, c.childIncome))
			if err != nil {
				t.Fatalf("table.IncomeTaxTable: %v", err)
			}

			row, ok := built.At(year)
			if !ok {
				t.Fatalf("%d の行が無い", year)
			}

			if row.Deductions.Dependents != c.want {
				t.Errorf("扶養控除が %+v である（%+v のはず）。子の所得 %d 円、所得要件 %d 円",
					row.Deductions.Dependents, c.want, c.childIncome, ceiling)
			}
		})
	}
}

func oneChildIncomeTaxInput(
	t *testing.T, year date.Year, ceiling, childIncome money.Yen,
) table.IncomeTaxInput {
	t.Helper()

	years := []date.Year{year}
	return table.IncomeTaxInput{
		Calendar: relation.Constant(years, table.CalendarRow{
			Municipality: "世田谷区",
			Ages: []table.PersonYear{
				{Name: "夫", Age: 45},
				{Name: "子1", Age: 19, Stage: "大学", Relation: table.Child},
			},
		}),
		Taxpayer:         "夫",
		ClaimsDependents: true,
		Income: map[table.PersonName]relation.Table[money.Yen]{
			"夫":  relation.Constant(years, money.Yen(5_000_000)),
			"子1": relation.Constant(years, childIncome),
		},
		SpouseIncomeCeiling: ceilingOf(t, ceiling),
	}
}
