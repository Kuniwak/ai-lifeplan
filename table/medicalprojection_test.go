package table_test

import (
	"fmt"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
	"github.com/Kuniwak/lifeplan/tsv"
)

func theAgeCurve(t *testing.T) table.AgeCurve {
	t.Helper()

	read, err := tsv.ReadFile("../data/environment/medical-cost-by-age.tsv")
	if err != nil {
		t.Fatalf("tsv.ReadFile: %v", err)
	}
	curve, err := table.AgeCurveFrom(map[tsv.Slot]*tsv.Table{input.MedicalCostByAgeSlot: read})
	if err != nil {
		t.Fatalf("table.AgeCurveFrom: %v", err)
	}
	return curve
}

func TestTheAgeCurveRowsTheProjectionNeverReadsShouldStillBePinned(t *testing.T) {
	curve := theAgeCurve(t)

	for _, c := range []struct {
		age                   int
		cost, outOfPocket     money.Yen
		realisedSharePerMille int64
	}{
		{0, 282_000, 46_000, 163},
		{5, 162_000, 36_000, 222},
		{10, 136_000, 32_000, 235},
		{70, 630_000, 73_000, 115},
		{75, 781_000, 71_000, 90},
		{80, 937_000, 80_000, 85},
		{85, 1_087_000, 87_000, 80},
		{90, 1_202_000, 90_000, 74},
		{95, 1_278_000, 90_000, 70},
		{100, 1_242_000, 82_000, 66},
	} {
		t.Run(fmt.Sprint(c.age), func(t *testing.T) {
			if got := curve.CostAt(c.age); got != c.cost {
				t.Errorf("%d 歳の 1人当たり医療費 %d、PDF は %d", c.age, got, c.cost)
			}
			got := curve.OutOfPocketAt(c.age)
			if got != c.outOfPocket {
				t.Errorf("%d 歳の 1人当たり自己負担額 %d、PDF は %d", c.age, got, c.outOfPocket)
			}
			if share := int64(got) * 1000 / int64(c.cost); share != c.realisedSharePerMille {
				t.Errorf("%d 歳の実現負担率 %d‰、%d‰ のはず（README.md がこの数を根拠に引いている）",
					c.age, share, c.realisedSharePerMille)
			}
		})
	}
}

func TestPaidAtShouldReturnThePublishedFigureWhereTheShareIsNotInDoubt(t *testing.T) {
	curve := theAgeCurve(t)

	for _, age := range []int{10, 30, 45, 65, 69} {
		t.Run(fmt.Sprint(age), func(t *testing.T) {
			if got, want := curve.PaidAt(age, money.NewRate(3, 10)), curve.OutOfPocketAt(age); got != want {
				t.Errorf("%d 歳で 3 割を当てた自己負担 %d、公表の %d のはず", age, got, want)
			}
		})
	}

	if got, want := curve.PaidAt(80, money.NewRate(2, 10)), curve.OutOfPocketAt(80); got <= want {
		t.Errorf("80 歳で 2 割を当てた自己負担 %d が公表値 %d 以下である。"+
			"全国平均をそのまま使っていることになる", got, want)
	}
}

func TestMedicalProjectionShouldKeepTheRecordAndForecastTheRest(t *testing.T) {
	curve := theAgeCurve(t)

	const from date.Year = 2025
	years := make([]date.Year, 0, 6)
	for y := date.Year(2023); y <= 2028; y++ {
		years = append(years, y)
	}

	calendar := make([]relation.Row[table.CalendarRow], 0, len(years))
	recorded := make([]relation.Row[money.Yen], 0, len(years))
	for i, y := range years {
		calendar = append(calendar, relation.Row[table.CalendarRow]{
			Year: y,
			Value: table.CalendarRow{Ages: []table.PersonYear{
				{Name: "本人", Age: 40 + i, BornOn: date.Date{Year: y - 40 - date.Year(i), Month: 6, Day: 15}},
			}},
		})
		recorded = append(recorded, relation.Row[money.Yen]{Year: y, Value: money.Yen(100_000 + 1_000*i)})
	}

	built, err := table.MedicalProjection(table.MedicalProjectionInput{
		Calendar:   relation.New(calendar),
		Recorded:   relation.New(recorded),
		RecordedTo: from,
		Curve:      curve,
		Copay:      table.FlatCopay(money.NewRate(3, 10)),
		Projected:  []table.PersonName{"本人"},
	})
	if err != nil {
		t.Fatalf("table.MedicalProjection: %v", err)
	}

	for _, y := range years[:3] {
		want, _ := relation.New(recorded).At(y)
		got, ok := built.At(y)
		if !ok {
			t.Fatalf("%d がありません", y)
		}
		if got != want {
			t.Errorf("%d 年は記録の年なので %d のはず、%d になっている", y, want, got)
		}
	}

	base, _ := built.At(from)
	last, _ := built.At(years[len(years)-1])
	if last <= base {
		t.Errorf("見込みの最後の年が %d、起点の %d 以下である。年齢で伸びていない", last, base)
	}
	if last%table.MedicalProjectionRounding != 0 {
		t.Errorf("見込み %d が千円単位に丸められていない", last)
	}
}

func TestMedicalProjectionShouldRefuseWhatItCannotAnswer(t *testing.T) {
	curve := theAgeCurve(t)
	calendar := relation.New([]relation.Row[table.CalendarRow]{
		{Year: 2025, Value: table.CalendarRow{Ages: []table.PersonYear{{Name: "本人", Age: 40}}}},
		{Year: 2026, Value: table.CalendarRow{Ages: []table.PersonYear{{Name: "本人", Age: 41}}}},
	})
	recorded := relation.New([]relation.Row[money.Yen]{{Year: 2025, Value: 100_000}})

	for _, c := range []struct {
		name table.PersonName
		in   table.MedicalProjectionInput
	}{
		{
			name: "窓口負担割合が渡されていない",
			in: table.MedicalProjectionInput{
				Calendar: calendar, Recorded: recorded, RecordedTo: 2025,
				Curve: curve, Projected: []table.PersonName{"本人"},
			},
		},
		{
			name: "記録の最後としている年に記録が無い",
			in: table.MedicalProjectionInput{
				Calendar: calendar, Recorded: recorded, RecordedTo: 2024,
				Curve: curve, Copay: table.FlatCopay(money.NewRate(3, 10)), Projected: []table.PersonName{"本人"},
			},
		},
		{
			name: "見込む相手が暦にいない",
			in: table.MedicalProjectionInput{
				Calendar: calendar, Recorded: recorded, RecordedTo: 2025,
				Curve: curve, Copay: table.FlatCopay(money.NewRate(3, 10)), Projected: []table.PersonName{"だれか"},
			},
		},
		{
			name: "誰も見込まないので割る相手が 0 になる",
			in: table.MedicalProjectionInput{
				Calendar: calendar, Recorded: recorded, RecordedTo: 2025,
				Curve: curve, Copay: table.FlatCopay(money.NewRate(3, 10)),
			},
		},
	} {
		t.Run(string(c.name), func(t *testing.T) {
			if _, err := table.MedicalProjection(c.in); err == nil {
				t.Error("黙って通った")
			}
		})
	}
}
