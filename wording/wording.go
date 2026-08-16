package wording

import (
	"fmt"
	"strings"
)

type Key struct{ written string }

func Name[S ~string](s S) Key { return Key{written: fmt.Sprintf("%q", string(s))} }

func Number[N ~int](n N) Key { return Key{written: fmt.Sprint(int(n))} }

func Pair[S ~string](parts ...S) Key {
	written := make([]string, 0, len(parts))
	for _, part := range parts {
		written = append(written, string(part))
	}
	return Key{written: strings.Join(written, " / ")}
}

func (k Key) String() string { return k.written }

type Undecidable string

type Undecided string

const (
	WhichRowReadsTheYear     Undecidable = "その年をどの行で読むのか決まらない"
	WhichRowAppliesInTheYear Undecidable = "その年にどの行が効くのか決まらない"
	WhichAmountIsTheRecord   Undecidable = "どちらの額を別の記録とするのか決まらない"
	WhichAnswerIsTaken       Undecidable = "どちらの答えが採られるのか決まらない"
	WhichRowAnswersForTheKey Undecidable = "どちらの行がその鍵について答えるのか決まらない"
	WhichRowDecidesTheValue  Undecidable = "どちらの行がその値を決めるのか決まらない"

	WhichRowReadsTheYearEn     Undecided = "which row reads that year"
	WhichRowAppliesInTheYearEn Undecided = "which row applies in it"
	WhichAmountIsTheRecordEn   Undecided = "which amount is the separate record"
	WhichHoldingItCountsAsEn   Undecided = "which row's holding it counts as"
)

func DuplicateKeyFinding(kind string, key Key, undecidable Undecidable) string {
	return fmt.Sprintf("%s %s が二度書かれており、%s", kind, key, undecidable)
}

func OutOfAscendingOrderFinding(kind string, previous, current Key) string {
	return fmt.Sprintf("%s %s が %s の後に来ている。%sは昇順でなければならない",
		kind, current, previous, kind)
}

func DuplicateKeyError(where, kind string, key Key, undecided Undecided) error {
	return fmt.Errorf("%s: the %s %s is written twice, so %s is undecided", where, kind, key, undecided)
}

func OutOfAscendingOrderError(where, kind string, previous, current Key) error {
	return fmt.Errorf("%s: the %s %s comes after %s, and they have to ascend",
		where, kind, current, previous)
}
