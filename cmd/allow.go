package cmd

import (
	"fmt"
	"time"

	"github.com/saurabh12nxf/registry-mirror/internal/storage"
	"github.com/spf13/cobra"
)

var allowCmd = &cobra.Command{
	Use:   "allow <image>",
	Short: "Temporarily allow an image with time-limited access",
	Long: `Grant temporary access to mirror an image with automatic expiration.
This is useful for one-time or temporary image needs that should auto-expire.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]

		// Parse expires-in duration
		duration, err := parseDuration(expiresInFlag)
		if err != nil {
			return fmt.Errorf("invalid expires-in format: %w", err)
		}

		db, err := storage.NewDB()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		// Set temporary policy
		ttlSeconds := int64(duration.Seconds())
		reason := fmt.Sprintf("Temporary access granted for %s", duration)
		if reasonFlag != "" {
			reason = reasonFlag
		}

		if err := db.SetCachePolicy(image, ttlSeconds, reason); err != nil {
			return fmt.Errorf("failed to set temporary policy: %w", err)
		}

		expiresAt := time.Now().Add(duration)
		fmt.Printf("✅ Temporary access granted for %s\n", image)
		fmt.Printf("   Duration: %s\n", duration)
		fmt.Printf("   Expires: %s\n", expiresAt.Format(time.RFC3339))
		fmt.Printf("   Reason: %s\n", reason)
		fmt.Printf("\n💡 The image will be automatically removed after expiration.\n")
		fmt.Printf("   Run 'registry-mirror policy cleanup' to manually trigger cleanup.\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(allowCmd)
	allowCmd.Flags().StringVar(&expiresInFlag, "expires-in", "2h", "Duration before access expires (e.g., 1h, 2h, 24h)")
	allowCmd.Flags().StringVar(&reasonFlag, "reason", "", "Reason for temporary access")
}
