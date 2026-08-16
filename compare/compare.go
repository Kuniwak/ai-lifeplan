package compare

import (
	"github.com/Kuniwak/lifeplan/actuals"
	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/plan"
	"github.com/Kuniwak/lifeplan/tsv"
)

const RecordSeries = string(Record)

func RecordOf(subjects []Subject) actuals.BalanceTable {
	if len(subjects) == 0 {
		return actuals.BalanceTable{}
	}
	return subjects[0].Record
}

type Subject struct {
	Name string

	Paths map[tsv.Slot]string

	StartsAfter date.Year

	Record actuals.BalanceTable

	Tables map[plan.TableName]*tsv.Table

	Overridden map[tsv.Slot]bool

	UnreadColumns map[tsv.Slot][]tsv.ColumnName
}
