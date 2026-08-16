package compare_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/compare"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestActualVsPlanShouldReportTheGapPerYearAndBucket(t *testing.T) {
	held := record(t,
		[]string{"2029", "150", "250", "350", ""},
		[]string{"2030", "160", "260", "360", ""},
	)
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2029, Record: held, Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
	}

	got, err := compare.ActualVsPlan(subjects)

	if err != nil {
		t.Fatalf("ActualVsPlan: %v", err)
	}
	assertTable(t, got, &tsv.Table{
		Header: []tsv.ColumnName{"西暦", "プロジェクト", "区分", "計画", "実績", "乖離", "起点以前", "一部未記録"},
		Rows: [][]string{
			{"2030", "base", "資産合計", "1000", "780", "-220", "", ""},
		},
	})
}

func TestActualVsPlanShouldCompareEachBucketAgainstItsOwnHolding(t *testing.T) {
	held := record(t, []string{"2030", "150", "250", "350", ""})
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2029, Record: held, Tables: map[plan.TableName]*tsv.Table{
			"assets": {
				Header: []tsv.ColumnName{"西暦", "貯蓄", "金融資産", "年金資産", "資産合計"},
				Rows:   [][]string{{"2030", "100", "200", "300", "600"}},
			},
		}},
	}

	got, err := compare.ActualVsPlan(subjects)

	if err != nil {
		t.Fatalf("ActualVsPlan: %v", err)
	}
	assertTable(t, got, &tsv.Table{
		Header: []tsv.ColumnName{"西暦", "プロジェクト", "区分", "計画", "実績", "乖離", "起点以前", "一部未記録"},
		Rows: [][]string{
			{"2030", "base", "貯蓄", "100", "150", "50", "", ""},
			{"2030", "base", "金融資産", "200", "250", "50", "", ""},
			{"2030", "base", "年金資産", "300", "350", "50", "", ""},
			{"2030", "base", "資産合計", "600", "750", "150", "", ""},
		},
	})
}

func TestActualVsPlanShouldSayWhenTheRecordIsALowerBound(t *testing.T) {
	held := record(t, []string{"2030", "160", "260", "360", "はい"})
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2029, Record: held, Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
	}

	got, err := compare.ActualVsPlan(subjects)

	if err != nil {
		t.Fatalf("ActualVsPlan: %v", err)
	}
	if len(got.Rows) != 1 {
		t.Fatalf("%d row(s), want 1: %v", len(got.Rows), got.Rows)
	}
	if got.Rows[0][7] != "はい" {
		t.Errorf("一部未記録 = %q, want %q", got.Rows[0][7], "はい")
	}
}

func TestActualVsPlanShouldMarkTheYearsThatAreTheRecordItself(t *testing.T) {
	held := record(t,
		[]string{"2030", "150", "250", "350", ""},
		[]string{"2031", "160", "260", "360", ""},
	)
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2030, Record: held, Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
	}

	got, err := compare.ActualVsPlan(subjects)

	if err != nil {
		t.Fatalf("ActualVsPlan: %v", err)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("%d row(s), want 2: %v", len(got.Rows), got.Rows)
	}
	if got.Rows[0][0] != "2030" || got.Rows[0][6] != "はい" {
		t.Errorf("2030 の行 = %v, 起点以前 に はい が要る", got.Rows[0])
	}
	if got.Rows[1][0] != "2031" || got.Rows[1][6] != "" {
		t.Errorf("2031 の行 = %v, 起点以前 は空欄のはず", got.Rows[1])
	}
}

func TestActualVsPlanShouldKeepAGapBeforeTheOriginWhenTheRecordsDiffer(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2030, Record: record(t,
			[]string{"2030", "150", "250", "350", ""},
		), Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
		{Name: "rich", StartsAfter: 2030, Record: record(t,
			[]string{"2030", "1150", "250", "350", ""},
		), Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"2000", "2030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
	}

	got, err := compare.ActualVsPlan(subjects)

	if err != nil {
		t.Fatalf("ActualVsPlan: %v", err)
	}
	found := false
	for _, row := range got.Rows {
		if row[0] == "2030" && row[1] == "rich" {
			found = true
			if row[5] != "-1250" {
				t.Errorf("乖離 = %q, want %q（実績 750 − 計画 2000）", row[5], "-1250")
			}
			if row[6] != "はい" {
				t.Errorf("起点以前 = %q, want %q", row[6], "はい")
			}
		}
	}
	if !found {
		t.Errorf("起点以前の食い違いが 1 行も出ていない: %v", got.Rows)
	}
}

func TestActualVsPlanShouldBeEmptyWhenNothingOverlaps(t *testing.T) {
	held := record(t, []string{"2029", "150", "250", "350", ""})
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2029, Record: held, Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
	}

	got, err := compare.ActualVsPlan(subjects)

	if err != nil {
		t.Fatalf("ActualVsPlan: %v", err)
	}
	if len(got.Rows) != 0 {
		t.Errorf("%d row(s), want none: %v", len(got.Rows), got.Rows)
	}
	if len(got.Header) == 0 {
		t.Error("行が無くても見出しは要る")
	}
}

func TestActualVsPlanShouldRefuseATableWithNoYearColumn(t *testing.T) {
	held := record(t, []string{"2030", "160", "260", "360", ""})

	tables := twoYears(
		[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
		[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})
	tables["assets"] = &tsv.Table{
		Header: []tsv.ColumnName{"年度", "手が届く額"},
		Rows:   [][]string{{"2030", "1000"}},
	}
	subjects := []compare.Subject{{Name: "base", StartsAfter: 2029, Record: held, Tables: tables}}

	_, err := compare.ActualVsPlan(subjects)

	if err == nil {
		t.Fatal("西暦 の無い表が黙って受け入れられた")
	}
	if !strings.Contains(err.Error(), "西暦") {
		t.Errorf("エラーが 西暦 に触れていない: %v", err)
	}
}
