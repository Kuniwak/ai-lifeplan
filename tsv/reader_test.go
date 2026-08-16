package tsv_test

import (
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/tsv"
)

func aTable() *tsv.Table {
	return &tsv.Table{
		Header: []tsv.ColumnName{"西暦", "自治体", "人数"},
		Rows:   [][]string{{"2018", "世田谷区", " 3 "}, {"2023", "架空市", "4"}},
	}
}

func TestNewReaderShouldRefuseATableThatIsNotThere(t *testing.T) {
	_, err := tsv.NewReader(nil, "residence", "西暦")

	if err == nil {
		t.Fatal("want an error about the table not being there, got none")
	}
	if !strings.Contains(err.Error(), "residence") {
		t.Errorf("the error does not name what was being read: %v", err)
	}
}

func TestNewReaderShouldRefuseAColumnThatIsNotThere(t *testing.T) {
	_, err := tsv.NewReader(aTable(), "residence", "西暦", "無い列")

	if err == nil {
		t.Fatal("want an error about the missing column, got none")
	}
	for _, want := range []string{"residence", "無い列"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q: %v", want, err)
		}
	}
}

func TestFieldShouldReadByNameRatherThanPosition(t *testing.T) {
	r, err := tsv.NewReader(aTable(), "residence", "自治体")
	if err != nil {
		t.Fatalf("tsv.NewReader: %v", err)
	}

	if got, want := r.Rows(), 2; got != want {
		t.Errorf("Rows() = %d, want %d", got, want)
	}
	if got, want := r.Field(1, "自治体"), "架空市"; got != want {
		t.Errorf("Field = %q, want %q", got, want)
	}
}

func TestCountShouldIgnoreSpaceAroundTheNumber(t *testing.T) {
	r, err := tsv.NewReader(aTable(), "household", "人数")
	if err != nil {
		t.Fatalf("tsv.NewReader: %v", err)
	}

	got, err := r.Count(0, "人数")

	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if want := 3; got != want {
		t.Errorf("Count = %d, want %d", got, want)
	}
}

func TestCountShouldSayWhichRowAndColumnItCouldNotRead(t *testing.T) {
	table := &tsv.Table{Header: []tsv.ColumnName{"人数"}, Rows: [][]string{{"さん"}}}
	r, err := tsv.NewReader(table, "household", "人数")
	if err != nil {
		t.Fatalf("tsv.NewReader: %v", err)
	}

	_, err = r.Count(0, "人数")

	if err == nil {
		t.Fatal("want an error, got none")
	}
	for _, want := range []string{"household", "1", "人数", "さん"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q: %v", want, err)
		}
	}
}

func TestFieldShouldRefuseAColumnTheReaderWasNotAskedFor(t *testing.T) {
	table := &tsv.Table{
		Header: []tsv.ColumnName{"適用開始年", "額[円]"},
		Rows:   [][]string{{"不明", "1000"}},
	}
	reader, err := tsv.NewReader(table, "例の表", "適用開始年", "額[円]")
	if err != nil {
		t.Fatalf("tsv.NewReader: %v", err)
	}

	if got := reader.Field(0, "適用開始年"); got != "不明" {
		t.Errorf("名指した列が %q になっている", got)
	}
	defer func() {
		said, panicked := recover().(string)
		if !panicked {
			t.Fatal("読み手に渡していない列を読めてしまった")
		}
		if !strings.Contains(said, "額[円]") || !strings.Contains(said, "適用終了年") {
			t.Errorf("言われたのは %q だが、どの列を訊いたのかと何が読めるのかを言ってほしい", said)
		}
	}()
	reader.Field(0, "適用終了年")
}
