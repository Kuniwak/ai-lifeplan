package law

import (
	"fmt"
	"io/fs"
	"path"
	"slices"

	"github.com/Kuniwak/lifeplan/input"
	"github.com/Kuniwak/lifeplan/tsv"
	"github.com/Kuniwak/lifeplan/validate"
)

const LawDirectory = "data/law"

func Rules(fsys fs.FS) ([]validate.Rule, map[tsv.Slot]*tsv.Table, error) {
	tables := make(map[tsv.Slot]*tsv.Table)
	var rules []validate.Rule

	for _, shape := range Shapes() {
		names := []string{""}
		if shape.Regional {
			regions, err := RegionsWith(fsys, shape.Name)
			if err != nil {
				return nil, nil, fmt.Errorf("law: %w", err)
			}
			names = regions
		}

		for _, region := range names {
			table, err := LoadShape(fsys, shape, region)
			if err != nil {
				return nil, nil, fmt.Errorf("law: %w", err)
			}

			slot := TablePath(region, shape.Name)
			tables[slot] = table

			rules = append(rules,
				validate.Scoped(validate.LawSource(slot, LawSourceColumn), slot),
				validate.Scoped(validate.LawValidity(slot, shape.Start(), LawEndYearColumn), slot),
			)
			{
				rules = append(rules,
					validate.Scoped(validate.UniqueKey(slot, shape.Keys()), slot))
			}
			if shape.BandColumn != "" {
				rules = append(rules,
					validate.Scoped(validate.LawRangeTotal(slot, shape.BandColumn, shape.DomainMin), slot))
			}
		}
	}

	return rules, tables, nil
}

func MunicipalityRules(fsys fs.FS) ([]validate.Rule, error) {
	everyAddress, err := residentTaxGate(fsys)
	if err != nil {
		return nil, err
	}
	lastAddress, err := retirementGate(fsys)
	if err != nil {
		return nil, err
	}
	return []validate.Rule{everyAddress, lastAddress}, nil
}

func residentTaxGate(fsys fs.FS) (validate.Rule, error) {
	needs := ResidentTableNames()
	supported, err := RegionsWithAll(fsys, needs...)
	if err != nil {
		return validate.Rule{}, fmt.Errorf("law: %w", err)
	}

	short, err := missingUnder(fsys, needs)
	if err != nil {
		return validate.Rule{}, err
	}
	return validate.MunicipalitySupported(input.ResidenceSlot, input.MunicipalityColumn,
		validate.MunicipalityGate{
			Rule:      validate.MunicipalityRule,
			What:      "住む自治体",
			Supported: supported,
			Missing:   short,
		}), nil
}

func retirementGate(fsys fs.FS) (validate.Rule, error) {
	needs := LastMunicipalityTableNames()
	withOwn, err := RegionsWithAll(fsys, needs...)
	if err != nil {
		return validate.Rule{}, fmt.Errorf("law: %w", err)
	}
	short, err := missingUnder(fsys, needs)
	if err != nil {
		return validate.Rule{}, err
	}

	withKouki, err := RegionsWith(fsys, KoukiRatesTableName)
	if err != nil {
		return validate.Rule{}, fmt.Errorf("law: %w", err)
	}
	municipalities, err := LoadMunicipalities(fsys)
	if err != nil {
		return validate.Rule{}, fmt.Errorf("law: %w", err)
	}

	koukiOf := func(municipality Municipality) []string {
		prefectures, err := municipalities.PrefecturesOf(municipality)
		if err != nil {
			return []string{string(TablePath("", MunicipalitiesTableName)) + " に " + string(municipality) + " の行"}
		}
		var missing []string
		for _, prefecture := range prefectures {
			if !slices.Contains(withKouki, string(prefecture)) {
				missing = append(missing, string(TablePath(string(prefecture), KoukiRatesTableName)))
			}
		}
		return missing
	}

	supported := slices.DeleteFunc(withOwn, func(municipality string) bool {
		return len(koukiOf(Municipality(municipality))) > 0
	})
	return validate.MunicipalitySupported(input.ResidenceSlot, input.MunicipalityColumn,
		validate.MunicipalityGate{
			Rule:      validate.LastMunicipalityRule,
			What:      "住み終える自治体",
			LastOnly:  true,
			Supported: supported,
			Missing: func(municipality string) []string {
				return append(short(municipality), koukiOf(Municipality(municipality))...)
			},
		}), nil
}

func LastMunicipalityTableNames() []string {
	return []string{KokuhoTableName, PropertyTaxTableName, NursingCarePremiumTableName}
}

func TablePath(region, name string) tsv.Slot {
	return tsv.Slot(path.Join(LawDirectory, region, name) + ".tsv")
}

func missingUnder(fsys fs.FS, names []string) (func(string) []string, error) {
	has := make(map[string][]string, len(names))
	for _, name := range names {
		with, err := RegionsWith(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("law: %w", err)
		}
		for _, region := range with {
			has[region] = append(has[region], name)
		}
	}

	return func(municipality string) []string {
		var short []string
		for _, name := range names {
			if !slices.Contains(has[municipality], name) {
				short = append(short, string(TablePath(municipality, name)))
			}
		}
		return short
	}, nil
}
