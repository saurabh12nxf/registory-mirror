package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/saurabh12nxf/registry-mirror/internal/analytics"
	"github.com/saurabh12nxf/registry-mirror/internal/storage"
	"github.com/spf13/cobra"
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Show usage statistics and savings",
	Long:  `Displays aggregated statistics about your registry usage, including bandwidth saved and cache performance.`,
	RunE:  runAnalytics,
}

func init() {
	rootCmd.AddCommand(analyticsCmd)
}

func runAnalytics(cmd *cobra.Command, args []string) error {
	db, err := storage.NewDB()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	analyzer := analytics.NewAnalyzer(db)
	report, err := analyzer.GenerateReport()
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	fmt.Println("📊 Registry Mirror Analytics")
	fmt.Println("===========================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "unique images cached:\t%d\n", report.TotalImages)
	fmt.Fprintf(w, "total data served:\t%s\n", report.TotalBandwidth)
	fmt.Fprintf(w, "estimated time saved:\t%s\n", report.TimeSaved)
	fmt.Fprintf(w, "average throughput:\t%s\n", report.AvgSpeed)
	w.Flush()

	fmt.Println("\n💡 Tip: Run 'registry-mirror auto' to pre-fetch popular images.")
	return nil
}
