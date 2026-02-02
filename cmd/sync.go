package cmd

import (
	"fmt"
	"time"

	"github.com/saurabh12nxf/registry-mirror/internal/cache"
	"github.com/saurabh12nxf/registry-mirror/internal/mirror"
	"github.com/saurabh12nxf/registry-mirror/internal/storage"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync <image>",
	Short: "Mirror a specific image to local registry",
	Long: `Sync pulls an image from Docker Hub and pushes it to your local registry.
	
Examples:
  registry-mirror sync nginx:latest
  registry-mirror sync tensorflow/tensorflow:2.11.0
  registry-mirror sync postgres:15-alpine`,
	Args: cobra.ExactArgs(1),
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().BoolP("force", "f", false, "force re-sync even if image exists")
	syncCmd.Flags().Int("parallel", 3, "number of parallel layer downloads")
}

func runSync(cmd *cobra.Command, args []string) error {
	image := args[0]
	force, _ := cmd.Flags().GetBool("force")
	parallel, _ := cmd.Flags().GetInt("parallel")
	registry, _ := cmd.Flags().GetString("registry")

	fmt.Printf("🔄 Syncing %s to %s...\n", image, registry)

	// Init DB and Tracker
	db, err := storage.NewDB()
	if err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}
	defer db.Close()

	tracker := mirror.NewTracker(db)
	start := time.Now()

	syncer := mirror.NewSyncer(registry, parallel)

	err = syncer.Sync(image, force)
	duration := time.Since(start)

	if err != nil {
		tracker.TrackSyncError(image, err)
		return fmt.Errorf("sync failed: %w", err)
	}

	// Calculate total bytes (simplified, in real app we'd get this from syncer)
	tracker.TrackSyncComplete(image, 0, duration)

	fmt.Printf("✅ Successfully synced %s\n", image)

	// Check cache policy (Default: 10GB limit)
	cacheMgr := cache.NewManager(db, 10000, cache.PolicyLRU)
	if err := cacheMgr.EnforcePolicy(); err != nil {
		fmt.Printf("⚠️  Cache policy check failed: %v\n", err)
	}

	return nil
}
