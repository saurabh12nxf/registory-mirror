package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/saurabh12nxf/registry-mirror/internal/cache"
	"github.com/saurabh12nxf/registry-mirror/internal/mirror"
	"github.com/saurabh12nxf/registry-mirror/internal/security"
	"github.com/saurabh12nxf/registry-mirror/internal/storage"
)

var (
	verifySig bool
)

var syncCmd = &cobra.Command{
	Use:   "sync <image>",
	Short: "Mirror a specific image to local registry",
	Long: `Sync pulls an image from Docker Hub and pushes it to your local registry.
	
Examples:
  registry-mirror sync nginx:latest
  registry-mirror sync tensorflow/tensorflow:2.11.0
  registry-mirror sync postgres:15-alpine
  registry-mirror sync nginx:latest --verify-signature`,
	Args: cobra.ExactArgs(1),
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().BoolP("force", "f", false, "force re-sync even if image exists")
	syncCmd.Flags().Int("parallel", 3, "number of parallel layer downloads")
	syncCmd.Flags().BoolVar(&verifySig, "verify-signature", false, "verify image signature before syncing")
}

func runSync(cmd *cobra.Command, args []string) error {
	image := args[0]
	force, _ := cmd.Flags().GetBool("force")
	parallel, _ := cmd.Flags().GetInt("parallel")
	registry, _ := cmd.Flags().GetString("registry")

	// Verify signature if requested
	if verifySig {
		fmt.Printf("🔐 Verifying signature for %s...\n", image)
		verifier := security.NewVerifier(true, []string{})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := verifier.VerifyImage(ctx, image)
		if err != nil || !result.Verified {
			return fmt.Errorf("signature verification failed: image not signed or signature invalid")
		}
		fmt.Println("✅ Signature verified")
	}

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
