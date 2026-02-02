package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/saurabh12nxf/registry-mirror/internal/storage"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status and recent activity",
	Long:  `Displays the recent synchronization activity and status of mirrored images.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().IntP("limit", "n", 10, "number of recent entries to show")
}

func runStatus(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")

	db, err := storage.NewDB()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	records, err := db.GetRecentSyncs(limit)
	if err != nil {
		return fmt.Errorf("failed to fetch status: %w", err)
	}

	if len(records) == 0 {
		fmt.Println("No sync activity recorded yet.")
		return nil
	}

	fmt.Printf("🔍 Recent Sync Activity (Last %d)\n\n", limit)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "IMAGE\tSTATUS\tSIZE\tDURATION\tTIME")

	for _, r := range records {
		timeAgo := time.Since(r.Timestamp).Round(time.Second)
		sizeMB := float64(r.Bytes) / (1024 * 1024)

		statusIcon := "✅"
		if r.Status != "completed" {
			statusIcon = "❌"
		}

		fmt.Fprintf(w, "%s\t%s %s\t%.1f MB\t%.2fs\t%s ago\n",
			r.Image,
			statusIcon, r.Status,
			sizeMB,
			r.Duration,
			timeAgo)
	}
	w.Flush()

	return nil
}
