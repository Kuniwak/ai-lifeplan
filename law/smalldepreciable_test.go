package law

import (
	"testing"

	"github.com/Kuniwak/lifeplan/input"
)

func TestTheSmallDepreciableCeilingShouldAgreeWithTheColumn(t *testing.T) {
	if input.SmallDepreciableYearlyLimit != SmallDepreciableYearlyLimit {
		t.Errorf("input.SmallDepreciableYearlyLimit = %d, want %d",
			input.SmallDepreciableYearlyLimit, SmallDepreciableYearlyLimit)
	}
}
