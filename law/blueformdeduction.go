package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/money"
)

type BlueFormRecordKeeping string

const (
	WhiteForm BlueFormRecordKeeping = "白色申告"

	BlueFormSimplified BlueFormRecordKeeping = "青色申告（簡易）"

	BlueFormDoubleEntry BlueFormRecordKeeping = "青色申告（複式）"

	BlueFormDoubleEntryElectronic BlueFormRecordKeeping = "青色申告（複式・電子帳簿保存またはe-Tax）"
)

var blueFormDeductionCeiling = map[BlueFormRecordKeeping]money.Yen{
	WhiteForm:                     0,
	BlueFormSimplified:            100_000,
	BlueFormDoubleEntry:           550_000,
	BlueFormDoubleEntryElectronic: 650_000,
}

func BlueFormDeduction(kind BlueFormRecordKeeping, incomeBeforeDeduction money.Yen) (money.Yen, error) {
	ceiling, ok := blueFormDeductionCeiling[kind]
	if !ok {
		return 0, fmt.Errorf(
			"law.BlueFormDeduction: %q is not a recognised 青色申告区分", kind)
	}
	if incomeBeforeDeduction <= 0 {
		return 0, nil
	}
	if incomeBeforeDeduction < ceiling {
		return incomeBeforeDeduction, nil
	}
	return ceiling, nil
}
