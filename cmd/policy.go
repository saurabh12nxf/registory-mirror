package cmd

import (
	"fmt"
	"time"

	"github.com/saurabh12nxf/registry-mirror/internal/storage"
	"github.com/spf13/cobra"
)

var (
	ttlFlag       string
	expiresInFlag string
	reasonFlag    string
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage time-based cache policies",
	Long:  `Set, view, and manage TTL-based cache policies for images.`,
}

var setPolicyCmd = &cobra.Command{
	Use:   "set <image>",
	Short: "Set a TTL policy for an image",
	Long:  `Set a time-to-live policy for a cached image. The image will be automatically removed after the TTL expires.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]
		
		// Parse TTL
		ttlDuration, err := parseDuration(ttlFlag)
		if err != nil {
			return fmt.Errorf("invalid TTL format: %w", err)
		}

		db, err := storage.NewDB()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		ttlSeconds := int64(ttlDuration.Seconds())
		if err := db.SetCachePolicy(image, ttlSeconds, reasonFlag); err != nil {
			return fmt.Errorf("failed to set policy: %w", err)
		}

		expiresAt := time.Now().Add(ttlDuration)
		fmt.Printf("✅ Policy set for %s\n", image)
		fmt.Printf("   TTL: %s\n", ttlDuration)
		fmt.Printf("   Expires: %s\n", expiresAt.Format(time.RFC3339))
		if reasonFlag != "" {
			fmt.Printf("   Reason: %s\n", reasonFlag)
		}

		return nil
	},
}

var listPoliciesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active cache policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.NewDB()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		policies, err := db.GetAllActivePolicies()
		if err != nil {
			return fmt.Errorf("failed to get policies: %w", err)
		}

		if len(policies) == 0 {
			fmt.Println("No active policies")
			return nil
		}

		fmt.Println("📋 Active Cache Policies")
		fmt.Println("========================")
		for _, p := range policies {
			timeLeft := time.Until(p.ExpiresAt)
			fmt.Printf("\n%s\n", p.Image)
			fmt.Printf("  TTL: %s\n", formatDuration(time.Duration(p.TTL)*time.Second))
			fmt.Printf("  Expires in: %s\n", formatDuration(timeLeft))
			fmt.Printf("  Expires at: %s\n", p.ExpiresAt.Format(time.RFC3339))
			if p.Reason != "" {
				fmt.Printf("  Reason: %s\n", p.Reason)
			}
		}

		return nil
	},
}

var revokePolicyCmd = &cobra.Command{
	Use:   "revoke <image>",
	Short: "Revoke a cache policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]

		db, err := storage.NewDB()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		if err := db.RevokeCachePolicy(image, reasonFlag); err != nil {
			return fmt.Errorf("failed to revoke policy: %w", err)
		}

		fmt.Printf("✅ Policy revoked for %s\n", image)
		if reasonFlag != "" {
			fmt.Printf("   Reason: %s\n", reasonFlag)
		}

		return nil
	},
}

var cleanupPoliciesCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up expired policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.NewDB()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		count, err := db.CleanupExpiredPolicies()
		if err != nil {
			return fmt.Errorf("failed to cleanup policies: %w", err)
		}

		fmt.Printf("✅ Cleaned up %d expired policies\n", count)
		return nil
	},
}

var auditLogCmd = &cobra.Command{
	Use:   "audit",
	Short: "View policy audit log",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := storage.NewDB()
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		logs, err := db.GetPolicyAuditLog(50)
		if err != nil {
			return fmt.Errorf("failed to get audit log: %w", err)
		}

		if len(logs) == 0 {
			fmt.Println("No audit log entries")
			return nil
		}

		fmt.Println("📜 Policy Audit Log")
		fmt.Println("===================")
		for _, log := range logs {
			actionIcon := "📝"
			switch log.Action {
			case "created":
				actionIcon = "✅"
			case "expired":
				actionIcon = "⏰"
			case "revoked":
				actionIcon = "❌"
			}

			fmt.Printf("\n%s %s - %s\n", actionIcon, log.Action, log.Image)
			fmt.Printf("  Time: %s\n", log.Timestamp.Format(time.RFC3339))
			if log.TTL > 0 {
				fmt.Printf("  TTL: %s\n", formatDuration(time.Duration(log.TTL)*time.Second))
			}
			if log.Reason != "" {
				fmt.Printf("  Reason: %s\n", log.Reason)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(policyCmd)

	// Set policy command
	setPolicyCmd.Flags().StringVar(&ttlFlag, "ttl", "24h", "Time to live (e.g., 1h, 24h, 7d)")
	setPolicyCmd.Flags().StringVar(&reasonFlag, "reason", "", "Reason for the policy")
	policyCmd.AddCommand(setPolicyCmd)

	// Revoke policy command
	revokePolicyCmd.Flags().StringVar(&reasonFlag, "reason", "", "Reason for revocation")
	policyCmd.AddCommand(revokePolicyCmd)

	// Other subcommands
	policyCmd.AddCommand(listPoliciesCmd)
	policyCmd.AddCommand(cleanupPoliciesCmd)
	policyCmd.AddCommand(auditLogCmd)
}

// parseDuration parses duration strings like "1h", "24h", "7d"
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	// Handle days
	if s[len(s)-1] == 'd' {
		days := s[:len(s)-1]
		d, err := time.ParseDuration(days + "h")
		if err != nil {
			return 0, err
		}
		return d * 24, nil
	}

	return time.ParseDuration(s)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "expired"
	}

	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
