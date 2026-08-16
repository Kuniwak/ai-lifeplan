package table_test

import (
	"slices"
	"testing"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/table"
)

func TestTheRelationVocabularyShouldAgreeWithTheConstants(t *testing.T) {
	want := make([]string, 0, len(table.Relations()))
	for _, r := range table.Relations() {
		want = append(want, string(r))
	}
	got := slices.Clone(input.RelationWords)
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("input.RelationWords = %v, want %v", got, want)
	}
}
