package law

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

func TestMunicipalitiesShouldBeReadFromTheStatutoryTable(t *testing.T) {
	municipalities, err := LoadMunicipalities(os.DirFS("../" + LawDirectory))
	if err != nil {
		t.Fatalf("law.LoadMunicipalities: %v", err)
	}

	for municipality, want := range map[string]string{
		"世田谷区": "東京都",
	} {
		got, err := municipalities.PrefectureOf(Municipality(municipality), 2026)
		if err != nil {
			t.Errorf("law.Municipalities.PrefectureOf(%q): %v", municipality, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s の県が %q になっている（%q のはず）", municipality, got, want)
		}
	}
}

func TestPrefectureOfShouldKnowEveryMunicipalityWithTables(t *testing.T) {
	fsys := os.DirFS("../" + LawDirectory)

	regions, err := RegionsWith(fsys, ResidentRateTableName)
	if err != nil {
		t.Fatalf("law.RegionsWith: %v", err)
	}
	if len(regions) == 0 {
		t.Fatal("住民税の表を持つ自治体が一つも無い。この検査が空回りしている")
	}

	municipalities, err := LoadMunicipalities(fsys)
	if err != nil {
		t.Fatalf("law.LoadMunicipalities: %v", err)
	}

	for _, region := range regions {
		if _, err := municipalities.PrefectureOf(Municipality(region), 2026); err != nil {
			t.Errorf("%s の住民税の表があるのに、県が書かれていない: %v", region, err)
		}
	}
}

func TestPrefectureOfShouldRefuseAMunicipalityNobodyHasWrittenDown(t *testing.T) {
	municipalities, err := LoadMunicipalities(os.DirFS("../" + LawDirectory))
	if err != nil {
		t.Fatalf("law.LoadMunicipalities: %v", err)
	}

	prefecture, err := municipalities.PrefectureOf("架空市", 2026)
	if err == nil {
		t.Fatalf("書かれていない自治体に %q を返している", prefecture)
	}
	if !strings.Contains(err.Error(), "架空市") {
		t.Errorf("どの自治体が書かれていないのかを言っていない: %v", err)
	}
}

func TestMunicipalitiesShouldRefuseARowThatDoesNotDecideAPrefecture(t *testing.T) {
	cases := map[string]struct {
		rows string
		says []string
	}{
		"一つの自治体の帯が重なっている": {
			rows: "架空市\t大阪府\t不明\t無期限\n架空市\t東京都\t2007\t無期限\n",
			says: []string{"大阪府", "東京都", "2007"},
		},
		"県の欄が空": {
			rows: "架空市\t\t不明\t無期限\n",
			says: []string{string(PrefectureColumn)},
		},
		"自治体の欄が空": {
			rows: "\t大阪府\t不明\t無期限\n",
			says: []string{string(MunicipalityColumn)},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			table, err := tsv.Read(strings.NewReader(
				municipalitiesHeader() + tc.rows))
			if err != nil {
				t.Fatalf("tsv.Read: %v", err)
			}

			municipalities, err := ParseMunicipalities(table)
			if err == nil {
				prefecture, _ := municipalities.PrefectureOf("架空市", 2026)
				t.Fatalf("通してしまい、架空市 に %q を返している", prefecture)
			}
			for _, want := range tc.says {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("何が悪いのかを言っていない（%q が無い）: %v", want, err)
				}
			}
		})
	}
}

func residentTaxOnly() fstest.MapFS {
	row := &fstest.MapFile{Data: []byte("適用開始年\t出典\n不明\thttps://example.invalid\n")}
	fsys := fstest.MapFS{
		"札幌市/" + ResidentRateTableName + ".tsv": row,

		MunicipalitiesTableName + ".tsv": &fstest.MapFile{Data: []byte(
			municipalitiesHeaderWithSource() +
				"不明\t架空市\t大阪府\t無期限\thttps://example.invalid\n" +
				"不明\t札幌市\t北海道\t無期限\thttps://example.invalid\n")},
		"大阪府/" + KoukiRatesTableName + ".tsv": row,
	}
	for _, name := range append(ResidentTableNames(), LastMunicipalityTableNames()...) {
		fsys["架空市/"+name+".tsv"] = row
	}
	return fsys
}

func TestRegionsWithAllShouldSkipAHalfWrittenDirectory(t *testing.T) {
	fsys := residentTaxOnly()

	rateOnly, err := RegionsWith(fsys, ResidentRateTableName)
	if err != nil {
		t.Fatalf("RegionsWith: %v", err)
	}
	if !slices.Contains(rateOnly, "札幌市") {
		t.Fatal("札幌市 が税率の表を持っていない。この検査の前提が崩れている")
	}

	all, err := RegionsWithAll(fsys, ResidentTableNames()...)
	if err != nil {
		t.Fatalf("RegionsWithAll: %v", err)
	}

	if slices.Contains(all, "札幌市") {
		t.Error("税率しか無い 札幌市 が住民税を書き上げた自治体に数えられている")
	}
	if !slices.Contains(all, "架空市") {
		t.Error("3 つとも持つ 架空市 が数えられていない")
	}
}

func TestTheMunicipalityGateShouldRefuseAHalfWrittenDirectory(t *testing.T) {
	rule := gateNamed(t, residentTaxOnly(), validate.MunicipalityRule)
	lived := func(municipality string) map[tsv.Slot]*tsv.Table {
		read, err := tsv.Read(strings.NewReader(
			"開始年\t" + string(input.MunicipalityColumn) + "\n2018\t" + municipality + "\n"))
		if err != nil {
			t.Fatalf("tsv.Read: %v", err)
		}
		return map[tsv.Slot]*tsv.Table{input.ResidenceSlot: read}
	}

	found := rule.Check(lived("札幌市"))
	if len(found) == 0 {
		t.Fatal("税率しか無い 札幌市 に住む計画が通ってしまう")
	}

	for _, name := range []string{ResidentPerCapitaTableName, ResidentExemptionTableName} {
		if !strings.Contains(found[0].Message, name+".tsv") {
			t.Errorf("findings が %s.tsv に触れていない: %s", name, found[0].Message)
		}
	}
	if strings.Contains(found[0].Message, ResidentRateTableName+".tsv") {
		t.Errorf("既にある %s.tsv を足せと言っている: %s", ResidentRateTableName, found[0].Message)
	}
	if found := rule.Check(lived("架空市")); len(found) != 0 {
		t.Errorf("3 つとも持つ 架空市 に住む計画が %v で拒まれる", found)
	}
}

func TestEveryDirectoryWithAResidentTaxTableShouldHaveAllThree(t *testing.T) {
	fsys := os.DirFS("../" + LawDirectory)

	withAny := make(map[string][]string)
	for _, name := range ResidentTableNames() {
		regions, err := RegionsWith(fsys, name)
		if err != nil {
			t.Fatalf("RegionsWith(%q): %v", name, err)
		}
		for _, region := range regions {
			withAny[region] = append(withAny[region], name)
		}
	}
	if len(withAny) == 0 {
		t.Fatal("住民税の表を持つディレクトリが一つも無い。この検査が空回りしている")
	}

	for region, names := range withAny {
		if len(names) != len(ResidentTableNames()) {
			t.Errorf("data/law/%s は住民税の表を %v しか持っていない（%v が要る）",
				region, names, ResidentTableNames())
		}
	}
}

func livedIn(t *testing.T, municipalities ...string) map[tsv.Slot]*tsv.Table {
	t.Helper()

	rows := "開始年\t" + string(input.MunicipalityColumn) + "\n"
	for i, municipality := range municipalities {
		rows += fmt.Sprintf("%d\t%s\n", 2018+i, municipality)
	}
	read, err := tsv.Read(strings.NewReader(rows))
	if err != nil {
		t.Fatalf("tsv.Read: %v", err)
	}
	return map[tsv.Slot]*tsv.Table{input.ResidenceSlot: read}
}

func TestTheLastMunicipalityGateShouldRefuseADirectoryWithoutTheRetirementTables(t *testing.T) {
	rule := gateNamed(t, residentTaxOnly(), validate.LastMunicipalityRule)

	for name, c := range map[string]struct {
		lived     []string
		wantFound bool
		named     []string
	}{
		"最後が書き上がっていれば通る": {lived: []string{"架空市"}},

		"最後が退職後の表を持たなければ拒む": {
			lived: []string{"架空市", "札幌市"}, wantFound: true,
			named: []string{KokuhoTableName, PropertyTaxTableName, NursingCarePremiumTableName, KoukiRatesTableName},
		},

		"途中に居ただけなら問われない": {lived: []string{"札幌市", "架空市"}},
	} {
		t.Run(name, func(t *testing.T) {
			found := rule.Check(livedIn(t, c.lived...))

			if !c.wantFound {
				if len(found) != 0 {
					t.Fatalf("%v に住む計画が %v で拒まれる", c.lived, found)
				}
				return
			}
			if len(found) != 1 {
				t.Fatalf("findings が %d 件ある（1 件のはず）: %v", len(found), found)
			}
			for _, want := range c.named {
				if !strings.Contains(found[0].Message, want+".tsv") {
					t.Errorf("%s.tsv に触れていない: %s", want, found[0].Message)
				}
			}

			for _, other := range ResidentTableNames() {
				if strings.Contains(found[0].Message, other+".tsv") {
					t.Errorf("退職後の門が住民税の %s.tsv に触れている: %s", other, found[0].Message)
				}
			}
		})
	}
}

func TestTheLastMunicipalityGateShouldAnswerAboutThePrefectureToo(t *testing.T) {
	row := &fstest.MapFile{Data: []byte("適用開始年\t出典\n不明\thttps://example.invalid\n")}
	header := municipalitiesHeaderWithSource()
	placed := "不明\t架空市\t大阪府\t無期限\thttps://example.invalid\n"
	elsewhere := "不明\t札幌市\t北海道\t無期限\thttps://example.invalid\n"

	for name, c := range map[string]struct {
		mapping string
		kouki   bool
		want    string
	}{
		"表は読めるが行が無い":             {mapping: header + elsewhere, want: MunicipalitiesTableName + ".tsv に 架空市 の行"},
		"置けるが県に kouki-rates が無い": {mapping: header + placed, want: "大阪府/" + KoukiRatesTableName + ".tsv"},
		"どちらも揃っていれば通る":           {mapping: header + placed, kouki: true},
	} {
		t.Run(name, func(t *testing.T) {
			fsys := fstest.MapFS{MunicipalitiesTableName + ".tsv": &fstest.MapFile{Data: []byte(c.mapping)}}
			for _, table := range append(ResidentTableNames(), LastMunicipalityTableNames()...) {
				fsys["架空市/"+table+".tsv"] = row
			}
			if c.kouki {
				fsys["大阪府/"+KoukiRatesTableName+".tsv"] = row
			}

			rule := gateNamed(t, fsys, validate.LastMunicipalityRule)

			found := rule.Check(livedIn(t, "架空市"))

			if c.want == "" {
				if len(found) != 0 {
					t.Fatalf("通るはずが %v で拒まれる", found)
				}
				return
			}
			if len(found) != 1 {
				t.Fatalf("findings が %d 件ある（1 件のはず）: %v", len(found), found)
			}
			if !strings.Contains(found[0].Message, c.want) {
				t.Errorf("%q に触れていない: %s", c.want, found[0].Message)
			}
			for _, table := range LastMunicipalityTableNames() {
				if strings.Contains(found[0].Message, "架空市/"+table+".tsv") {
					t.Errorf("既にある %s を足せと言っている: %s", table, found[0].Message)
				}
			}
		})
	}
}

func gateNamed(t *testing.T, fsys fs.FS, name validate.RuleName) validate.Rule {
	t.Helper()

	rules, err := MunicipalityRules(fsys)
	if err != nil {
		t.Fatalf("MunicipalityRules: %v", err)
	}
	for _, rule := range rules {
		if rule.Name == name {
			return rule
		}
	}
	t.Fatalf("%q という門が無い。あるのは %v", name, rules)
	return validate.Rule{}
}

func TestTheGatesShouldRefuseToBeBuiltOnAMappingThatWillNotLoad(t *testing.T) {
	row := &fstest.MapFile{Data: []byte("適用開始年\t出典\n不明\thttps://example.invalid\n")}
	header := municipalitiesHeaderWithSource()

	for name, mapping := range map[string]string{
		"行が無い": header,
		"自治体が二度書かれている": header +
			"不明\t架空市\t大阪府\t無期限\thttps://example.invalid\n" +
			"不明\t架空市\t東京都\t無期限\thttps://example.invalid\n",
	} {
		t.Run(name, func(t *testing.T) {
			fsys := fstest.MapFS{
				MunicipalitiesTableName + ".tsv":      &fstest.MapFile{Data: []byte(mapping)},
				"大阪府/" + KoukiRatesTableName + ".tsv": row,
			}
			for _, table := range append(ResidentTableNames(), LastMunicipalityTableNames()...) {
				fsys["架空市/"+table+".tsv"] = row
			}

			_, err := MunicipalityRules(fsys)

			if err == nil {
				t.Fatal("読めない 自治体→都道府県 の表で門が組み上がってしまう")
			}
			if !strings.Contains(err.Error(), MunicipalitiesTableName) {
				t.Errorf("どの表の話か言っていない: %v", err)
			}
		})
	}
}

func TestAMunicipalityMayChangePrefecture(t *testing.T) {
	table, err := tsv.Read(strings.NewReader(
		municipalitiesHeader() +
			"架空町\t大阪府\t不明\t2006\n" +
			"架空町\t東京都\t2007\t無期限\n"))
	if err != nil {
		t.Fatalf("tsv.Read: %v", err)
	}

	municipalities, err := ParseMunicipalities(table)
	if err != nil {
		t.Fatalf("県の変わった自治体を拒んでいる: %v", err)
	}

	for year, want := range map[int]Prefecture{
		1990: "大阪府",
		2006: "大阪府",
		2007: "東京都",
		2090: "東京都",
	} {
		got, err := municipalities.PrefectureOf("架空町", date.Year(year))
		if err != nil {
			t.Errorf("%d 年: %v", year, err)
			continue
		}
		if got != want {
			t.Errorf("%d 年の県が %q。%q のはず", year, got, want)
		}
	}

	prefectures, err := municipalities.PrefecturesOf("架空町")
	if err != nil {
		t.Fatalf("PrefecturesOf: %v", err)
	}
	if want := []Prefecture{"大阪府", "東京都"}; !slices.Equal(prefectures, want) {
		t.Errorf("PrefecturesOf = %v, want %v", prefectures, want)
	}
}

func TestPrefectureOfShouldRefuseAYearNoBandCovers(t *testing.T) {
	table, err := tsv.Read(strings.NewReader(
		municipalitiesHeader() +
			"架空町\t大阪府\t2000\t2006\n" +
			"架空町\t東京都\t2010\t無期限\n"))
	if err != nil {
		t.Fatalf("tsv.Read: %v", err)
	}
	municipalities, err := ParseMunicipalities(table)
	if err != nil {
		t.Fatalf("ParseMunicipalities: %v", err)
	}

	for _, year := range []int{1999, 2007, 2009} {
		if got, err := municipalities.PrefectureOf("架空町", date.Year(year)); err == nil {
			t.Errorf("%d 年に %q と答えている。どの帯も覆っていない年である", year, got)
		}
	}
}

func municipalitiesHeader() string {
	return string(MunicipalityColumn) + "\t" + string(PrefectureColumn) + "\t" +
		string(LawStartYearColumn) + "\t" + string(LawEndYearColumn) + "\n"
}

func municipalitiesHeaderWithSource() string {
	return string(LawStartYearColumn) + "\t" + string(MunicipalityColumn) + "\t" +
		string(PrefectureColumn) + "\t" + string(LawEndYearColumn) + "\t" +
		string(LawSourceColumn) + "\n"
}
