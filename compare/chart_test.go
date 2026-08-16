package compare_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/chart"
	"github.com/Kuniwak/lifeplan/compare"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestAssetsChartShouldMarkTheYearEachProjectRunsOut(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2029, Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
		{Name: "depleted", StartsAfter: 2029, Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "900"}, [2]string{"20", "-790"},
			[2]string{"1000", "0"}, [2]string{"0", "60"}, [2]string{"9", "9"})},
	}

	got, err := compare.AssetsChart(subjects)

	if err != nil {
		t.Fatalf("AssetsChart: %v", err)
	}
	if len(got.Series) != 2 {
		t.Fatalf("%d series, want 2", len(got.Series))
	}
	for i, want := range []string{"base", "depleted"} {
		if got.Series[i].Name != want {
			t.Errorf("series %d is %q, want %q", i, got.Series[i].Name, want)
		}
	}
	want := []chart.Point{{Year: 2030, Value: 1000}, {Year: 2031, Value: 0}}
	if !slices.Equal(got.Series[1].Points, want) {
		t.Errorf("depleted の線 = %v, want %v", got.Series[1].Points, want)
	}

	if len(got.Marks) != 1 {
		t.Fatalf("%d mark(s), want 1: %v", len(got.Marks), got.Marks)
	}
	if got.Marks[0].Year != 2031 {
		t.Errorf("印の年 = %d, want 2031", got.Marks[0].Year)
	}
	if !strings.Contains(got.Marks[0].Label, "depleted") {
		t.Errorf("印 %q がどのプロジェクトのものか言っていない", got.Marks[0].Label)
	}
}

func TestAssetsChartShouldOverlayTheRecordAsADashedLine(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2029, Record: record(t,
			[]string{"2028", "100", "200", "300", ""},
			[]string{"2029", "150", "250", "350", ""},
		), Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
	}

	got, err := compare.AssetsChart(subjects)

	if err != nil {
		t.Fatalf("AssetsChart: %v", err)
	}
	if len(got.Series) != 2 {
		t.Fatalf("%d series, want 2（計画 1 本と実績 1 本）", len(got.Series))
	}
	actual := got.Series[len(got.Series)-1]
	if actual.Name != compare.RecordSeries {
		t.Errorf("実績の系列名 = %q, want %q", actual.Name, compare.RecordSeries)
	}
	if !actual.Dashed {
		t.Error("実績が実線で引かれている")
	}
	want := []chart.Point{{Year: 2028, Value: 600}, {Year: 2029, Value: 750}}
	if !slices.Equal(actual.Points, want) {
		t.Errorf("実績の線 = %v, want %v", actual.Points, want)
	}
}

func TestAssetsChartShouldDrawTheRecordOnceHoweverManyProjects(t *testing.T) {
	held := record(t, []string{"2029", "150", "250", "350", ""})
	tables := func() map[plan.TableName]*tsv.Table {
		return twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})
	}
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2029, Record: held, Tables: tables()},
		{Name: "other", StartsAfter: 2029, Record: held, Tables: tables()},
	}

	got, err := compare.AssetsChart(subjects)

	if err != nil {
		t.Fatalf("AssetsChart: %v", err)
	}
	drawn := 0
	for _, series := range got.Series {
		if series.Name == compare.RecordSeries {
			drawn++
		}
	}
	if drawn != 1 {
		t.Errorf("実績の系列が %d 本ある（1 本のはず）", drawn)
	}
}

func TestComparedChartShouldBeDeterministic(t *testing.T) {
	subjects := []compare.Subject{
		{Name: "base", StartsAfter: 2029, Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "80"}, [2]string{"20", "30"},
			[2]string{"1000", "1030"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
		{Name: "other", StartsAfter: 2029, Tables: twoYears(
			[2]string{"100", "110"}, [2]string{"80", "90"}, [2]string{"20", "20"},
			[2]string{"1000", "1020"}, [2]string{"0", "0"}, [2]string{"9", "9"})},
	}

	first, err := compare.Of(subjects)
	if err != nil {
		t.Fatalf("Of: %v", err)
	}
	second, err := compare.Of(subjects)
	if err != nil {
		t.Fatalf("Of: %v", err)
	}

	if string(first.Chart) != string(second.Chart) {
		t.Error("two runs drew different bytes")
	}
	if len(first.Chart) == 0 {
		t.Error("no chart was drawn")
	}
}
