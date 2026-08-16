package plan

import (
	"reflect"
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func TestWithShouldCarryEveryFieldOfTheInput(t *testing.T) {
	in, err := Load(Sources{Root: "..", ProjectPath: "../projects/classic.tsv"})
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}
	table, err := in.Table(tsv.Slot("investment_return"))
	if err != nil {
		t.Fatalf("plan.Input.Table: %v", err)
	}

	next := in.With(tsv.Slot("investment_return"), table)

	before, after := reflect.ValueOf(*in), reflect.ValueOf(*next)
	for i := 0; i < before.NumField(); i++ {
		name := before.Type().Field(i).Name
		if before.Field(i).IsZero() {
			t.Errorf("%s が元から空である。この検査は「写されたか」を見るので、"+
				"空の欄では何も言えない", name)
			continue
		}
		if after.Field(i).IsZero() {
			t.Errorf("%s が With のあとで空になっている。**掃引と計画が別の計画を"+
				"組むことになる**ので、欄をすべて写すこと", name)
		}
	}
}
