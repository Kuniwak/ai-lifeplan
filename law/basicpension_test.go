package law_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

const theFullAmountOfReiwaSeven money.Yen = 831_700

func TestTheFullBasicPensionShouldBeTheFullAmount(t *testing.T) {
	got, err := law.OldAgeBasicPension(theFullAmountOfReiwaSeven, law.BasicPensionFullMonths)
	if err != nil {
		t.Fatalf("law.OldAgeBasicPension: %v", err)
	}
	if got != theFullAmountOfReiwaSeven {
		t.Errorf("480 月の老齢基礎年金 = %d、満額 %d のはず", got, theFullAmountOfReiwaSeven)
	}
}

func TestMoreMonthsThanTheStatuteAllowsShouldBeRefused(t *testing.T) {
	_, err := law.OldAgeBasicPension(theFullAmountOfReiwaSeven, law.BasicPensionFullMonths+1)
	if err == nil {
		t.Fatal("480 月を超える納付済月数が通った。20 歳から60 歳までに 480 月しかない")
	}
	if !strings.Contains(err.Error(), "480") {
		t.Errorf("エラーが月数の上限を言っていない: %v", err)
	}
}

func TestNegativeMonthsShouldBeRefused(t *testing.T) {
	if _, err := law.OldAgeBasicPension(theFullAmountOfReiwaSeven, -1); err == nil {
		t.Fatal("負の納付済月数が通った")
	}
}
