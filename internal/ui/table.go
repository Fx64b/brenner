package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/fx64b/brenner/internal/device"
)

// RenderDeviceTable renders the removable devices as a bordered, colourised
// table suitable for `brenner list`.
func RenderDeviceTable(devices []device.Device) string {
	cell := lipgloss.NewStyle().Foreground(bone).Padding(0, 1)
	header := HeaderStyle.Padding(0, 1)

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(ember)).
		Headers("DEVICE", "SIZE", "MODEL", "LABEL").
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return header
			}
			return cell
		})

	for _, d := range devices {
		label := d.Label
		if label == "" {
			label = "—"
		}
		t.Row(d.Path, device.HumanSize(d.Size), d.Title(), label)
	}
	return t.Render()
}
