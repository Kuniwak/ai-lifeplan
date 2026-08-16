package chart

import (
	"bytes"
	"fmt"
	"html"
	"maps"
	"slices"
	"strconv"
)

type Point struct {
	Year  int
	Value int64
}

type Series struct {
	Name string

	Points []Point

	Dashed bool
}

type Mark struct {
	Year  int
	Label string
}

type Lines struct {
	Title  string
	Series []Series

	Marks []Mark
}

type axis struct {
	years []int
	at    map[int]int
}

func (l Lines) axisOf() axis {
	seen := make(map[int]struct{})
	for _, series := range l.Series {
		for _, point := range series.Points {
			seen[point.Year] = struct{}{}
		}
	}

	years := slices.Sorted(maps.Keys(seen))
	at := make(map[int]int, len(years))
	for i, year := range years {
		at[year] = i
	}
	return axis{years: years, at: at}
}

func (a axis) x(at int) int {
	if len(a.years) == 1 {
		return PadLeft + plotWidth/2
	}
	return PadLeft + plotWidth*at/(len(a.years)-1)
}

const (
	plotWidth = 688
	height    = 540

	PadLeft   = 96
	PlotRight = PadLeft + plotWidth

	padTop    = 56
	padBottom = 48

	legendDash   = 28
	legendBefore = 12
	legendAfter  = 18
)

const MinYearLabelGap = 60

var palette = []string{"#1b4d8f", "#b3541e", "#2f7a3e", "#7a2f6b", "#6b6b1e"}

func (l Lines) SVG() ([]byte, error) {
	if err := l.assertDrawable(); err != nil {
		return nil, err
	}
	a := l.axisOf()
	width := l.width()

	var b bytes.Buffer
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" role="img" aria-label="%s">`,
		width, height, esc(l.Title))
	b.WriteString("\n")
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#ffffff"/>`, width, height)
	b.WriteString("\n")
	fmt.Fprintf(&b, `<style>text{font-family:sans-serif;fill:#222222}.t{font-size:20px}.a{font-size:13px}.m{font-size:13px;fill:#a11}</style>`)
	b.WriteString("\n")
	fmt.Fprintf(&b, `<text class="t" x="%d" y="32">%s</text>`+"\n", PadLeft, esc(l.Title))

	low, high := l.span()
	l.writeAxes(&b, a, low, high)
	l.writeMarks(&b, a, low, high)
	l.writeSeries(&b, a, low, high)
	l.writeLegend(&b)

	b.WriteString("</svg>\n")
	return b.Bytes(), nil
}

func (l Lines) assertDrawable() error {
	if len(l.axisOf().years) == 0 {
		return fmt.Errorf("chart: no years to draw")
	}
	for _, series := range l.Series {
		if len(series.Points) == 0 {
			return fmt.Errorf("chart: series %q has no points to draw", series.Name)
		}
		for i := 1; i < len(series.Points); i++ {
			was, is := series.Points[i-1].Year, series.Points[i].Year
			if was == is {
				return fmt.Errorf("chart: series %q holds the year %d twice", series.Name, is)
			}
			if was > is {
				return fmt.Errorf(
					"chart: series %q has the year %d after %d, so it is not in year order",
					series.Name, is, was)
			}
		}
	}
	return nil
}

func (l Lines) span() (low, high int64) {
	for _, series := range l.Series {
		for _, point := range series.Points {
			low = min(low, point.Value)
			high = max(high, point.Value)
		}
	}
	if low == high {
		high = low + 1
	}
	return low, high
}

func (l Lines) width() int {
	longest := 0
	for _, series := range l.Series {
		longest = max(longest, TextWidth(series.Name))
	}
	return PlotRight + legendBefore + legendDash + legendBefore + longest + legendAfter
}

func TextWidth(s string) int {
	width := 0
	for _, r := range s {
		if r > 0xFF {
			width += 13
		} else {
			width += 8
		}
	}
	return width
}

func y(value, low, high int64) int {
	inner := int64(height - padTop - padBottom)
	return height - padBottom - int(inner*(value-low)/(high-low))
}

func (l Lines) writeAxes(b *bytes.Buffer, a axis, low, high int64) {
	fmt.Fprintf(b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#888888"/>`+"\n",
		PadLeft, padTop, PadLeft, height-padBottom)
	fmt.Fprintf(b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#888888"/>`+"\n",
		PadLeft, height-padBottom, PlotRight, height-padBottom)

	if low <= 0 && 0 <= high {
		at := y(0, low, high)
		fmt.Fprintf(b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#444444"/>`+"\n",
			PadLeft, at, PlotRight, at)
		fmt.Fprintf(b, `<text class="a" x="%d" y="%d" text-anchor="end">0</text>`+"\n",
			PadLeft-8, at+4)
	}
	fmt.Fprintf(b, `<text class="a" x="%d" y="%d" text-anchor="end">%s</text>`+"\n",
		PadLeft-8, padTop+4, strconv.FormatInt(high, 10))

	for _, at := range a.labelledYears() {
		fmt.Fprintf(b, `<text class="a" x="%d" y="%d" text-anchor="middle">%d</text>`+"\n",
			a.x(at), height-padBottom+20, a.years[at])
	}
}

func (l Lines) writeMarks(b *bytes.Buffer, a axis, low, high int64) {
	for _, mark := range l.Marks {
		at, drawn := a.at[mark.Year]
		if !drawn {
			continue
		}

		x := a.x(at)
		fmt.Fprintf(b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#a11" stroke-dasharray="4 3"/>`+"\n",
			x, padTop, x, height-padBottom)

		words := fmt.Sprintf("%s %d", mark.Label, mark.Year)
		anchor, at := anchorFor(x, TextWidth(words))
		fmt.Fprintf(b, `<text class="m" x="%d" y="%d" text-anchor="%s">%s</text>`+"\n",
			at, padTop-8, anchor, esc(words))
	}
}

func (a axis) labelledYears() []int {
	if len(a.years) == 1 {
		return []int{0}
	}

	last := len(a.years) - 1
	at := []int{0}
	for i, year := range a.years {
		if i == 0 || i == last || year%10 != 0 {
			continue
		}
		if a.x(i)-a.x(at[len(at)-1]) < MinYearLabelGap {
			continue
		}
		if a.x(last)-a.x(i) < MinYearLabelGap {
			continue
		}
		at = append(at, i)
	}
	return append(at, last)
}

func anchorFor(x, words int) (anchor string, at int) {
	switch {
	case x+words/2 > PlotRight:
		return "end", min(x, PlotRight)
	case x-words/2 < PadLeft:
		return "start", max(x, PadLeft)
	default:
		return "middle", x
	}
}

func (l Lines) writeSeries(b *bytes.Buffer, a axis, low, high int64) {
	for i, series := range l.Series {
		var points bytes.Buffer
		for at, point := range series.Points {
			if at > 0 {
				points.WriteString(" ")
			}
			fmt.Fprintf(&points, "%d,%d", a.x(a.at[point.Year]), y(point.Value, low, high))
		}

		dash := ""
		if series.Dashed {
			dash = ` stroke-dasharray="8 5"`
		}
		fmt.Fprintf(b, `<polyline fill="none" stroke="%s" stroke-width="2"%s points="%s"/>`+"\n",
			colour(i), dash, points.String())

		if len(series.Points) == 1 {
			only := series.Points[0]
			fmt.Fprintf(b, `<circle cx="%d" cy="%d" r="3" fill="%s"/>`+"\n",
				a.x(a.at[only.Year]), y(only.Value, low, high), colour(i))
		}
	}
}

func (l Lines) writeLegend(b *bytes.Buffer) {
	for i, series := range l.Series {
		top := padTop + i*24
		dash := ""
		if series.Dashed {
			dash = ` stroke-dasharray="8 5"`
		}
		fmt.Fprintf(b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="2"%s/>`+"\n",
			PlotRight+legendBefore, top, PlotRight+legendBefore+legendDash, top, colour(i), dash)
		fmt.Fprintf(b, `<text class="a" x="%d" y="%d">%s</text>`+"\n",
			PlotRight+legendBefore+legendDash+legendBefore, top+4, esc(series.Name))
	}
}

func colour(at int) string { return palette[at%len(palette)] }

func esc(s string) string { return html.EscapeString(s) }
