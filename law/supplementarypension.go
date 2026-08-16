package law

import (
	"fmt"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/tsv"
)

const SupplementaryPensionTableName = "national/supplementary-pension"

const (
	SpouseSupplementColumn        tsv.ColumnName = "配偶者加給年金額[円/年]"
	SpouseSupplementSpecialColumn tsv.ColumnName = "特別加算額[円/年]"
)

const SupplementaryPensionMonths = 240

const SpouseSupplementaryPensionAgeLimit = 65

func SpouseSupplementaryPension(amount, special money.Yen, insuredMonths int) money.Yen {
	if insuredMonths < SupplementaryPensionMonths {
		return 0
	}
	return amount + special
}

func SpouseSupplementaryPensionSuspended(spouseInsuredMonths int) bool {
	return spouseInsuredMonths >= SupplementaryPensionMonths
}

func SpouseSupplementaryPensionFrom(payableFrom date.Date) date.Date {
	return date.FirstOfMonth(payableFrom).AddMonths(1)
}

func SpouseSupplementaryPensionThrough(spouseBorn date.Date) date.Date {
	return date.FirstOfMonth(spouseBorn.ReachesAge(SpouseSupplementaryPensionAgeLimit))
}

func AssertSupplementaryPensionSpecialAddition(born date.Date) error {
	from := date.Date{Year: 1943, Month: 4, Day: 2}
	if born.Before(from) {
		return fmt.Errorf(
			"law.AssertSupplementaryPensionSpecialAddition: %s 生まれは昭和18年4月2日より前である。"+
				"特別加算額は生年月日で 36,000〜179,900 円に分かれ、%s はその最後の行しか持っていない"+
				"", born, SupplementaryPensionTableName)
	}
	return nil
}
