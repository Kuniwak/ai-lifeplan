package table_test

import (
	"testing"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/table"
)

func TestDeductionsByShouldPartitionWhatTheHouseholdIsCharged(t *testing.T) {
	for _, c := range []struct {
		name table.PersonName
		row  table.HouseholdInsuranceRow
		want map[table.PersonName]money.Yen
	}{
		{
			name: "国民健康保険税は世帯主に課されるので稼ぎ手が控除する",
			row:  table.HouseholdInsuranceRow{Kokuho: 300_000},
			want: map[table.PersonName]money.Yen{"夫": 300_000},
		},
		{
			name: "特別徴収されたぶんは天引きされた本人のもの",
			row: table.HouseholdInsuranceRow{
				NursingCareOf: []table.NursingCarePremium{
					{PersonPremium: table.PersonPremium{Name: "夫", Premium: 120_000, SpeciallyCollected: true}},
					{PersonPremium: table.PersonPremium{Name: "妻", Premium: 80_000, SpeciallyCollected: true}},
				},
			},
			want: map[table.PersonName]money.Yen{"夫": 120_000, "妻": 80_000},
		},
		{
			name: "普通徴収なら世帯が誰に払わせてもよいので稼ぎ手が控除する",
			row: table.HouseholdInsuranceRow{
				NursingCareOf: []table.NursingCarePremium{
					{PersonPremium: table.PersonPremium{Name: "妻", Premium: 80_000}},
				},
			},
			want: map[table.PersonName]money.Yen{"夫": 80_000},
		},
		{
			name: "後期だけが普通徴収に落ちる年もある",
			row: table.HouseholdInsuranceRow{
				NursingCareOf: []table.NursingCarePremium{
					{PersonPremium: table.PersonPremium{Name: "妻", Premium: 400_000, SpeciallyCollected: true}},
				},
				KoukiOf: []table.PersonPremium{{Name: "妻", Premium: 200_000}},
			},
			want: map[table.PersonName]money.Yen{"夫": 200_000, "妻": 400_000},
		},
		{
			name: "国民年金は窓口で払うので稼ぎ手が控除する",
			row: table.HouseholdInsuranceRow{
				NationalPensionOf: []table.PersonPremium{{Name: "子1", Premium: 200_000}},
			},
			want: map[table.PersonName]money.Yen{"夫": 200_000},
		},
	} {
		t.Run(string(c.name), func(t *testing.T) {
			got := c.row.DeductionsBy("夫")

			var total, charged money.Yen
			for _, amount := range got {
				total += amount
			}
			charged = c.row.Kokuho
			for _, split := range [][]table.PersonPremium{
				c.row.KoukiOf, c.row.NursingCarePremiums(), c.row.NationalPensionOf,
			} {
				for _, premium := range split {
					charged += premium.Premium
				}
			}
			if total != charged {
				t.Errorf("課された %d に対して控除は合計 %d。落ちたか二重になっている", charged, total)
			}

			for person, want := range c.want {
				if got[person] != want {
					t.Errorf("%s が控除するのは %d、%d のはず", person, got[person], want)
				}
			}
			for person, amount := range got {
				if _, named := c.want[person]; !named && amount != 0 {
					t.Errorf("%s が %d を控除している。誰も払っていない", person, amount)
				}
			}
		})
	}
}

func TestSocialInsuranceDeductionsShouldRefuseAYearItCannotJoin(t *testing.T) {
	household := relation.New([]relation.Row[table.HouseholdInsuranceRow]{
		{Year: 2030, Value: table.HouseholdInsuranceRow{Kokuho: 100_000}},
	})
	employed := map[table.PersonName]relation.Table[table.SocialInsuranceRow]{
		"夫": relation.New([]relation.Row[table.SocialInsuranceRow]{
			{Year: 2031, Value: table.SocialInsuranceRow{}},
		}),
	}

	if _, err := table.SocialInsuranceDeductions(household, employed, "夫"); err == nil {
		t.Error("年が揃っていないのに通った。控除が黙って 0 になる")
	}
}
