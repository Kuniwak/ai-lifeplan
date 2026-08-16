package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/wording"
)

type PersonName string

type Person struct {
	Name PersonName

	BornOn date.Date

	Relation Relation
}

type Stage string

const Independent Stage = "独立"

const Unborn Stage = "未誕生"

type SchoolingBand struct {
	Stage   Stage
	FromAge int
}

type CalendarInput struct {
	From, To date.Year

	People []Person

	Schooling []SchoolingBand

	Residence []ResidenceFrom

	BoughtIn *date.Year
}

type ResidenceFrom struct {
	FromYear     date.Year
	Municipality law.Municipality
}

type CalendarRow struct {
	Ages []PersonYear

	Municipality law.Municipality

	yearsOwned int

	owns bool
}

func (r CalendarRow) YearsOwned() (int, bool) {
	return r.yearsOwned, r.owns
}

type PersonYear struct {
	Name PersonName
	Age  int

	BornOn date.Date

	Stage Stage

	Relation Relation
}

func (p PersonYear) IsChild() bool { return p.Relation == Child }

func FirstYearAnybodyReaches(calendar relation.Table[CalendarRow], age int) (date.Year, bool) {
	for _, year := range calendar.Years() {
		row, ok := calendar.At(year)
		if !ok {
			continue
		}
		for _, p := range row.Ages {
			if p.Age >= age {
				return year, true
			}
		}
	}
	return 0, false
}

func (r CalendarRow) InHousehold(name PersonName) bool {
	for _, p := range r.Ages {
		if p.Name != name {
			continue
		}
		if !p.IsChild() {
			return true
		}
		return p.Stage != Unborn && p.Stage != Independent
	}
	return false
}

func (r CalendarRow) StageOf(name PersonName) (Stage, bool) {
	for _, p := range r.Ages {
		if p.Name == name {
			return p.Stage, p.IsChild()
		}
	}
	return "", false
}

func (r CalendarRow) AgeOf(name PersonName) (int, bool) {
	for _, p := range r.Ages {
		if p.Name == name {
			return p.Age, true
		}
	}
	return 0, false
}

func (r CalendarRow) BornOnOf(name PersonName) (date.Date, bool) {
	for _, p := range r.Ages {
		if p.Name == name {
			return p.BornOn, p.BornOn != date.Date{}
		}
	}
	return date.Date{}, false
}

func BornOnIn(calendar relation.Table[CalendarRow], name PersonName) (date.Date, error) {
	rows := calendar.Rows()
	if len(rows) == 0 {
		return date.Date{}, fmt.Errorf("table.BornOnIn: 暦が空なので、%q が世帯にいるかどうかも言えない", name)
	}
	born, ok := rows[0].Value.BornOnOf(name)
	if !ok {
		return date.Date{}, fmt.Errorf("table.BornOnIn: %q は世帯にいない", name)
	}
	return born, nil
}

func Calendar(in CalendarInput) (relation.Table[CalendarRow], error) {
	var empty relation.Table[CalendarRow]

	if in.From > in.To {
		return empty, fmt.Errorf("table.Calendar: the span runs backwards: from %d to %d", in.From, in.To)
	}
	if len(in.People) == 0 {
		return empty, fmt.Errorf("table.Calendar: nobody is in the household, so there is no plan to make")
	}

	seen := make(map[PersonName]bool, len(in.People))
	for _, p := range in.People {
		if p.BornOn == (date.Date{}) {
			return empty, fmt.Errorf("table.Calendar: %q の生年月日が空である。年齢も続柄も就学段階もそこから出る", p.Name)
		}
		if seen[p.Name] {
			return empty, wording.DuplicateKeyError("table.Calendar", "person", wording.Name(p.Name),
				"that person's age and relation")
		}
		seen[p.Name] = true
	}

	if err := assertBandsNotUnborn(in.Schooling); err != nil {
		return empty, err
	}
	return relation.Over(relation.Span(in.From, in.To), func(y date.Year) CalendarRow {
		ages := make([]PersonYear, 0, len(in.People))
		for _, p := range in.People {
			age := int(y - p.BornOn.Year)
			var stage Stage
			if p.Relation == Child {
				stage = stageAt(in.Schooling, age)
			}
			ages = append(ages, PersonYear{Name: p.Name, Age: age, BornOn: p.BornOn, Stage: stage, Relation: p.Relation})
		}
		row := CalendarRow{Ages: ages, Municipality: municipalityAt(in.Residence, y)}
		if in.BoughtIn != nil {
			row.yearsOwned, row.owns = int(y-*in.BoughtIn), true
		}
		return row
	}), nil
}

func stageAt(bands []SchoolingBand, age int) Stage {
	stage := Unborn
	for _, band := range bands {
		if age < band.FromAge {
			break
		}
		stage = band.Stage
	}
	return stage
}

func assertBandsNotUnborn(bands []SchoolingBand) error {
	for _, band := range bands {
		if band.Stage == Unborn {
			return fmt.Errorf("table.Calendar: %q is not a stage anyone reaches by growing older; it is what an age before the birth means", Unborn)
		}
	}
	return nil
}

func municipalityAt(residence []ResidenceFrom, y date.Year) law.Municipality {
	where := law.Municipality("")
	for _, r := range residence {
		if y < r.FromYear {
			break
		}
		where = r.Municipality
	}
	return where
}
