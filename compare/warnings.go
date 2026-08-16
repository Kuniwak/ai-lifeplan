package compare

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/input"
)

func Warnings(subjects []Subject) []string {
	var out []string
	if warning := mixedOrigins(subjects); warning != "" {
		out = append(out, warning)
	}
	if warning := mixedRecords(subjects); warning != "" {
		out = append(out, warning)
	}
	if warning := settledOnTheCommandLine(subjects); warning != "" {
		out = append(out, warning)
	}
	out = append(out, unclassifiedSlots(subjects)...)
	return out
}

func settledOnTheCommandLine(subjects []Subject) string {
	var said []string
	for _, difference := range Differences(subjects) {
		if difference.Class != FromCLI {
			continue
		}
		said = append(said, fmt.Sprintf("%s は %s", difference.Slot, difference.Paths[0]))
	}
	if len(said) == 0 {
		return ""
	}

	return fmt.Sprintf(
		"コマンド引数で差し替えた表がある（%s）。この結果はマニフェストのままの結果ではなく、"+
			"-slot-override を外して走らせれば別の数字になる",
		strings.Join(said, "・"))
}

func mixedRecords(subjects []Subject) string {
	if len(subjects) < 2 {
		return ""
	}

	same := true
	said := make([]string, len(subjects))
	for i, subject := range subjects {
		said[i] = fmt.Sprintf("%s は %s", subject.Name, subject.Paths[input.BalanceSlot])
		if !sameRecord(subject.Record, subjects[0].Record) {
			same = false
		}
	}
	if same {
		return ""
	}

	return fmt.Sprintf(
		"読んでいる実績が揃っていない（%s）。乖離は %s の実績に対して測ってある",
		strings.Join(said, "・"), subjects[0].Name)
}

func sameRecord(a, b actuals.BalanceTable) bool {
	years := a.Years()
	if !slices.Equal(years, b.Years()) {
		return false
	}
	for _, year := range years {
		was, _ := a.At(year)
		is, _ := b.At(year)
		if was != is {
			return false
		}
	}
	return true
}

func mixedOrigins(subjects []Subject) string {
	if len(subjects) < 2 {
		return ""
	}

	same := true
	said := make([]string, len(subjects))
	for i, subject := range subjects {
		said[i] = fmt.Sprintf("%s は %d 年末", subject.Name, subject.StartsAfter)
		if subject.StartsAfter != subjects[0].StartsAfter {
			same = false
		}
	}
	if same {
		return ""
	}

	return fmt.Sprintf(
		"起点が揃っていない（%s）。起点が違えば資産の水準そのものが違うので、"+
			"差の大半は計画の優劣ではなく起点の差である",
		strings.Join(said, "・"))
}

func unclassifiedSlots(subjects []Subject) []string {
	var out []string
	for _, difference := range Differences(subjects) {
		if difference.Class != Unclassified {
			continue
		}
		out = append(out, fmt.Sprintf(
			"slot %s が 入力・環境・実績 のどれか決められない。"+
				"data/controllable・data/environment・actuals のいずれかの下に置くと決まる",
			difference.Slot))
	}
	return out
}
