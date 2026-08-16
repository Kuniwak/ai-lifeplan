package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Kuniwak/lifeplan/project"
	"github.com/Kuniwak/lifeplan/tsv"
)

type Origin string

const (
	OriginDefault Origin = "default"

	OriginCLI Origin = "cli"
)

type Layer struct {
	Origin Origin
	Slots  map[tsv.Slot]string
}

type Value struct {
	Path   string
	Origin Origin
}

type Config struct {
	values map[tsv.Slot]Value
}

func (c Config) Lookup(slot tsv.Slot) (Value, bool) {
	v, ok := c.values[slot]
	return v, ok
}

func (c Config) SlotNames() []tsv.Slot {
	return slices.Sorted(maps.Keys(c.values))
}

func Resolve(layers []Layer) Config {
	c := Config{values: make(map[tsv.Slot]Value)}
	for _, layer := range layers {
		for slot, path := range layer.Slots {
			c.values[slot] = Value{Path: path, Origin: layer.Origin}
		}
	}
	return c
}

func ParseSlotOverrides(args []string) (map[tsv.Slot]string, error) {
	slots := make(map[tsv.Slot]string, len(args))
	for _, arg := range args {
		if arg == "" {
			return nil, fmt.Errorf("config.ParseSlotOverrides: empty slot override")
		}

		name, path, found := strings.Cut(arg, "=")
		slot := tsv.Slot(name)
		if !found {
			return nil, fmt.Errorf("config.ParseSlotOverrides: %q has no %q, expected name=path", arg, "=")
		}
		if name == "" {
			return nil, fmt.Errorf("config.ParseSlotOverrides: %q has no slot name before the %q", arg, "=")
		}
		if path == "" {
			return nil, fmt.Errorf("config.ParseSlotOverrides: slot %q has no path (to leave it unset, omit the flag)", slot)
		}
		if _, dup := slots[slot]; dup {
			return nil, fmt.Errorf("config.ParseSlotOverrides: slot %q is overridden more than once", slot)
		}
		slots[slot] = path
	}
	return slots, nil
}

func LayersOf(loaded *project.Project, overrides map[tsv.Slot]string) ([]Layer, error) {
	var layers []Layer

	set := make(map[tsv.Slot]struct{}, len(loaded.SlotNames()))
	for _, slot := range loaded.SlotNames() {
		path, _ := loaded.Path(slot)
		manifest, _ := loaded.DecidedIn(slot)
		layers = append(layers, Layer{
			Origin: Origin(manifest),
			Slots:  map[tsv.Slot]string{slot: path},
		})
		set[slot] = struct{}{}
	}

	var unknown []string
	for slot := range overrides {
		if _, ok := set[slot]; !ok {
			unknown = append(unknown, string(slot))
		}
	}
	if len(unknown) > 0 {
		slices.Sort(unknown)
		return nil, fmt.Errorf(
			"config.LayersOf: %s sets no slot named %s, so there is nothing for the override to replace",
			loaded.ManifestPath(), strings.Join(unknown, ", "))
	}

	if len(overrides) > 0 {
		layers = append(layers, Layer{Origin: OriginCLI, Slots: overrides})
	}

	return layers, nil
}

func SlotPaths(manifestPath string, overrides map[tsv.Slot]string) (map[tsv.Slot]string, error) {
	loaded, err := project.Load(manifestPath)
	if err != nil {
		return nil, err
	}
	layers, err := LayersOf(loaded, overrides)
	if err != nil {
		return nil, err
	}

	settled := Resolve(layers)
	paths := make(map[tsv.Slot]string, len(settled.SlotNames()))
	for _, slot := range settled.SlotNames() {
		value, _ := settled.Lookup(slot)
		paths[slot] = value.Path
	}
	return paths, nil
}
