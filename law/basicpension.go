package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const BasicPensionFullTableName = "national/basic-pension"

const BasicPensionFullColumn tsv.ColumnName = "満額[円/年]"

const BasicPensionFullMonths = 480

func OldAgeBasicPension(full money.Yen, paidMonths int) (money.Yen, error) {
	switch {
	case paidMonths < 0:
		return 0, fmt.Errorf("law.OldAgeBasicPension: 納付済月数が %d である", paidMonths)
	case paidMonths > BasicPensionFullMonths:
		return 0, fmt.Errorf(
			"law.OldAgeBasicPension: 納付済月数 %d が上限の %d 月を超えている。"+
				"20 歳から 60 歳までに %d 月しかないので、これは数え方の誤りである"+
				"——60 歳以降の厚生年金は経過的加算として老齢厚生年金に乗るのであって、"+
				"老齢基礎年金には入らない",
			paidMonths, BasicPensionFullMonths, BasicPensionFullMonths)
	}
	return full.Mul(money.NewRate(int64(paidMonths), BasicPensionFullMonths), money.HalfUp), nil
}
