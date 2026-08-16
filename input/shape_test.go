package input_test

import (
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

func TestEveryRequiredSlotShouldHaveAShape(t *testing.T) {
	described := make(map[tsv.Slot]bool, len(input.Shapes()))
	for _, shape := range input.Shapes() {
		if described[shape.Slot] {
			t.Errorf("the slot %q is described twice", shape.Slot)
		}
		described[shape.Slot] = true
	}

	for _, slot := range input.RequiredSlots() {
		if !described[slot] {
			t.Errorf("the slot %q has no shape, so nothing about its table is checked", slot)
		}
	}
	for slot := range described {
		if !slices.Contains(input.RequiredSlots(), slot) {
			t.Errorf("the shape %q describes a slot the plan does not use", slot)
		}
	}
}

func TestEveryShapeShouldNameTheColumnsItsTableHas(t *testing.T) {
	tables := loadBase(t)

	for _, shape := range input.Shapes() {
		table, ok := tables[shape.Slot]
		if !ok {
			t.Errorf("the base project has no table for the slot %q", shape.Slot)
			continue
		}

		for _, column := range shape.Columns {
			if _, ok := table.ColumnIndex(column.Name); !ok {
				t.Errorf("%s: the shape names a column %q the table does not have", shape.Slot, column.Name)
			}
		}
		for _, name := range table.Header {
			described := slices.ContainsFunc(shape.Columns, func(c validate.Column) bool {
				return c.Name == name
			})
			if !described {
				t.Errorf("%s: the table has a column %q the shape does not name", shape.Slot, name)
			}
		}
	}
}

func TestAStepShapedTableShouldBeKeyedByYear(t *testing.T) {
	for _, shape := range input.Shapes() {
		switch shape.Kind {
		case input.Step, input.Events:
			if shape.YearColumn == "" {
				t.Errorf("%s: a table read by year names no year column", shape.Slot)
			}
		case input.Lookup:
			if shape.YearColumn != "" {
				t.Errorf("%s: a table not read by year names the year column %q", shape.Slot, shape.YearColumn)
			}
		default:
			t.Errorf("%s: unknown kind %v", shape.Slot, shape.Kind)
		}
	}
}

func TestThePensionShapeShouldHaveNoAmountLeft(t *testing.T) {
	var found []tsv.ColumnName
	for _, shape := range input.Shapes() {
		if shape.Slot != input.PensionSlot {
			continue
		}
		for _, c := range shape.Columns {
			found = append(found, c.Name)
		}
	}
	if len(found) == 0 {
		t.Fatal("the pension shape names no columns")
	}
	for _, gone := range []tsv.ColumnName{"基礎年金額[円/年]", "報酬比例年金額[円/年]", "加給年金額[円/年]"} {
		if slices.Contains(found, gone) {
			t.Errorf("%q がまだ形にある。3 つとも記録・投影・条文から出す", gone)
		}
	}
	for _, want := range []tsv.ColumnName{input.PersonColumn, input.PensionStartColumn, input.PensionExpectedColumn} {
		if !slices.Contains(found, want) {
			t.Errorf("%q が形から消えている", want)
		}
	}
}
