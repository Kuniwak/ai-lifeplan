package law_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/law"
	"github.com/Kuniwak/lifeplan/money"
)

func TestTheSpecialCollectionFloorShouldBeTheOneThePolicySets(t *testing.T) {
	if want := money.Yen(180_000); law.SpecialCollectionPensionFloor != want {
		t.Errorf("特別徴収の敷居 %d、%d のはず（介護保険法施行令第四十一条）",
			law.SpecialCollectionPensionFloor, want)
	}
}
