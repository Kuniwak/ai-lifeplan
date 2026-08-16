package table

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/tsv"
)

type Relation string

const (
	Child  Relation = "子"
	Spouse Relation = "配偶者"
	Self   Relation = "本人"
)

func Relations() []Relation { return []Relation{Child, Spouse, Self} }

func CalendarInputFrom(tables map[tsv.Slot]*tsv.Table) (CalendarInput, error) {
	var in CalendarInput

	from, to, err := input.PlanSpan(tables[input.PlanSlot])
	if err != nil {
		return in, err
	}
	in.From, in.To = from, to

	if in.People, err = readHousehold(tables[input.HouseholdSlot]); err != nil {
		return in, err
	}
	if in.Schooling, err = readSchooling(tables[input.SchoolingSlot]); err != nil {
		return in, err
	}
	if in.Residence, err = readResidence(tables[input.ResidenceSlot]); err != nil {
		return in, err
	}
	if in.BoughtIn, err = readBoughtIn(tables[input.HousingSlot]); err != nil {
		return in, err
	}

	return in, nil
}

func readHousehold(table *tsv.Table) ([]Person, error) {
	r, err := tsv.NewReader(table, input.HouseholdSlot, input.PersonColumn, input.RelationColumn, input.BornOnColumn)
	if err != nil {
		return nil, err
	}

	people := make([]Person, 0, r.Rows())
	for row := range r.Rows() {
		bornOn, err := date.Parse(r.Field(row, input.BornOnColumn))
		if err != nil {
			return nil, r.Errorf(row, input.BornOnColumn, "%v", err)
		}
		people = append(people, Person{
			Name:     PersonName(r.Field(row, input.PersonColumn)),
			BornOn:   bornOn,
			Relation: Relation(r.Field(row, input.RelationColumn)),
		})
	}
	return people, nil
}

func readSchooling(table *tsv.Table) ([]SchoolingBand, error) {
	r, err := tsv.NewReader(table, input.SchoolingSlot, input.StageColumn, input.StageFromAgeColumn)
	if err != nil {
		return nil, err
	}

	bands := make([]SchoolingBand, 0, r.Rows())
	for row := range r.Rows() {
		fromAge, err := r.Count(row, input.StageFromAgeColumn)
		if err != nil {
			return nil, err
		}
		bands = append(bands, SchoolingBand{Stage: Stage(r.Field(row, input.StageColumn)), FromAge: fromAge})
	}
	return bands, nil
}

func readResidence(table *tsv.Table) ([]ResidenceFrom, error) {
	r, err := tsv.NewReader(table, input.ResidenceSlot, input.YearColumn, input.MunicipalityColumn)
	if err != nil {
		return nil, err
	}

	residence := make([]ResidenceFrom, 0, r.Rows())
	for row := range r.Rows() {
		fromYear, err := r.Year(row, input.YearColumn)
		if err != nil {
			return nil, err
		}
		residence = append(residence, ResidenceFrom{FromYear: fromYear, Municipality: law.Municipality(r.Field(row, input.MunicipalityColumn))})
	}
	return residence, nil
}

func readBoughtIn(table *tsv.Table) (*date.Year, error) {
	r, err := tsv.NewReader(table, input.HousingSlot, input.BoughtInColumn)
	if err != nil {
		return nil, err
	}

	switch r.Rows() {
	case 0:
		return nil, nil
	case 1:
		bought, err := r.Year(0, input.BoughtInColumn)
		if err != nil {
			return nil, err
		}
		return &bought, nil
	default:
		return nil, fmt.Errorf(
			"table.CalendarInputFrom: %s has %d purchases, and the years counted from a purchase would then be ambiguous",
			input.HousingSlot, r.Rows())
	}
}
