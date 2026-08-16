package law

import (
	"testing"

	"github.com/Kuniwak/lifeplan/panictest"
)

func TestNewSpeedTableShouldRefuseATableThatCannotAnswer(t *testing.T) {
	testCases := map[string]func(){
		"行が無い": func() { NewSpeedTable() },
		"最後の行に上端がある": func() {
			NewSpeedTable(
				SpeedTableStep{Upto: 1_000_000, Add: 100_000},
				SpeedTableStep{Upto: 2_000_000, Rate: 10},
			)
		},
		"上端が昇順でない": func() {
			NewSpeedTable(
				SpeedTableStep{Upto: 2_000_000, Add: 100_000},
				SpeedTableStep{Upto: 1_000_000, Rate: 10},
				SpeedTableStep{Add: 500_000},
			)
		},
	}

	for name, build := range testCases {
		t.Run(name, func(t *testing.T) {
			refused := panictest.Recovered(build)

			if refused == nil {
				t.Error("答えられない速算表が受け付けられた")
			}
		})
	}
}

func TestTheZeroSpeedTableShouldSayItWasNotBuilt(t *testing.T) {
	var zero SpeedTable

	refused := panictest.Recovered(func() { zero.At(1_000_000) })

	if refused == nil {
		t.Error("組み立てられていない速算表が答えを返した")
	}
}
