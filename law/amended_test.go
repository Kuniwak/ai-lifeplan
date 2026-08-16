package law

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/panictest"
)

func TestAmendedShouldAnswerForEveryYear(t *testing.T) {
	amended := NewAmended("例の額",
		YearRow[int]{FromYear: 0, Value: 1},
		YearRow[int]{FromYear: 2021, Value: 2},
		YearRow[int]{FromYear: 2025, Value: 3},
	)

	cases := map[string]struct {
		year int
		want int
	}{
		"表の最初の行より前":    {year: 1900, want: 1},
		"2020 年（境界値）":  {year: 2020, want: 1},
		"2021 年（境界値）":  {year: 2021, want: 2},
		"帯の途中":         {year: 2023, want: 2},
		"2024 年（境界値）":  {year: 2024, want: 2},
		"2025 年（境界値）":  {year: 2025, want: 3},
		"最後の行より後はそのまま": {year: 2100, want: 3},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := amended.At(date.Year(c.year))

			if got != c.want {
				t.Errorf("At(%d) = %d, want %d", c.year, got, c.want)
			}
		})
	}
}

func TestAmendedShouldRefuseATableThatCannotAnswer(t *testing.T) {
	cases := map[string]func(){
		"行が無い": func() { NewAmended[int]("例の額") },
		"最初の行が 0 年から始まらない":    func() { NewAmended("例の額", YearRow[int]{FromYear: 2021, Value: 2}) },
		"記録の始まりを言ったが行が無い":     func() { NewAmendedFrom[int]("例の額", 2017) },
		"記録の始まりといちばん古い行が食い違う": func() { NewAmendedFrom("例の額", 2017, YearRow[int]{FromYear: 2021, Value: 2}) },
	}

	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			refused := panictest.Recovered(build)

			if refused == nil {
				t.Error("答えられない表が受け付けられた")
			}
		})
	}
}

func TestTheZeroAmendedShouldSayItWasNotBuilt(t *testing.T) {
	var zero Amended[int]

	refused := panictest.Recovered(func() { zero.At(2025) })

	if refused == nil {
		t.Error("組み立てられていない表が答えを返した")
	}
}

func TestAmendedFromShouldRefuseAYearBeforeItsRecord(t *testing.T) {
	const name = "例の額"

	amended := NewAmendedFrom(name, 2017,
		YearRow[int]{FromYear: 2017, Value: 1},
		YearRow[int]{FromYear: 2021, Value: 2},
	)

	type testCase struct {
		year        date.Year
		want        int
		wantRefused bool
	}

	testCases := map[string]testCase{
		"記録の最初の年（境界値）": {year: 2017, want: 1},
		"その 1 年前（境界値）": {year: 2016, wantRefused: true},
		"はるか前":         {year: 1900, wantRefused: true},
		"次の行（境界値）":     {year: 2021, want: 2},
		"最後の行より後はそのまま": {year: 2100, want: 2},
	}

	for caseName, tc := range testCases {
		t.Run(caseName, func(t *testing.T) {
			var got int

			refused := panictest.Recovered(func() { got = amended.At(tc.year) })

			if !tc.wantRefused {
				if refused != nil {
					t.Fatalf("%d 年が拒まれた: %v", tc.year, refused)
				}
				if got != tc.want {
					t.Errorf("At(%d) = %d, want %d", tc.year, got, tc.want)
				}
				return
			}
			if refused == nil {
				t.Fatalf("記録より前の %d 年が黙って %d と答えられた", tc.year, got)
			}
			message, _ := refused.(string)
			for _, want := range []string{name, "2017"} {
				if !strings.Contains(message, want) {
					t.Errorf("panic のメッセージに %q が無い: %s", want, message)
				}
			}
		})
	}
}

func TestEveryRecordFloorShouldStartWhereItsReasonSays(t *testing.T) {
	want := map[string]date.Year{
		"障害者等の非課税限度額":       2017,
		"均等割非課税限度額の基本額に足す額": 2017,
		"所得割の非課税限度額":        2017,

		"配偶者控除等の納税者本人の所得段階": 2018,
		"配偶者控除":          2018,
		"配偶者特別控除":        2018,
		"控除対象配偶者の人的控除の差": 2018,
		"配偶者特別控除の人的控除の差": 2018,
	}

	floors := RecordFloors()
	if len(floors) != len(want) {
		t.Fatalf("床が %d 個ある（%d 個のはず）。増やしたならその年もここに書くこと", len(floors), len(want))
	}

	for _, floor := range floors {
		t.Run(floor.What, func(t *testing.T) {
			expected, named := want[floor.What]
			if !named {
				t.Fatalf("この名前の床は覚えられていない。%v のどれかのはず", want)
			}

			first, ok := floor.FirstWritten()

			if !ok {
				t.Fatal("記録の始まりを持っていない")
			}
			if first != expected {
				t.Errorf("記録の始まりが %d 年である（%d 年のはず）", first, expected)
			}
		})
	}
}

func TestEveryAmendedWithARecordFloorShouldBeListed(t *testing.T) {
	listed := make(map[string]bool, len(RecordFloors()))
	for _, floor := range RecordFloors() {
		listed[floor.What] = true
	}

	declared, err := amendedNamesInThisPackage()
	if err != nil {
		t.Fatalf("パッケージの読み取り: %v", err)
	}
	if len(declared) < len(listed) {
		t.Fatalf("NewAmendedFrom の呼び出しが %d 件しか見つからない。床は %d 個ある", len(declared), len(listed))
	}

	for _, name := range declared {
		if !listed[name] {
			t.Errorf("%q は NewAmendedFrom で作られているのに RecordFloors に載っていない。"+
				"plan.build が届かない年を先に拒めず、panic になる", name)
		}
	}
}

func amendedNamesInThisPackage() ([]string, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}

	call := regexp.MustCompile(`NewAmendedFrom\(("(?:[^"\\]|\\.)*")`)
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		for _, match := range call.FindAllStringSubmatch(string(source), -1) {
			quoted, err := strconv.Unquote(match[1])
			if err != nil {
				return nil, err
			}
			names = append(names, quoted)
		}
	}
	return names, nil
}
