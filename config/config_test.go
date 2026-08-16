package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/Kuniwak/lifeplan/project"
	"github.com/Kuniwak/lifeplan/tsv"
)

func TestResolveShouldLetTheLaterLayerWin(t *testing.T) {
	type testCase struct {
		Layers       []Layer
		Slot         tsv.Slot
		ExpectedPath string
		ExpectedFrom Origin
	}

	base := Layer{
		Origin: "projects/base.tsv",
		Slots:  map[tsv.Slot]string{"income_wife": "data/income_wife.tsv"},
	}
	overriding := Layer{
		Origin: "projects/wife-fulltime.tsv",
		Slots:  map[tsv.Slot]string{"income_wife": "data/income_wife-fulltime.tsv"},
	}
	fromArgs := Layer{
		Origin: OriginCLI,
		Slots:  map[tsv.Slot]string{"income_wife": "data/income_wife-probe.tsv"},
	}
	defaults := Layer{
		Origin: OriginDefault,
		Slots:  map[tsv.Slot]string{"income_wife": "data/income_wife-default.tsv"},
	}

	testCases := map[string]testCase{
		"only the defaults": {
			Layers: []Layer{defaults}, Slot: "income_wife",
			ExpectedPath: "data/income_wife-default.tsv", ExpectedFrom: OriginDefault,
		},
		"the project beats the defaults": {
			Layers: []Layer{defaults, base}, Slot: "income_wife",
			ExpectedPath: "data/income_wife.tsv", ExpectedFrom: "projects/base.tsv",
		},
		"a later project beats an earlier one": {
			Layers: []Layer{defaults, base, overriding}, Slot: "income_wife",
			ExpectedPath: "data/income_wife-fulltime.tsv", ExpectedFrom: "projects/wife-fulltime.tsv",
		},
		"the command line beats everything": {
			Layers: []Layer{defaults, base, overriding, fromArgs}, Slot: "income_wife",
			ExpectedPath: "data/income_wife-probe.tsv", ExpectedFrom: OriginCLI,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got := Resolve(tc.Layers)

			value, ok := got.Lookup(tc.Slot)
			if !ok {
				t.Fatalf("Lookup(%q) found nothing", tc.Slot)
			}
			if value.Path != tc.ExpectedPath {
				t.Errorf("Path = %q, want %q", value.Path, tc.ExpectedPath)
			}
			if value.Origin != tc.ExpectedFrom {
				t.Errorf("Origin = %q, want %q", value.Origin, tc.ExpectedFrom)
			}
		})
	}
}

func TestResolveShouldUnionTheSlotsOfEveryLayer(t *testing.T) {
	layers := []Layer{
		{Origin: OriginDefault, Slots: map[tsv.Slot]string{"household": "data/household.tsv"}},
		{Origin: "projects/base.tsv", Slots: map[tsv.Slot]string{"market": "data/market.tsv"}},
		{Origin: OriginCLI, Slots: map[tsv.Slot]string{"balance": "actuals/balance.tsv"}},
	}

	got := Resolve(layers)

	want := []tsv.Slot{"balance", "household", "market"}
	if diff := cmp.Diff(want, got.SlotNames()); diff != "" {
		t.Errorf("SlotNames mismatch (-want +got):\n%s", diff)
	}
}

func TestResolveShouldReportNothingForAnUnsetSlot(t *testing.T) {
	got := Resolve([]Layer{{Origin: OriginDefault, Slots: map[tsv.Slot]string{"household": "x"}}})

	if _, ok := got.Lookup("market"); ok {
		t.Error("Lookup reported a slot that no layer set")
	}
}

func TestResolveWithNoLayers(t *testing.T) {
	got := Resolve(nil)

	if got.SlotNames() != nil {
		t.Errorf("SlotNames() = %v, want nil", got.SlotNames())
	}
}

func TestParseSlotOverridesOK(t *testing.T) {
	type testCase struct {
		Args     []string
		Expected map[tsv.Slot]string
	}

	testCases := map[string]testCase{
		"a single override (representative)": {
			Args:     []string{"market=data/environment/market-minus2.tsv"},
			Expected: map[tsv.Slot]string{"market": "data/environment/market-minus2.tsv"},
		},
		"several overrides": {
			Args: []string{"market=a.tsv", "living_cost=b.tsv"},
			Expected: map[tsv.Slot]string{
				"market":      "a.tsv",
				"living_cost": "b.tsv",
			},
		},
		"a path containing an equals sign keeps it": {
			Args:     []string{"market=data/a=b.tsv"},
			Expected: map[tsv.Slot]string{"market": "data/a=b.tsv"},
		},
		"no overrides at all": {
			Args:     nil,
			Expected: map[tsv.Slot]string{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			got, err := ParseSlotOverrides(tc.Args)

			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if diff := cmp.Diff(tc.Expected, got); diff != "" {
				t.Errorf("ParseSlotOverrides mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseSlotOverridesNG(t *testing.T) {
	testCases := map[string]struct {
		Args     []string
		Mentions string
	}{
		"no equals sign":      {Args: []string{"market"}, Mentions: "market"},
		"no slot name":        {Args: []string{"=a.tsv"}, Mentions: "a.tsv"},
		"no path":             {Args: []string{"market="}, Mentions: "market"},
		"the same slot twice": {Args: []string{"market=a.tsv", "market=b.tsv"}, Mentions: "market"},
		"an empty argument":   {Args: []string{""}, Mentions: "empty"},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			_, err := ParseSlotOverrides(tc.Args)

			if err == nil {
				t.Fatal("want error, got none")
			}
			if !strings.Contains(err.Error(), tc.Mentions) {
				t.Errorf("the message does not mention %q: %v", tc.Mentions, err)
			}
		})
	}
}

func TestLayersOfShouldRefuseAnOverrideOfASlotNothingSets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "base.tsv")
	if err := os.WriteFile(path,
		[]byte("slot\tpath\ninflation\tdata/environment/inflation.tsv\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	loaded, err := project.Load(path)
	if err != nil {
		t.Fatalf("project.Load: %v", err)
	}

	_, err = LayersOf(loaded, map[tsv.Slot]string{"balnace": "x.tsv"})

	if err == nil {
		t.Fatal("LayersOf accepted an override of a slot nothing sets")
	}
	if !strings.Contains(err.Error(), "balnace") {
		t.Errorf("%q does not name the slot", err)
	}
}

func TestLayersOfShouldAcceptAnOverrideOfASlotTheManifestSets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "base.tsv")
	if err := os.WriteFile(path,
		[]byte("slot\tpath\ninflation\tdata/environment/inflation.tsv\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	loaded, err := project.Load(path)
	if err != nil {
		t.Fatalf("project.Load: %v", err)
	}

	layers, err := LayersOf(loaded, map[tsv.Slot]string{"inflation": "/dev/fd/63"})

	if err != nil {
		t.Fatalf("LayersOf: %v", err)
	}
	value, ok := Resolve(layers).Lookup("inflation")
	if !ok {
		t.Fatal("Lookup found nothing")
	}
	if value.Path != "/dev/fd/63" || value.Origin != OriginCLI {
		t.Errorf("path %q origin %q, want %q from %q", value.Path, value.Origin, "/dev/fd/63", OriginCLI)
	}
}

func TestSlotPathsShouldSettleTheManifestAgainstTheOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "base.tsv")
	if err := os.WriteFile(path, []byte(
		"slot\tpath\ninflation\tdata/environment/inflation.tsv\nbalance\tactuals/balance.tsv\n",
	), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := SlotPaths(path, map[tsv.Slot]string{"balance": "actuals/balance-2022.tsv"})

	if err != nil {
		t.Fatalf("SlotPaths: %v", err)
	}
	want := map[tsv.Slot]string{
		"inflation": "data/environment/inflation.tsv",
		"balance":   "actuals/balance-2022.tsv",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SlotPaths mismatch (-want +got):\n%s", diff)
	}
}

func TestSlotPathsShouldRefuseAnOverrideOfASlotNothingSets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "base.tsv")
	if err := os.WriteFile(path,
		[]byte("slot\tpath\ninflation\tdata/environment/inflation.tsv\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := SlotPaths(path, map[tsv.Slot]string{"balnace": "x.tsv"})

	if err == nil {
		t.Fatal("SlotPaths accepted an override of a slot nothing sets")
	}
}
