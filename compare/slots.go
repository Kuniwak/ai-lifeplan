package compare

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/tsv"
)

const (
	SlotColumn  tsv.ColumnName = "slot"
	ClassColumn tsv.ColumnName = "分類"
)

type Class string

const (
	Chosen       Class = "入力"
	Environment  Class = "環境"
	Record       Class = "実績"
	Unclassified Class = "不明"

	FromCLI Class = "コマンド引数"
)

const (
	chosenDir      = "controllable"
	environmentDir = "environment"
	recordDir      = "actuals"
)

func ClassOf(path string) Class {
	for _, element := range strings.Split(filepath.ToSlash(filepath.Clean(path)), "/") {
		switch element {
		case chosenDir:
			return Chosen
		case environmentDir:
			return Environment
		case recordDir:
			return Record
		}
	}
	return Unclassified
}

type Difference struct {
	Slot  tsv.Slot
	Class Class

	Paths []string
}

func Differences(subjects []Subject) []Difference {
	var out []Difference
	for _, slot := range slotsOf(subjects) {
		paths := make([]string, len(subjects))
		for i, subject := range subjects {
			paths[i] = subject.Paths[slot]
		}
		if overriddenIn(subjects, slot) {
			out = append(out, Difference{Slot: slot, Class: FromCLI, Paths: paths})
			continue
		}
		if allSame(paths) {
			continue
		}
		out = append(out, Difference{Slot: slot, Class: classOfAny(paths), Paths: paths})
	}
	return out
}

func overriddenIn(subjects []Subject, slot tsv.Slot) bool {
	for _, subject := range subjects {
		if subject.Overridden[slot] {
			return true
		}
	}
	return false
}

func Slots(subjects []Subject) *tsv.Table {
	header := []tsv.ColumnName{SlotColumn, ClassColumn}
	for _, subject := range subjects {
		header = append(header, tsv.ColumnName(subject.Name))
	}

	out := &tsv.Table{Header: header}
	for _, difference := range Differences(subjects) {
		row := []string{string(difference.Slot), string(difference.Class)}
		out.Rows = append(out.Rows, append(row, difference.Paths...))
	}
	return out
}

func slotsOf(subjects []Subject) []tsv.Slot {
	var all []tsv.Slot
	for _, subject := range subjects {
		for slot := range subject.Paths {
			all = append(all, slot)
		}
	}
	slices.Sort(all)
	return slices.Compact(all)
}

func classOfAny(paths []string) Class {
	found := Unclassified
	for _, path := range paths {
		if path == "" {
			continue
		}
		switch class := ClassOf(path); {
		case class == Unclassified:
			return Unclassified
		case found == Unclassified:
			found = class
		case found != class:
			return Unclassified
		}
	}
	return found
}

func allSame(values []string) bool {
	for _, value := range values[1:] {
		if value != values[0] {
			return false
		}
	}
	return true
}
