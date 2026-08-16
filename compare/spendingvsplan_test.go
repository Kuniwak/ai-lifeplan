package compare_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/compare"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestSpendingVsPlanShouldReportTheGapPerYear(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", Tables: map[plan.TableName]*tsv.Table{
			"outturn": {
				Header: []tsv.ColumnName{"西暦", "支出−運用損益"},
				Rows:   [][]string{{"2023", "5000000"}},
			},
			"timeline": {
				Header: []tsv.ColumnName{"西暦", "総支出"},
				Rows:   [][]string{{"2023", "3000000"}},
			},
		}},
	}

	got, err := compare.SpendingVsPlan(subjects)

	if err != nil {
		t.Fatalf("SpendingVsPlan: %v", err)
	}
	assertTable(t, got, &tsv.Table{
		Header: []tsv.ColumnName{"西暦", "プロジェクト", "計画の総支出", "支出−運用損益", "計画との差"},
		Rows: [][]string{
			{"2023", "base", "3000000", "5000000", "2000000"},
		},
	})
}

func TestSpendingVsPlanShouldOnlyReportYearsBothSidesReach(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", Tables: map[plan.TableName]*tsv.Table{
			"outturn": {
				Header: []tsv.ColumnName{"西暦", "支出−運用損益"},
				Rows:   [][]string{{"2022", "1000"}, {"2023", "2000"}},
			},
			"timeline": {
				Header: []tsv.ColumnName{"西暦", "総支出"},
				Rows:   [][]string{{"2023", "1800"}, {"2024", "1900"}},
			},
		}},
	}

	got, err := compare.SpendingVsPlan(subjects)

	if err != nil {
		t.Fatalf("SpendingVsPlan: %v", err)
	}
	assertTable(t, got, &tsv.Table{
		Header: []tsv.ColumnName{"西暦", "プロジェクト", "計画の総支出", "支出−運用損益", "計画との差"},
		Rows: [][]string{
			{"2023", "base", "1800", "2000", "200"},
		},
	})
}

func TestSpendingVsPlanShouldSkipASubjectWithNoOutturnTable(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "no-record", Tables: map[plan.TableName]*tsv.Table{
			"timeline": {
				Header: []tsv.ColumnName{"西暦", "総支出"},
				Rows:   [][]string{{"2023", "1800"}},
			},
		}},
	}

	got, err := compare.SpendingVsPlan(subjects)

	if err != nil {
		t.Fatalf("SpendingVsPlan: %v", err)
	}
	if len(got.Rows) != 0 {
		t.Errorf("%d row(s), want none: %v", len(got.Rows), got.Rows)
	}
}
