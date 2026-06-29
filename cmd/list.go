package cmd

import (
	"fmt"

	"github.com/fx64b/brenner/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List removable devices",
	Long:    "List removable devices as a table. Does not require root.",
	RunE:    runList,
}

func runList(cmd *cobra.Command, _ []string) error {
	enum := newEnumerator()
	devices, err := enum.ListRemovable()
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if len(devices) == 0 {
		fmt.Fprintln(out, ui.SubtitleStyle.Render("No removable devices found."))
		return nil
	}
	fmt.Fprintln(out, ui.RenderDeviceTable(devices))
	return nil
}
