package cmd

import (
	"fmt"

	"github.com/saurabh12nxf/registry-mirror/internal/mirror"
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

	syncer := mirror.NewSyncer(registry, parallel)

	if err := syncer.Sync(image, force); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Printf("✅ Successfully synced %s\n", image)
	return nil
}
