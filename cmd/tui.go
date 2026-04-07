package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/saurabh12nxf/registry-mirror/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI dashboard",
	Long:  `Launch an interactive terminal user interface with real-time monitoring, image management, and analytics.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		m := tui.NewModel()
		p := tea.NewProgram(m, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("failed to start TUI: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
