package law

import (
	"fmt"
	"io/fs"
	"path"
	"slices"

	"github.com/Kuniwak/lifeplan/date"
	"github.com/Kuniwak/lifeplan/relation"
	"github.com/Kuniwak/lifeplan/tsv"
)

func LoadRegionalTable(fsys fs.FS, region, name string) (*tsv.Table, error) {
	regions, err := RegionsWith(fsys, name)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(regions, region) {
		return nil, fmt.Errorf("law.LoadRegionalTable: no %s table for %q; the regions with one are %v", name, region, regions)
	}

	file, err := fsys.Open(path.Join(region, name+".tsv"))
	if err != nil {
		return nil, fmt.Errorf("law.LoadRegionalTable: %w", err)
	}
	defer file.Close()

	table, err := tsv.Read(file)
	if err != nil {
		return nil, fmt.Errorf("law.LoadRegionalTable: %s of %s: %w", name, region, err)
	}
	return table, nil
}

func RegionsWith(fsys fs.FS, name string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("law.RegionsWith: %w", err)
	}

	var regions []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := fs.Stat(fsys, path.Join(entry.Name(), name+".tsv")); err != nil {
			continue
		}
		regions = append(regions, entry.Name())
	}
	slices.Sort(regions)
	return regions, nil
}

func RegionsWithAll(fsys fs.FS, names ...string) ([]string, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("law.RegionsWithAll: 表の名前が 1 つも渡されていない。どの地域も条件を満たしてしまう")
	}

	var regions []string
	for i, name := range names {
		with, err := RegionsWith(fsys, name)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			regions = with
			continue
		}
		regions = slices.DeleteFunc(regions, func(region string) bool {
			return !slices.Contains(with, region)
		})
	}
	return regions, nil
}

const MunicipalitiesTableName = "national/municipalities"

const (
	MunicipalityColumn tsv.ColumnName = "自治体"
	PrefectureColumn   tsv.ColumnName = "都道府県"
)

type Municipalities struct {
	prefectureOf map[Municipality]relation.Periods[date.Year, Prefecture]
}

type (
	Municipality string
	Prefecture   string
)

func ParseMunicipalities(table *tsv.Table) (Municipalities, error) {
	r, err := newReader(table, MunicipalitiesTableName,
		MunicipalityColumn, PrefectureColumn, LawStartYearColumn, LawEndYearColumn)
	if err != nil {
		return Municipalities{}, fmt.Errorf("law.ParseMunicipalities: %w", err)
	}

	prefectureOf := make(map[Municipality][]relation.Period[date.Year, Prefecture], r.Rows())
	for row := range r.Rows() {
		municipality := Municipality(r.Field(row, MunicipalityColumn))
		prefecture := Prefecture(r.Field(row, PrefectureColumn))
		if municipality == "" {
			return Municipalities{}, fmt.Errorf("law.ParseMunicipalities: %w",
				r.Errorf(row, MunicipalityColumn, "自治体の欄が空で、どの住所についての行なのかが決まらない"))
		}
		if prefecture == "" {
			return Municipalities{}, fmt.Errorf("law.ParseMunicipalities: %w",
				r.Errorf(row, PrefectureColumn, "%s の県の欄が空で、後期高齢者医療の保険料をどの広域連合から引くかが決まらない", municipality))
		}
		from, err := r.startBound(row)
		if err != nil {
			return Municipalities{}, fmt.Errorf("law.ParseMunicipalities: %w", err)
		}
		to, err := r.endBound(row)
		if err != nil {
			return Municipalities{}, fmt.Errorf("law.ParseMunicipalities: %w", err)
		}
		band := relation.NewPeriod(from, to, prefecture)

		for _, written := range prefectureOf[municipality] {
			if overlap, ok := relation.Overlap(written, band); ok {
				return Municipalities{}, fmt.Errorf("law.ParseMunicipalities: %w",
					r.Errorf(row, MunicipalityColumn,
						"%d 年について %s と %s のどちらの県か決まらない。適用開始年と適用終了年が重なっている",
						overlap, written.Value(), prefecture))
			}
		}
		prefectureOf[municipality] = append(prefectureOf[municipality], band)
	}

	if len(prefectureOf) == 0 {
		return Municipalities{}, fmt.Errorf("law.ParseMunicipalities: %s に行が無いので、どの住所も都道府県に置けない", string(TablePath("", MunicipalitiesTableName)))
	}

	built := make(map[Municipality]relation.Periods[date.Year, Prefecture], len(prefectureOf))
	for municipality, periods := range prefectureOf {
		spans, err := relation.NewPeriods(periods)
		if err != nil {
			return Municipalities{}, fmt.Errorf("law.ParseMunicipalities: %s: %w", municipality, err)
		}
		built[municipality] = spans
	}
	return Municipalities{prefectureOf: built}, nil
}

func LoadMunicipalities(fsys fs.FS) (Municipalities, error) {
	table, err := LoadShape(fsys, Shape{Name: MunicipalitiesTableName}, "")
	if err != nil {
		return Municipalities{}, err
	}
	return ParseMunicipalities(table)
}

func (m Municipalities) PrefectureOf(municipality Municipality, year date.Year) (Prefecture, error) {
	bands, ok := m.prefectureOf[municipality]
	if !ok {
		return "", fmt.Errorf(
			"law.Municipalities.PrefectureOf: nobody has written which prefecture %q is in, and 後期高齢者医療 is charged by the prefecture; add a row to %s",
			municipality, TablePath("", MunicipalitiesTableName))
	}
	if prefecture, covered := bands.Lookup(year); covered {
		return prefecture, nil
	}
	return "", fmt.Errorf(
		"law.Municipalities.PrefectureOf: %s の %d 年の県が %s に書かれていない。後期高齢者医療は県ごとの広域連合が課すので、その年の県が決まらない",
		municipality, year, TablePath("", MunicipalitiesTableName))
}

func (m Municipalities) PrefecturesOf(municipality Municipality) ([]Prefecture, error) {
	bands, ok := m.prefectureOf[municipality]
	if !ok {
		return nil, fmt.Errorf(
			"law.Municipalities.PrefecturesOf: nobody has written which prefecture %q is in; add a row to %s",
			municipality, TablePath("", MunicipalitiesTableName))
	}
	out := make([]Prefecture, 0, bands.Len())
	for _, band := range bands.All() {
		out = append(out, band.Value())
	}
	return out, nil
}
