package table

import (
	"testing"

	"github.com/Kuniwak/lifeplan/money"
	"github.com/Kuniwak/lifeplan/panictest"
)

func TestTheShortfallShouldRefuseABalanceThatIsNotNegative(t *testing.T) {
	for name, c := range map[string]struct {
		cash       money.Yen
		want       money.Yen
		wantRefuse bool
	}{
		"払えなかった額が正で返る": {cash: -1_000, want: 1_000},
		"1 円足りない":      {cash: -1, want: 1},
		"ちょうど 0 は断る":   {cash: 0, wantRefuse: true},
		"余っていれば断る":     {cash: 1, wantRefuse: true},
		"大きく余っていても断る":  {cash: 1_000_000, wantRefuse: true},
	} {
		t.Run(name, func(t *testing.T) {

			var got money.Yen
			refused := panictest.Recovered(func() { got = shortfallOf(c.cash) })

			if c.wantRefuse {
				if refused == nil {
					t.Errorf("残高 %d を受け付けている。払えている年に不足を立ててはいけない", c.cash)
				}
				return
			}
			if refused != nil {
				t.Fatalf("残高 %d を断っている: %v", c.cash, refused)
			}
			if got != c.want {
				t.Errorf("shortfallOf(%d) = %d, want %d", c.cash, got, c.want)
			}
		})
	}
}
