package breakeven

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/plan"
)

type Outcome struct {
	plan.Outcome

	Setting Setting

	Deferred money.Yen

	Recorded money.Yen
}

func Sweep(in *plan.Input, dial Dial, settings []Setting) (Swept, error) {
	startsAfter, err := in.StartsAfter()
	if err != nil {
		return Swept{}, err
	}
	if dial.From != startsAfter+1 {
		return Swept{}, fmt.Errorf(
			"breakeven.Sweep: %s を %d 年から回そうとしている。実績は %d 年まであるので、回せるのは %d 年からである",
			dial.Name, dial.From, startsAfter, startsAfter+1)
	}

	written, err := in.Table(dial.Slot)
	if err != nil {
		return Swept{}, fmt.Errorf("breakeven.Sweep: %s: %w", dial.Name, err)
	}

	now, err := dial.Written(written)
	if err != nil {
		return Swept{}, err
	}
	painted, err := dial.PaintedOver(written, now)
	if err != nil {
		return Swept{}, err
	}

	outcomes := make([]Outcome, 0, len(settings))
	var leftAlone []date.Year
	for _, setting := range settings {
		turned, err := dial.turn(written, setting)
		if err != nil {
			return Swept{}, err
		}

		if turned.Wrote == 0 {
			return Swept{}, fmt.Errorf(
				"breakeven.Sweep: %s（%s）は %d 年以降のどの年も回せない。"+
					"%s がどの年も 0 以下で、%s を書き込める年が無い。"+
					"その列を書き直すか、就労延長（PostponeDialOf）のように別の回し方の要る問いに"+
					"なっていないか確かめること",
				dial.Name, dial.Column, dial.From, turned.Stopped, dial.Column)
		}
		leftAlone = turned.LeftAlone

		built, err := in.With(dial.Slot, turned.Table).Build()
		if err != nil {
			return Swept{}, fmt.Errorf("breakeven.Sweep: %s = %s: %w", dial.Name, setting, err)
		}
		outcome, err := outcomeOf(setting, dial, built)
		if err != nil {
			return Swept{}, fmt.Errorf("breakeven.Sweep: %s = %s: %w", dial.Name, setting, err)
		}
		outcomes = append(outcomes, outcome)
	}
	return Swept{Dial: dial, Now: now, Outcomes: outcomes, Painted: painted, LeftAlone: leftAlone}, nil
}

func outcomeOf(setting Setting, dial Dial, built *plan.Plan) (Outcome, error) {
	came, err := built.Outcome()
	if err != nil {
		return Outcome{}, err
	}

	held, _ := built.Assets.At(dial.From - 1)

	last, _ := built.LastHeld()

	return Outcome{
		Outcome:  came,
		Setting:  setting,
		Recorded: held.Total,
		Deferred: last.DeferredTax(),
	}, nil
}

type Turn struct {
	Before, After Outcome
}

func Turns(outcomes []Outcome) []Turn {
	var turns []Turn
	for i := 1; i < len(outcomes); i++ {
		if outcomes[i-1].Fails() != outcomes[i].Fails() {
			turns = append(turns, Turn{Before: outcomes[i-1], After: outcomes[i]})
		}
	}
	return turns
}
