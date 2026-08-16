package compare

import (
	"fmt"
	"strconv"

	"github.com/Kuniwak/lifeplan/chart"
	"github.com/Kuniwak/lifeplan/date"
)

const ChartFile = "assets.svg"

func AssetsChart(subjects []Subject) (chart.Lines, error) {
	drawn := chart.Lines{Title: "資産合計の推移"}

	for _, subject := range subjects {
		points, err := subject.points(finalMetric)
		if err != nil {
			return chart.Lines{}, err
		}
		drawn.Series = append(drawn.Series, chart.Series{Name: subject.Name, Points: points})

		came, err := subject.outcome()
		if err != nil {
			return chart.Lines{}, err
		}
		if came.ShortFrom == 0 {
			continue
		}
		drawn.Marks = append(drawn.Marks, chart.Mark{
			Year:  int(came.ShortFrom),
			Label: subject.Name + " が尽きる",
		})
	}

	if held := recordSeries(subjects); held != nil {
		drawn.Series = append(drawn.Series, *held)
	}
	return drawn, nil
}

func recordSeries(subjects []Subject) *chart.Series {
	held := RecordOf(subjects)
	years := held.Years()
	if len(years) == 0 {
		return nil
	}

	points := make([]chart.Point, 0, len(years))
	for _, year := range years {
		balance, ok := held.At(year)
		if !ok {
			continue
		}
		points = append(points, chart.Point{Year: int(year), Value: int64(balance.Total())})
	}
	return &chart.Series{Name: RecordSeries, Points: points, Dashed: true}
}

func (s Subject) points(metric Metric) ([]chart.Point, error) {
	years, err := s.years(metric.Table)
	if err != nil {
		return nil, err
	}
	values, err := s.column(metric)
	if err != nil {
		return nil, err
	}

	out := make([]chart.Point, len(values))
	for i, value := range values {
		year, err := date.ParseYear(years[i])
		if err != nil {
			return nil, fmt.Errorf("compare: %s: %s: %w", s.Name, metric.Table, err)
		}
		amount, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"compare: %s: %s: %q is not an amount", s.Name, metric.Column, value)
		}
		out[i] = chart.Point{Year: int(year), Value: amount}
	}
	return out, nil
}
