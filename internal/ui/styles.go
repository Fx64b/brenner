// Package ui holds Brenner's terminal presentation: the lipgloss "ember" theme,
// the interactive huh forms, and the animated progress bar.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// The ember palette — warm oranges and reds evoking a burner at work.
var (
	ember = lipgloss.Color("#FF7A18")
	flame = lipgloss.Color("#FF3D2E")
	spark = lipgloss.Color("#FFD166")
	ash   = lipgloss.Color("#9A8C82")
	coal  = lipgloss.Color("#1A1410")
	leaf  = lipgloss.Color("#7BD88F")
	bone  = lipgloss.Color("#EDE6E1")
)

// Shared styles.
var (
	TitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(ember)
	SubtitleStyle = lipgloss.NewStyle().Foreground(ash)
	HeaderStyle   = lipgloss.NewStyle().Bold(true).Foreground(spark)
	WarnStyle     = lipgloss.NewStyle().Bold(true).Foreground(flame)
	OkStyle       = lipgloss.NewStyle().Bold(true).Foreground(leaf)
	BoxStyle      = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ember).
			Padding(0, 1)
)

// Banner returns the Brenner wordmark and tagline.
func Banner() string {
	logo := lipgloss.NewStyle().Bold(true).Foreground(ember).Render("🔥 brenner")
	tag := SubtitleStyle.Italic(true).Render("burn ISOs to USB — like music to a CD")
	return lipgloss.JoinVertical(lipgloss.Left, logo, tag)
}

// Step renders a completed-selection breadcrumb, e.g. "✓ Device  /dev/sdb", so
// the interactive flow leaves a readable history of what was chosen.
func Step(label, value string) string {
	check := OkStyle.Render("✓")
	name := SubtitleStyle.Width(7).Render(label)
	val := lipgloss.NewStyle().Foreground(bone).Render(value)
	return check + " " + name + "  " + val
}

var barTrackStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#4A3B33"))

// emberColor returns the hex colour at position t in [0,1] along the flame→spark
// ramp. It interpolates in RGB on purpose: red→yellow keeps the red channel
// maxed and ramps green up, sweeping cleanly through orange. A perceptual space
// (as bubbles' default gradient uses) bows the same endpoints through green.
func emberColor(t float64) string {
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	lerp := func(a, b int) int { return a + int(float64(b-a)*t+0.5) }
	// flame #FF3D2E → spark #FFD166
	return fmt.Sprintf("#%02X%02X%02X", lerp(0xFF, 0xFF), lerp(0x3D, 0xD1), lerp(0x2E, 0x66))
}

// EmberBar renders a width-cell progress bar filled to frac (in [0,1]). The
// filled run is shaded along the ember ramp so the bar "heats up" from red to
// yellow as it nears completion.
func EmberBar(width int, frac float64) string {
	if width < 1 {
		width = 1
	}
	switch {
	case frac < 0:
		frac = 0
	case frac > 1:
		frac = 1
	}
	filled := min(int(frac*float64(width)+0.5), width)

	var b strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			t := 0.0
			if width > 1 {
				t = float64(i) / float64(width-1)
			}
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(emberColor(t))).Render("█"))
		} else {
			b.WriteString(barTrackStyle.Render("░"))
		}
	}
	return b.String()
}

// Theme returns the huh theme used by every interactive form.
func Theme() *huh.Theme {
	t := huh.ThemeBase()
	t.Focused.Title = t.Focused.Title.Foreground(ember).Bold(true)
	t.Focused.Base = t.Focused.Base.BorderForeground(ember)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(ember)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(spark)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Background(ember).Foreground(coal)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(ember)
	return t
}
