package money

import (
	"testing"

	"github.com/Kuniwak/lifeplan/panictest"
)

func TestYenTruncateShouldPanicOnNonPositiveUnit(t *testing.T) {
	testCases := map[string]Yen{
		"zero unit":     0,
		"negative unit": -1000,
	}

	for name, unit := range testCases {
		t.Run(name, func(t *testing.T) {
			refused := panictest.Recovered(func() { Yen(1000).Truncate(unit) })

			if refused == nil {
				t.Errorf("Truncate(%d): want panic, got none", unit)
			}
		})
	}
}
