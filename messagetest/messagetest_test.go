package messagetest_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/messagetest"
)

func TestMatchShouldFindAnAssignmentWhateverOrderTheExpectationsAreIn(t *testing.T) {
	for name, c := range map[string]struct {
		got  []string
		want [][]string
	}{
		"the narrow expectation is written first": {[]string{"A B", "A"}, [][]string{{"A"}, {"A", "B"}}},
		"the wide expectation is written first":   {[]string{"A B", "A"}, [][]string{{"A", "B"}, {"A"}}},
		"the messages arrive in the other order":  {[]string{"A", "A B"}, [][]string{{"A"}, {"A", "B"}}},
	} {
		t.Run(name, func(t *testing.T) {
			missing, unmatched := messagetest.Match(c.got, c.want)
			if len(missing) > 0 {
				t.Errorf("待っている message が %d 件、無いと言われた: %v", len(missing), missing)
			}
			if len(unmatched) > 0 {
				t.Errorf("誰も待っていないと言われた message が %d 件: %v", len(unmatched), unmatched)
			}
		})
	}
}

func TestMatchShouldStillCountWhatIsMissingAndWhatIsSpare(t *testing.T) {
	for name, c := range map[string]struct {
		got                        []string
		want                       [][]string
		wantMissing, wantUnmatched int
	}{
		"a check said one thing too many":              {[]string{"A", "B"}, [][]string{{"A"}}, 0, 1},
		"a check said nothing where something was due": {[]string{"A"}, [][]string{{"A"}, {"C"}}, 1, 0},
		"nothing expected and nothing said":            {nil, nil, 0, 0},
		"two messages, one expectation that fits both": {[]string{"A B", "A C"}, [][]string{{"A"}}, 0, 1},
	} {
		t.Run(name, func(t *testing.T) {
			missing, unmatched := messagetest.Match(c.got, c.want)
			if len(missing) != c.wantMissing {
				t.Errorf("missing が %d 件（%d 件のはず）: %v", len(missing), c.wantMissing, missing)
			}
			if len(unmatched) != c.wantUnmatched {
				t.Errorf("unmatched が %d 件（%d 件のはず）: %v", len(unmatched), c.wantUnmatched, unmatched)
			}
		})
	}
}
