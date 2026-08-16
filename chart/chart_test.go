package chart_test

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/chart"
)

func lines() chart.Lines {
	return chart.Lines{
		Title: "資産推移",
		Series: []chart.Series{
			{Name: "base", Points: points(2030, 100, 200, 150)},
			{Name: "settle-2050", Points: points(2030, 100, 180, 90)},
		},
	}
}

func flat(from, to int) []chart.Point {
	out := make([]chart.Point, 0, to-from+1)
	for year := from; year <= to; year++ {
		out = append(out, chart.Point{Year: year})
	}
	return out
}

func points(from int, values ...int64) []chart.Point {
	out := make([]chart.Point, len(values))
	for i, value := range values {
		out[i] = chart.Point{Year: from + i, Value: value}
	}
	return out
}

func TestSVGShouldDrawALineOverOnlyTheYearsItHas(t *testing.T) {
	drawn := chart.Lines{
		Title: "資産推移",
		Series: []chart.Series{
			{Name: "実績", Points: points(2022, 100, 110), Dashed: true},
			{Name: "base", Points: points(2024, 120, 130)},
		},
	}

	got, err := drawn.SVG()

	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	for _, want := range []string{">2022<", ">2025<"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("the chart does not say %q", want)
		}
	}
	span := seriesSpan(t, got, 0)
	if span[0] != chart.PadLeft {
		t.Errorf("実績の線が %d から始まっている（%d のはず）", span[0], chart.PadLeft)
	}
	if span[1] >= chart.PlotRight {
		t.Errorf("実績の線が %d まで伸びている（図の右端 %d より手前で終わるはず）",
			span[1], chart.PlotRight)
	}
	span = seriesSpan(t, got, 1)
	if span[0] <= chart.PadLeft {
		t.Errorf("計画の線が %d から始まっている（図の左端 %d より右のはず）", span[0], chart.PadLeft)
	}
	if span[1] != chart.PlotRight {
		t.Errorf("計画の線が %d で終わっている（%d のはず）", span[1], chart.PlotRight)
	}
}

func seriesSpan(t *testing.T, svg []byte, at int) [2]int {
	t.Helper()

	found := regexp.MustCompile(`<polyline [^>]*points="([^"]*)"`).FindAllSubmatch(svg, -1)
	if at >= len(found) {
		t.Fatalf("%d polyline(s), want at least %d", len(found), at+1)
	}

	span := [2]int{-1, -1}
	for _, pair := range strings.Fields(string(found[at][1])) {
		x, err := strconv.Atoi(strings.SplitN(pair, ",", 2)[0])
		if err != nil {
			t.Fatalf("Atoi: %v", err)
		}
		if span[0] < 0 || x < span[0] {
			span[0] = x
		}
		if x > span[1] {
			span[1] = x
		}
	}
	return span
}

func TestSVGShouldFetchNothing(t *testing.T) {
	got, err := lines().SVG()

	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	without := bytes.ReplaceAll(got, []byte(`xmlns="http://www.w3.org/2000/svg"`), nil)
	for _, forbidden := range []string{"http://", "https://", "<image", "<script", "@import", "xlink:href", "url("} {
		if bytes.Contains(without, []byte(forbidden)) {
			t.Errorf("the chart reaches outside itself: %q", forbidden)
		}
	}
}

func TestSVGShouldBeDeterministic(t *testing.T) {
	first, err := lines().SVG()
	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	second, err := lines().SVG()
	if err != nil {
		t.Fatalf("SVG: %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Error("two runs produced different bytes")
	}
}

func TestSVGShouldNameTheYearTheAssetsRunOut(t *testing.T) {
	drawn := lines()
	drawn.Marks = []chart.Mark{{Year: 2032, Label: "資産が尽きる"}}

	got, err := drawn.SVG()

	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	for _, want := range []string{"2032", "資産が尽きる"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("the chart does not say %q", want)
		}
	}
}

func TestSVGShouldNameEverySeries(t *testing.T) {
	got, err := lines().SVG()

	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	for _, want := range []string{"base", "settle-2050", "資産推移"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("the chart does not name %q", want)
		}
	}
}

func TestSVGShouldDrawActualsWithADashedLine(t *testing.T) {
	drawn := lines()
	drawn.Series[1].Dashed = true

	got, err := drawn.SVG()

	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	if n := strings.Count(string(got), "stroke-dasharray"); n != 2 {
		t.Errorf("stroke-dasharray が %d 個ある（線と凡例の 2 個のはず）", n)
	}
}

func TestSVGShouldRefuseASeriesHoldingOneYearTwice(t *testing.T) {
	drawn := lines()
	drawn.Series[0].Points = []chart.Point{{Year: 2030, Value: 100}, {Year: 2030, Value: 200}}

	_, err := drawn.SVG()

	if err == nil {
		t.Fatal("SVG accepted a series holding one year twice")
	}
	if !strings.Contains(err.Error(), "base") {
		t.Errorf("%q does not name the series", err)
	}
}

func TestSVGShouldRefuseASeriesWithNoPoints(t *testing.T) {
	drawn := lines()
	drawn.Series = append(drawn.Series, chart.Series{Name: "empty"})

	_, err := drawn.SVG()

	if err == nil {
		t.Fatal("SVG accepted a series with no points")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("%q does not name the series", err)
	}
}

func TestSVGShouldDrawASingleYear(t *testing.T) {
	drawn := chart.Lines{
		Title:  "資産推移",
		Series: []chart.Series{{Name: "base", Points: points(2030, 100)}},
	}

	got, err := drawn.SVG()

	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	if !bytes.Contains(got, []byte("2030")) {
		t.Error("the chart does not say the only year it has")
	}
}

func TestSVGShouldDrawAFlatSeries(t *testing.T) {
	drawn := chart.Lines{
		Title:  "資産推移",
		Series: []chart.Series{{Name: "base", Points: points(2030, 0, 0)}},
	}

	if _, err := drawn.SVG(); err != nil {
		t.Fatalf("SVG: %v", err)
	}
}

func yearLabels(t *testing.T, svg []byte) []int {
	t.Helper()

	var at []int
	for _, m := range regexp.MustCompile(
		`<text class="a" x="(\d+)" y="\d+" text-anchor="middle">\d{4}</text>`,
	).FindAllSubmatch(svg, -1) {
		x, err := strconv.Atoi(string(m[1]))
		if err != nil {
			t.Fatalf("Atoi: %v", err)
		}
		at = append(at, x)
	}
	return at
}

func TestSVGShouldNotCollideYearLabels(t *testing.T) {
	drawn := chart.Lines{Title: "資産推移"}
	drawn.Series = []chart.Series{{Name: "base", Points: flat(2018, 2090)}}

	got, err := drawn.SVG()
	if err != nil {
		t.Fatalf("SVG: %v", err)
	}

	at := yearLabels(t, got)
	if len(at) < 2 {
		t.Fatalf("%d year label(s), want the first and the last at least", len(at))
	}
	for i := 1; i < len(at); i++ {
		if gap := at[i] - at[i-1]; gap < chart.MinYearLabelGap {
			t.Errorf("年ラベルが %d しか離れていない（%d 以上のはず）: %v", gap, chart.MinYearLabelGap, at)
			break
		}
	}
}

func TestSVGShouldWidenItselfForALongLegend(t *testing.T) {
	short := chart.Lines{
		Title:  "資産推移",
		Series: []chart.Series{{Name: "a", Points: points(2030, 1, 2)}},
	}
	long := short
	long.Series = []chart.Series{{Name: "case-zero-growth-depleted-and-then-some", Points: points(2030, 1, 2)}}

	narrow, err := short.SVG()
	if err != nil {
		t.Fatalf("SVG: %v", err)
	}
	wide, err := long.SVG()
	if err != nil {
		t.Fatalf("SVG: %v", err)
	}

	if viewBoxWidth(t, wide) <= viewBoxWidth(t, narrow) {
		t.Errorf("長い凡例で図が広がっていない: %d と %d",
			viewBoxWidth(t, wide), viewBoxWidth(t, narrow))
	}
}

func viewBoxWidth(t *testing.T, svg []byte) int {
	t.Helper()

	m := regexp.MustCompile(`viewBox="0 0 (\d+) \d+"`).FindSubmatch(svg)
	if m == nil {
		t.Fatalf("no viewBox in %s", svg)
	}
	width, err := strconv.Atoi(string(m[1]))
	if err != nil {
		t.Fatalf("Atoi: %v", err)
	}
	return width
}

func TestSVGShouldKeepAMarkLabelInsideThePlot(t *testing.T) {
	for _, test := range []struct {
		name string
		year int
	}{
		{name: "右端", year: 2090},
		{name: "左端", year: 2018},
	} {
		t.Run(test.name, func(t *testing.T) {
			drawn := chart.Lines{Title: "資産推移"}
			drawn.Series = []chart.Series{{Name: "base", Points: flat(2018, 2090)}}
			drawn.Marks = []chart.Mark{
				{Year: test.year, Label: "case-zero-growth-depleted が尽きる"},
			}

			got, err := drawn.SVG()
			if err != nil {
				t.Fatalf("SVG: %v", err)
			}

			left, right := markLabelSpan(t, got)
			if left < chart.PadLeft {
				t.Errorf("印のラベルが左へ %d はみ出している", chart.PadLeft-left)
			}
			if right > chart.PlotRight {
				t.Errorf("印のラベルが右へ %d はみ出している", right-chart.PlotRight)
			}
		})
	}
}

func markLabelSpan(t *testing.T, svg []byte) (left, right int) {
	t.Helper()

	m := regexp.MustCompile(
		`<text class="m" x="(\d+)" y="\d+" text-anchor="(\w+)">([^<]*)</text>`,
	).FindSubmatch(svg)
	if m == nil {
		t.Fatalf("no mark label in %s", svg)
	}
	x, err := strconv.Atoi(string(m[1]))
	if err != nil {
		t.Fatalf("Atoi: %v", err)
	}

	width := chart.TextWidth(string(m[3]))
	switch anchor := string(m[2]); anchor {
	case "start":
		return x, x + width
	case "end":
		return x - width, x
	case "middle":
		return x - width/2, x + width/2
	default:
		t.Fatalf("unknown text-anchor %q", anchor)
		return 0, 0
	}
}
