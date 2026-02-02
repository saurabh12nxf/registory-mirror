package cmd

import (
	"fmt"

	"github.com/saurabh12nxf/registry-mirror/internal/cache"
	"github.com/saurabh12nxf/registry-mirror/internal/mirror"
	"github.com/saurabh12nxf/registry-mirror/internal/storage"
	"github.com/spf13/cobra"
)

var autoCmd = &cobra.Command{
	Use:   "auto",
	Short: "Auto-mirror popular images",
	Long:  `Automatically detects and mirrors popular images that are missing from your local registry.`,
	RunE:  runAuto,
}

func init() {
	rootCmd.AddCommand(autoCmd)
	autoCmd.Flags().IntP("top", "t", 5, "number of top images to mirror")
	autoCmd.Flags().BoolP("dry-run", "d", false, "show what would be mirrored without acting")
}

func runAuto(cmd *cobra.Command, args []string) error {
	top, _ := cmd.Flags().GetInt("top")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	registry, _ := cmd.Flags().GetString("registry")

	db, err := storage.NewDB()
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}
	defer db.Close()

	fmt.Printf("🔮 Analyzing usage patterns to predict top %d images...\n", top)

	predictor := cache.NewPredictor(db)
	suggestions, err := predictor.PredictTopImages(top)
	if err != nil {
		return err
	}

	if len(suggestions) == 0 {
		fmt.Println("✨ Your registry is up to date! No new popular images found to mirror.")
		return nil
	}

	fmt.Println("📋 Proposed Auto-Mirror List:")
	for _, img := range suggestions {
		fmt.Printf("   - %s\n", img.Name)
	}

	if dryRun {
		fmt.Println("\nDry run completed. No actions taken.")
		return nil
	}

	fmt.Println("\n🚀 Starting auto-mirror process...")
	syncer := mirror.NewSyncer(registry, 3)
	tracker := mirror.NewTracker(db)

	for i, img := range suggestions {
		fmt.Printf("[%d/%d] Mirroring %s...\n", i+1, len(suggestions), img.Name)

		err := syncer.Sync(img.Name, false)
		if err != nil {
			fmt.Printf("❌ Failed to sync %s: %v\n", img.Name, err)
			tracker.TrackSyncError(img.Name, err)
			continue
		}
		tracker.TrackSyncComplete(img.Name, 0, 0) // simplified stats
	}

	fmt.Println("\n✅ Auto-mirror completed successfully!")
	return nil
}
