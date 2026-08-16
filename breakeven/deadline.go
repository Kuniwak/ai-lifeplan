package breakeven

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

type Deadlined struct {
	Dial Dial
	At   Setting

	Years []Noticed

	TooLate date.Year

	HoldsAnyway bool

	Painted Painted
}

type Noticed struct {
	Year date.Year

	Holds bool

	Shortfall int64
}

func Deadline(in *plan.Input, dial Dial, at Setting, upTo date.Year) (Deadlined, error) {
	startsAfter, err := in.StartsAfter()
	if err != nil {
		return Deadlined{}, err
	}

	from := startsAfter + 1
	if upTo < from {
		return Deadlined{}, fmt.Errorf(
			"breakeven.Deadline: %d 年までを見ようとしているが、実績は %d 年まであるので、"+
				"見られるのは %d 年からである", upTo, startsAfter, from)
	}

	written, err := in.Table(dial.Slot)
	if err != nil {
		return Deadlined{}, fmt.Errorf("breakeven.Deadline: %s: %w", dial.Name, err)
	}

	if dial.Correction == Postpone {
		lastRow, err := dial.LastRowYear(written)
		if err != nil {
			return Deadlined{}, fmt.Errorf("breakeven.Deadline: %s: %w", dial.Name, err)
		}
		if upTo > lastRow {
			return Deadlined{}, fmt.Errorf(
				"breakeven.Deadline: %d 年まで見ようとしているが、%s の最後の行は %d 年である。"+
					"それより先に気付いても、もう延ばす行が無い", upTo, dial.Slot, lastRow)
		}
	}

	untouched, err := in.Build()
	if err != nil {
		return Deadlined{}, err
	}
	came, err := untouched.Outcome()
	if err != nil {
		return Deadlined{}, err
	}

	if rows := untouched.Assets.Rows(); len(rows) > 0 {
		if ends := rows[len(rows)-1].Year; upTo > ends {
			return Deadlined{}, fmt.Errorf(
				"breakeven.Deadline: %d 年まで見ようとしているが、計画は %d 年で終わる。"+
					"それより先に気付く年は無い", upTo, ends)
		}
	}

	now, err := dial.Written(written)
	if err != nil {
		return Deadlined{}, err
	}
	painted, err := dialFrom(dial, from).PaintedOver(written, now)
	if err != nil {
		return Deadlined{}, err
	}

	out := Deadlined{Dial: dial, At: at, HoldsAnyway: !came.Fails(), Painted: painted}
	for year := from; year <= upTo; year++ {
		noticed, err := noticedIn(in, dial, at, written, year)
		if err != nil {
			return Deadlined{}, err
		}
		out.Years = append(out.Years, noticed)
		if !noticed.Holds && out.TooLate == 0 {
			out.TooLate = year
		}
	}
	return out, nil
}

func dialFrom(dial Dial, year date.Year) Dial {
	dial.From = year
	return dial
}

func noticedIn(in *plan.Input, dial Dial, at Setting, written *tsv.Table, year date.Year) (Noticed, error) {
	turned, err := dialFrom(dial, year).Turn(written, at)
	if err != nil {
		return Noticed{}, err
	}

	built, err := in.With(dial.Slot, turned).Build()
	if err != nil {
		return Noticed{}, fmt.Errorf("breakeven.Deadline: %s を %d 年から %s: %w",
			dial.Name, year, at, err)
	}
	came, err := built.Outcome()
	if err != nil {
		return Noticed{}, err
	}

	return Noticed{
		Year: year, Holds: !came.Fails(), Shortfall: int64(came.Shortfall),
	}, nil
}

const (
	NoticedYearColumn tsv.ColumnName = "西暦"
	InTimeColumn      tsv.ColumnName = "間に合うか"
)

type InTime string

const (
	StillInTime InTime = "間に合う"
	NoLonger    InTime = "間に合わない"
)

func DeadlineTable(deadlined Deadlined) *tsv.Table {
	out := &tsv.Table{Header: []tsv.ColumnName{
		NoticedYearColumn, DialColumn, SettingColumn, InTimeColumn, ShortfallColumn,
	}}
	for _, year := range deadlined.Years {
		inTime := StillInTime
		if !year.Holds {
			inTime = NoLonger
		}
		out.Rows = append(out.Rows, []string{
			fmt.Sprint(year.Year), deadlined.Dial.Name, deadlined.At.Field(),
			string(inTime), fmt.Sprint(year.Shortfall),
		})
	}
	return out
}

func DeadlineSummary(deadlined Deadlined) string {
	dial, at, years := deadlined.Dial, deadlined.At, deadlined.Years

	if deadlined.HoldsAnyway {
		return fmt.Sprintf(
			"%s: この計画はもともと不足しない。%s にするかどうかで期限は生じない",
			dial.Name, at)
	}

	if len(years) == 0 {
		return fmt.Sprintf("%s: 1 年も見ていない", dial.Name)
	}
	first, last := years[0].Year, years[len(years)-1].Year

	if deadlined.TooLate == 0 {
		return fmt.Sprintf(
			"%s を %s にするなら、%d 年から %d 年までのどの年に気付いても間に合う",
			dial.Name, at, first, last)
	}

	if deadlined.TooLate == first {
		works := firstHolding(years)
		if works == 0 {
			return fmt.Sprintf(
				"%s: %s にしても、%d 年から %d 年までのどの年に気付いても不足が消えない。"+
					"**この手では足りない**", dial.Name, at, first, last)
		}
		s := fmt.Sprintf(
			"%s: %s にしても %d 年に気付いた分では不足が消えない。消えるのは %d 年以降に"+
				"気付いた場合である。**早く気付くほどよい、という向きになっていない**",
			dial.Name, at, first, works)
		if stops := firstFailingAfter(years, works); stops != 0 {
			s += fmt.Sprintf("。ただし %d 年以降に気付いた場合はまた消えない。**単調でない**", stops)
		}
		return s
	}

	s := fmt.Sprintf(
		"%s: %d 年までに気付けば、%s にすることで不足が消える。"+
			"**%d 年を過ぎると、そうしても消えない**",
		dial.Name, deadlined.TooLate-1, at, deadlined.TooLate-1)
	if again := firstHoldingAfter(years, deadlined.TooLate); again != 0 {
		s += fmt.Sprintf("。ただし %d 年に気付いた場合はまた消える。**単調でない**", again)
	}
	return s
}

func firstHolding(years []Noticed) date.Year {
	for _, year := range years {
		if year.Holds {
			return year.Year
		}
	}
	return 0
}

func firstFailingAfter(years []Noticed, after date.Year) date.Year {
	for _, year := range years {
		if year.Year > after && !year.Holds {
			return year.Year
		}
	}
	return 0
}

func firstHoldingAfter(years []Noticed, after date.Year) date.Year {
	for _, year := range years {
		if year.Year >= after && year.Holds {
			return year.Year
		}
	}
	return 0
}

func DeadlineWarning(deadlined Deadlined) string {
	return Swept{Dial: deadlined.Dial, Painted: deadlined.Painted}.Warning()
}
