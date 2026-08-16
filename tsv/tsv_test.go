package tsv

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReadOK(t *testing.T) {
	type testCase struct {
		Input    string
		Expected *Table
	}

	testCases := map[string]testCase{
		"header and rows (representative)": {
			Input: "西暦\t年収\n2031\t1000000\n2053\t0\n",
			Expected: &Table{
				Header: []ColumnName{"西暦", "年収"},
				Rows:   [][]string{{"2031", "1000000"}, {"2053", "0"}},
			},
		},
		"header only is a well formed empty table (boundary value)": {
			Input:    "西暦\t年収\n",
			Expected: &Table{Header: []ColumnName{"西暦", "年収"}},
		},
		"single column (boundary value)": {
			Input:    "西暦\n2031\n",
			Expected: &Table{Header: []ColumnName{"西暦"}, Rows: [][]string{{"2031"}}},
		},
		"empty field is kept (representative)": {
			Input: "西暦\t備考\n2031\t\n",
			Expected: &Table{
				Header: []ColumnName{"西暦", "備考"},
				Rows:   [][]string{{"2031", ""}},
			},
		},
		"quoted field containing a tab (representative)": {
			Input: "西暦\t備考\n2031\t\"a\tb\"\n",
			Expected: &Table{
				Header: []ColumnName{"西暦", "備考"},
				Rows:   [][]string{{"2031", "a\tb"}},
			},
		},
		"quoted field containing a newline (representative)": {
			Input: "西暦\t備考\n2031\t\"a\nb\"\n",
			Expected: &Table{
				Header: []ColumnName{"西暦", "備考"},
				Rows:   [][]string{{"2031", "a\nb"}},
			},
		},
		"missing trailing newline (boundary value)": {
			Input:    "西暦\n2031",
			Expected: &Table{Header: []ColumnName{"西暦"}, Rows: [][]string{{"2031"}}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got, err := Read(strings.NewReader(tc.Input))

			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if diff := cmp.Diff(tc.Expected, got); diff != "" {
				t.Errorf("Read mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadNG(t *testing.T) {
	testCases := map[string]string{
		"empty file has no header, so the columns are unknown": "",
		"a row with too few fields":                            "西暦\t年収\n2031\n",
		"a row with too many fields":                           "西暦\t年収\n2031\t100\t200\n",
		"a duplicated column name makes lookup ambiguous":      "西暦\t西暦\n2031\t2032\n",
		"an empty column name cannot be referred to":           "西暦\t\n2031\t100\n",
	}

	for name, input := range testCases {
		t.Run(name, func(t *testing.T) {

			_, err := Read(strings.NewReader(input))

			if err == nil {
				t.Errorf("Read(%q): want error, got none", input)
			}
		})
	}
}

func TestWriteShouldSeparateWithTabsAndEndEveryLine(t *testing.T) {
	type testCase struct {
		Input    *Table
		Expected string
	}

	testCases := map[string]testCase{
		"header and rows (representative)": {
			Input: &Table{
				Header: []ColumnName{"西暦", "年収"},
				Rows:   [][]string{{"2031", "1000000"}, {"2053", "0"}},
			},
			Expected: "西暦\t年収\n2031\t1000000\n2053\t0\n",
		},
		"header only (boundary value)": {
			Input:    &Table{Header: []ColumnName{"西暦", "年収"}},
			Expected: "西暦\t年収\n",
		},
		"a field containing a tab is quoted": {
			Input: &Table{
				Header: []ColumnName{"西暦", "備考"},
				Rows:   [][]string{{"2031", "a\tb"}},
			},
			Expected: "西暦\t備考\n2031\t\"a\tb\"\n",
		},
		"the only field being empty is quoted so the row survives": {
			Input: &Table{
				Header: []ColumnName{"備考"},
				Rows:   [][]string{{""}, {"a"}, {""}},
			},
			Expected: "備考\n\"\"\na\n\"\"\n",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var sb strings.Builder

			err := Write(&sb, tc.Input)

			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if diff := cmp.Diff(tc.Expected, sb.String()); diff != "" {
				t.Errorf("Write mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWriteNG(t *testing.T) {
	testCases := map[string]*Table{
		"no header means the columns are unknown": {
			Rows: [][]string{{"2031"}},
		},
		"a row shorter than the header": {
			Header: []ColumnName{"西暦", "年収"},
			Rows:   [][]string{{"2031"}},
		},
		"a row longer than the header": {
			Header: []ColumnName{"西暦"},
			Rows:   [][]string{{"2031", "1000000"}},
		},
		"a duplicated column name": {
			Header: []ColumnName{"西暦", "西暦"},
		},
		"an empty column name": {
			Header: []ColumnName{"西暦", ""},
		},
	}

	for name, table := range testCases {
		t.Run(name, func(t *testing.T) {
			var sb strings.Builder

			err := Write(&sb, table)

			if err == nil {
				t.Errorf("want error, got none (wrote %q)", sb.String())
			}
		})
	}
}

func TestReadShouldRoundTripThroughWrite(t *testing.T) {
	original := &Table{
		Header: []ColumnName{"西暦", "備考"},
		Rows: [][]string{
			{"2031", "a\tb"},
			{"2053", "a\nb"},
			{"2054", ""},
			{"2055", `quote " inside`},
		},
	}
	var sb strings.Builder
	if err := Write(&sb, original); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read(strings.NewReader(sb.String()))

	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if diff := cmp.Diff(original, got); diff != "" {
		t.Errorf("round trip changed the table (-want +got):\n%s", diff)
	}
}

func TestTableColumnIndex(t *testing.T) {
	table := &Table{Header: []ColumnName{"西暦", "年収"}}

	if got, ok := table.ColumnIndex("年収"); !ok || got != 1 {
		t.Errorf(`ColumnIndex("年収") = (%d, %v), want (1, true)`, got, ok)
	}
	if _, ok := table.ColumnIndex("存在しない列"); ok {
		t.Error(`ColumnIndex("存在しない列") reported a match`)
	}
}

func TestReadShouldTakeOffTheByteOrderMarkExcelWrites(t *testing.T) {
	written := "\ufeff西暦\t自治体\n2018\t世田谷区\n"

	got, err := Read(strings.NewReader(written))

	if err != nil {
		t.Fatalf("tsv.Read: %v", err)
	}
	if diff := cmp.Diff([]ColumnName{"西暦", "自治体"}, got.Header); diff != "" {
		t.Errorf("Header mismatch (-want +got):\n%s", diff)
	}
	if _, ok := got.ColumnIndex("西暦"); !ok {
		t.Error(`ColumnIndex("西暦") missed, so the mark is still on the name`)
	}
}

func TestReadShouldRefuseAFileThatIsNotUTF8RatherThanMisreadIt(t *testing.T) {
	written := []byte{0x90, 0xbc, 0x97, 0xef, '\t', 0x8e, 0xa9, 0x8e, 0xa1, 0x91, 0xcc, '\n'}

	_, err := Read(bytes.NewReader(written))

	if err == nil {
		t.Fatal("want an error about the encoding, got none")
	}
	if !strings.Contains(err.Error(), "UTF-8") {
		t.Errorf("the error does not mention the encoding: %v", err)
	}
}
