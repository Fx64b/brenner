package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fx64b/brenner/internal/cache"
	"github.com/fx64b/brenner/internal/device"
)

func TestStep(t *testing.T) {
	out := Step("Device", "/dev/sdb")
	for _, want := range []string{"✓", "Device", "/dev/sdb"} {
		if !strings.Contains(out, want) {
			t.Errorf("Step output %q missing %q", out, want)
		}
	}
}

func TestEmberColorSweepsThroughOrangeNotGreen(t *testing.T) {
	if got := emberColor(0); got != "#FF3D2E" {
		t.Errorf("emberColor(0) = %s, want #FF3D2E", got)
	}
	if got := emberColor(1); got != "#FFD166" {
		t.Errorf("emberColor(1) = %s, want #FFD166", got)
	}

	prevG := -1
	for i := 0; i <= 10; i++ {
		ti := float64(i) / 10
		var r, g, b int
		if _, err := fmt.Sscanf(emberColor(ti), "#%02X%02X%02X", &r, &g, &b); err != nil {
			t.Fatal(err)
		}
		if r != 0xFF {
			t.Errorf("t=%.1f: red=%d, want 255 (red stays maxed → orange path)", ti, r)
		}
		if g > r {
			t.Errorf("t=%.1f: green=%d exceeds red=%d (bows through green)", ti, g, r)
		}
		if g < prevG {
			t.Errorf("t=%.1f: green=%d decreased from %d (ramp not monotonic)", ti, g, prevG)
		}
		prevG = g
	}
}

func TestEmberBarFill(t *testing.T) {
	full := EmberBar(10, 1)
	if strings.Count(full, "█") != 10 {
		t.Errorf("EmberBar(10,1) should have 10 filled cells, got %d", strings.Count(full, "█"))
	}
	empty := EmberBar(10, 0)
	if strings.Contains(empty, "█") {
		t.Error("EmberBar(10,0) should have no filled cells")
	}
	if strings.Count(empty, "░") != 10 {
		t.Errorf("EmberBar(10,0) should be all track, got %d track cells", strings.Count(empty, "░"))
	}
	half := EmberBar(10, 0.5)
	if strings.Count(half, "█") != 5 {
		t.Errorf("EmberBar(10,0.5) should have 5 filled cells, got %d", strings.Count(half, "█"))
	}
}

func TestActionLabel(t *testing.T) {
	cases := map[string]string{
		ActionFlash: "Flash ISO",
		ActionWipe:  "Wipe",
		"custom":    "custom",
	}
	for in, want := range cases {
		if got := ActionLabel(in); got != want {
			t.Errorf("ActionLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeWipeMode(t *testing.T) {
	ok := map[string]string{
		"":          WipeQuick,
		"quick":     WipeQuick,
		"FAST":      WipeQuick,
		"zero":      WipeZero,
		"full":      WipeZero,
		"zero-fill": WipeZero,
		"secure":    WipeSecure,
		"random":    WipeSecure,
	}
	for in, want := range ok {
		got, err := NormalizeWipeMode(in)
		if err != nil {
			t.Errorf("NormalizeWipeMode(%q) errored: %v", in, err)
		}
		if got != want {
			t.Errorf("NormalizeWipeMode(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := NormalizeWipeMode("nuke"); err == nil {
		t.Error("expected an error for an unknown mode")
	}
}

func TestDeviceLabel(t *testing.T) {
	d := device.Device{Path: "/dev/sdb", Model: "Samsung Flash Drive", Size: 8_000_000_000}
	want := "[8.0 GB]  /dev/sdb - Samsung Flash Drive"
	if got := DeviceLabel(d); got != want {
		t.Errorf("DeviceLabel = %q, want %q", got, want)
	}
}

func TestDeviceLabelFallsBackToLabel(t *testing.T) {
	d := device.Device{Path: "/dev/sdc", Label: "KINGSTON", Size: 16_000_000_000}
	want := "[16.0 GB]  /dev/sdc - KINGSTON"
	if got := DeviceLabel(d); got != want {
		t.Errorf("DeviceLabel = %q, want %q", got, want)
	}
}

func TestIsoLabel(t *testing.T) {
	e := cache.Entry{Name: "arch.iso", Size: 842_500_000}
	want := "Cached: arch.iso (842.5 MB)"
	if got := IsoLabel(e); got != want {
		t.Errorf("IsoLabel = %q, want %q", got, want)
	}
}

func TestIsoOptionsAppendsActions(t *testing.T) {
	opts := isoOptions([]cache.Entry{{Name: "a.iso", Path: "/cache/a.iso", Size: 100}})
	if len(opts) != 3 {
		t.Fatalf("want 3 options, got %d", len(opts))
	}
	if opts[0].Value != "/cache/a.iso" {
		t.Errorf("first option value = %q", opts[0].Value)
	}
	if opts[1].Value != isoScanHome || opts[2].Value != isoManual {
		t.Errorf("trailing option values = %q, %q", opts[1].Value, opts[2].Value)
	}
}

func TestDeviceOptions(t *testing.T) {
	opts := deviceOptions([]device.Device{
		{Path: "/dev/sdb", Model: "Disk", Size: 1_000_000_000},
	})
	if len(opts) != 1 {
		t.Fatalf("want 1 option, got %d", len(opts))
	}
	if opts[0].Value != "/dev/sdb" {
		t.Errorf("option value = %q, want /dev/sdb", opts[0].Value)
	}
}
