package plan_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

func assetsOf(rows ...relation.Row[table.AssetsRow]) *plan.Plan {
	return &plan.Plan{Assets: relation.New(rows)}
}

func held(year date.Year, total, shortfall money.Yen) relation.Row[table.AssetsRow] {
	return relation.Row[table.AssetsRow]{
		Year:  year,
		Value: table.AssetsRow{Total: total, Shortfall: shortfall},
	}
}

func TestOutcomeShouldSayWhenThePlanRunsOutAndWhatIsLeft(t *testing.T) {
	for _, test := range []struct {
		name  string
		built *plan.Plan
		want  plan.Outcome
	}{
		{
			name: "尽きる",
			built: assetsOf(
				held(2030, 1000, 0),
				held(2031, 400, 0),
				held(2032, 0, 250),
				held(2033, 0, 300),
			),
			want: plan.Outcome{ShortFrom: 2032, Shortfall: 550, LastYear: 2033, Final: 0},
		},
		{
			name: "持つ",
			built: assetsOf(
				held(2030, 1000, 0),
				held(2031, 1100, 0),
			),
			want: plan.Outcome{ShortFrom: 0, Shortfall: 0, LastYear: 2031, Final: 1100},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.built.Outcome()

			if err != nil {
				t.Fatalf("Outcome: %v", err)
			}
			if got != test.want {
				t.Errorf("outcome = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestOutcomeShouldRefuseAPlanWithNoRows(t *testing.T) {
	if _, err := assetsOf().Outcome(); err == nil {
		t.Error("Outcome accepted a plan with no rows")
	}
	if _, err := plan.OutcomeOf(&tsv.Table{Header: []tsv.ColumnName{"西暦", "資産合計", "不足"}}); err == nil {
		t.Error("OutcomeOf accepted a table with no rows")
	}
}

func TestOutcomeShouldRefuseANegativeShortfall(t *testing.T) {
	built := assetsOf(held(2030, 1000, -5))

	if _, err := built.Outcome(); err == nil {
		t.Error("Outcome accepted a negative 不足")
	}
	written := &tsv.Table{
		Header: []tsv.ColumnName{"西暦", "資産合計", "不足"},
		Rows:   [][]string{{"2030", "1000", "-5"}},
	}
	if _, err := plan.OutcomeOf(written); err == nil {
		t.Error("OutcomeOf accepted a negative 不足")
	}
}

func TestOutcomeOfShouldNotDependOnTheOrderTheRowsWereWrittenIn(t *testing.T) {
	sorted := &tsv.Table{
		Header: []tsv.ColumnName{"西暦", "資産合計", "不足"},
		Rows: [][]string{
			{"2030", "1000", "0"},
			{"2032", "0", "250"},
			{"2033", "0", "300"},
		},
	}
	shuffled := &tsv.Table{
		Header: sorted.Header,
		Rows: [][]string{
			{"2033", "0", "300"},
			{"2030", "1000", "0"},
			{"2032", "0", "250"},
		},
	}

	was, err := plan.OutcomeOf(sorted)
	if err != nil {
		t.Fatalf("OutcomeOf: %v", err)
	}
	is, err := plan.OutcomeOf(shuffled)
	if err != nil {
		t.Fatalf("OutcomeOf: %v", err)
	}

	want := plan.Outcome{ShortFrom: 2032, Shortfall: 550, LastYear: 2033, Final: 0}
	if was != want {
		t.Errorf("順に並んだ表から = %+v, want %+v", was, want)
	}
	if is != want {
		t.Errorf("順が乱れた表から = %+v, want %+v", is, want)
	}
}

func TestOutcomeOfShouldReadAmountsTheWayEveryOtherReaderDoes(t *testing.T) {
	got, err := plan.OutcomeOf(&tsv.Table{
		Header: []tsv.ColumnName{"西暦", "資産合計", "不足"},
		Rows:   [][]string{{"2030", " 1,000 ", "0"}},
	})

	if err != nil {
		t.Fatalf("OutcomeOf: %v", err)
	}
	if got.Final != 1000 {
		t.Errorf("最終年の資産合計 = %d, want 1000", got.Final)
	}
}
