package validate

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

func yenAt(slot tsv.Slot, row int, column tsv.ColumnName, written string) (money.Yen, *Finding) {
	amount, err := money.ParseYen(written)
	if err != nil {
		bad := unreadableField(slot, row, column, err)
		return 0, &bad
	}
	return amount, nil
}

func fieldFinding(slot tsv.Slot, row int, column tsv.ColumnName, what string) Finding {
	return Finding{
		Slot:    slot,
		Message: fmt.Sprintf("row %d: %s: %s", row+1, column, what),
	}
}

func unreadableField(slot tsv.Slot, row int, column tsv.ColumnName, err error) Finding {
	return fieldFinding(slot, row, column, err.Error())
}

func boundColumnFinding(
	slot tsv.Slot, year string, column tsv.ColumnName, got money.Yen,
	source string, want money.Yen,
) Finding {
	return Finding{
		Slot:    slot,
		Message: fmt.Sprintf("%s 年の %s が %d 円。%s は %d 円", year, column, got, source, want),
	}
}
