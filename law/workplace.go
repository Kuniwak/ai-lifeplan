package law

import "fmt"

type Workplace struct {
	NormalWeeklyHours int

	Specified bool
}

func (w Workplace) shortTime(weeklyHours int) bool {
	if w.NormalWeeklyHours <= 0 {
		panic(fmt.Sprintf("law: 通常の労働者の一週間の所定労働時間が %d 時間である事業所は、"+
			"誰も四分の三未満にしない。表に書かれているか確かめること", w.NormalWeeklyHours))
	}

	return weeklyHours*4 < w.NormalWeeklyHours*3
}
