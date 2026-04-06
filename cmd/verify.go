package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/saurabh12nxf/registry-mirror/internal/security"
	"github.com/spf13/cobra"
)

var (
	requireSig  bool
	trustedKeys []string
)

var verifyCmd = &cobra.Command{
	Use:   "verify <image>",
	Short: "Verify image signature using Cosign",
	Long: `Verify the cryptographic signature of a container image using Cosign/Sigstore.
This ensures the image comes from a trusted source and hasn't been tampered with.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]

		fmt.Printf("🔐 Verifying signature for: %s\n\n", image)

		// Create verifier
		verifier := security.NewVerifier(requireSig, trustedKeys)

		// Verify image
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := verifier.VerifyImage(ctx, image)

		// Display results
		if result.Verified {
			fmt.Println("✅ Signature Verification: PASSED")
			fmt.Printf("   Image: %s\n", result.Image)
			fmt.Printf("   Signatures Found: %d\n\n", len(result.Signatures))

			if len(result.Signatures) > 0 {
				fmt.Println("📝 Signature Details:")
				for i, sig := range result.Signatures {
					fmt.Printf("\n   Signature %d:\n", i+1)
					fmt.Printf("   ├─ Issuer: %s\n", sig.Issuer)
					fmt.Printf("   ├─ Subject: %s\n", sig.Subject)
					fmt.Printf("   └─ Timestamp: %s\n", sig.Timestamp)
				}
			}

			fmt.Println("\n✅ Image is safe to mirror")
		} else {
			fmt.Println("❌ Signature Verification: FAILED")
			fmt.Printf("   Image: %s\n", result.Image)
			if err != nil {
				fmt.Printf("   Error: %s\n", err)
			}
			fmt.Println("\n⚠️  Warning: Image may not be from a trusted source")
		}

		if err != nil && requireSig {
			return fmt.Errorf("verification failed: %w", err)
		}

		return nil
	},
}

var verifyBatchCmd = &cobra.Command{
	Use:   "verify-batch",
	Short: "Verify signatures for multiple images",
	Long:  `Verify cryptographic signatures for multiple images in parallel.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("no images specified")
		}

		fmt.Printf("🔐 Verifying %d images...\n\n", len(args))

		verifier := security.NewVerifier(requireSig, trustedKeys)

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		results, err := verifier.VerifyBatch(ctx, args)

		// Display results
		verified := 0
		failed := 0

		for _, result := range results {
			if result.Verified {
				verified++
				fmt.Printf("✅ %s - VERIFIED\n", result.Image)
			} else {
				failed++
				fmt.Printf("❌ %s - FAILED", result.Image)
				if result.Error != nil {
					fmt.Printf(" (%s)", result.Error)
				}
				fmt.Println()
			}
		}

		fmt.Printf("\n📊 Summary:\n")
		fmt.Printf("   Total: %d\n", len(args))
		fmt.Printf("   Verified: %d\n", verified)
		fmt.Printf("   Failed: %d\n", failed)

		if err != nil && requireSig {
			return err
		}

		return nil
	},
}

var policyCheckCmd = &cobra.Command{
	Use:   "policy-check <image>",
	Short: "Check if image meets signature policy",
	Long:  `Verify if an image meets the configured signature policy requirements.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]

		fmt.Printf("📋 Checking policy for: %s\n\n", image)

		verifier := security.NewVerifier(requireSig, trustedKeys)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, _ := verifier.VerifyImage(ctx, image)

		// Define policy
		policy := &security.Policy{
			RequireSignature: requireSig,
			TrustedIssuers:   []string{"example-issuer"},
			MinSignatures:    1,
		}

		// Check policy
		err := policy.CheckPolicy(result)
		if err != nil {
			fmt.Printf("❌ Policy Check: FAILED\n")
			fmt.Printf("   Reason: %s\n", err)
			return err
		}

		fmt.Println("✅ Policy Check: PASSED")
		fmt.Println("   Image meets all policy requirements")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(verifyBatchCmd)
	rootCmd.AddCommand(policyCheckCmd)

	// Flags for verify command
	verifyCmd.Flags().BoolVar(&requireSig, "require-signature", false, "Require valid signature")
	verifyCmd.Flags().StringSliceVar(&trustedKeys, "trusted-keys", []string{}, "Trusted public keys")

	// Flags for verify-batch command
	verifyBatchCmd.Flags().BoolVar(&requireSig, "require-signature", false, "Require valid signature")
	verifyBatchCmd.Flags().StringSliceVar(&trustedKeys, "trusted-keys", []string{}, "Trusted public keys")

	// Flags for policy-check command
	policyCheckCmd.Flags().BoolVar(&requireSig, "require-signature", true, "Require valid signature")
}
