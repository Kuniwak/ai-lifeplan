package breakevencmd

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/breakeven"
)

func TestWhatToTurnShouldSweepTheProjectsOwnDialsByDefault(t *testing.T) {
	dials, settings, err := whatToTurn(&Options{To: 800, Step: breakeven.Step}, 2026)
	if err != nil {
		t.Fatalf("whatToTurn: %v", err)
	}

	if len(dials) != len(breakeven.Dials(2026)) {
		t.Errorf("%d 本のダイヤルが返っている（%d 本のはず）", len(dials), len(breakeven.Dials(2026)))
	}
	if len(settings) != 81 {
		t.Errorf("0.00%% から 8.00%% を 0.10pt 刻みで 81 点のはずが %d 点", len(settings))
	}
	if got := settings[len(settings)-1].Field(); got != "8.00%" {
		t.Errorf("最後の設定が %q である（8.00%% のはず）", got)
	}
}

func TestWhatToTurnShouldSweepANamedDialInItsOwnUnit(t *testing.T) {
	dials, settings, err := whatToTurn(&Options{
		Dial: "living_cost:生活費[円/月]", From: 250_000, To: 500_000, Step: 10_000,
	}, 2026)
	if err != nil {
		t.Fatalf("whatToTurn: %v", err)
	}

	if len(dials) != 1 {
		t.Fatalf("名指したのに %d 本のダイヤルが返っている", len(dials))
	}
	if dials[0].Kind.Unit() != "円" {
		t.Errorf("生活費を %s のダイヤルとして回そうとしている", dials[0].Kind.Unit())
	}
	if len(settings) != 26 {
		t.Errorf("250000 から 500000 を 10000 刻みで 26 点のはずが %d 点", len(settings))
	}
	if got := settings[0].Field(); got != "250000" {
		t.Errorf("最初の設定が %q である（250000 のはず）", got)
	}
}

func TestWhatToTurnShouldSweepAPostponedDialInYears(t *testing.T) {
	dials, settings, err := whatToTurn(&Options{
		Postpone: "income_husband", To: 10, Step: 1,
	}, 2026)
	if err != nil {
		t.Fatalf("whatToTurn: %v", err)
	}

	if len(dials) != 1 {
		t.Fatalf("名指したのに %d 本のダイヤルが返っている", len(dials))
	}
	if dials[0].Kind.Unit() != "年" {
		t.Errorf("延長ダイヤルの単位が %s である（年のはず）", dials[0].Kind.Unit())
	}
	if dials[0].Correction != breakeven.Postpone {
		t.Errorf("延長ダイヤルの Correction が %v である（breakeven.Postpone のはず）", dials[0].Correction)
	}
	if len(settings) != 11 {
		t.Errorf("0 から 10 を 1 刻みで 11 点のはずが %d 点", len(settings))
	}
}

func TestWhatToTurnShouldRefuseADialItCannotTurn(t *testing.T) {
	for _, c := range []struct {
		name string
		opts *Options
		want string
	}{
		{"slot:列名 の形でない", &Options{Dial: "living_cost", To: 800, Step: 10}, "slot:列名"},
		{"数でない列", &Options{Dial: "household:続柄", To: 10, Step: 1}, "テキスト の列で、掃引できない"},
		{"無い列", &Options{Dial: "living_cost:無い列", To: 10, Step: 1}, "という列は無い"},
		{"無い slot", &Options{Dial: "無い:生活費[円/月]", To: 10, Step: 1}, "という slot は無い"},
		{"点が多すぎる", &Options{
			Dial: "living_cost:生活費[円/月]", From: 250_000, To: 500_000, Step: 10,
		}, "刻みを大きくすること"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := whatToTurn(c.opts, 2026)
			if err == nil {
				t.Fatalf("%+v を受け付けている", c.opts)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("%q を含まない: %v", c.want, err)
			}
		})
	}
}
