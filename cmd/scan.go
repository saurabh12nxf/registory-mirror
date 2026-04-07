package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/saurabh12nxf/registry-mirror/internal/git"
)

var (
	scanPath   string
	autoMirror bool
	bomOutput  string
)

var scanCmd = &cobra.Command{
	Use:   "scan-repo [path]",
	Short: "Scan Git repository for container images",
	Long: `Scan a Git repository to find all container image references in:
- Dockerfiles
- Kubernetes manifests (YAML)
- Docker Compose files

Optionally auto-mirror all discovered images.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine scan path
		path := scanPath
		if len(args) > 0 {
			path = args[0]
		}
		if path == "" {
			path = "." // Current directory
		}

		// Convert to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		// Check if path exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", absPath)
		}

		fmt.Printf("🔍 Scanning repository: %s\n\n", absPath)

		// Create scanner
		scanner := git.NewScanner(absPath)

		// Scan for images
		images, err := scanner.ScanRepository()
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		if len(images) == 0 {
			fmt.Println("No container images found in repository.")
			return nil
		}

		// Display results
		fmt.Printf("📦 Found %d unique container images:\n\n", len(images))

		// Group by source
		bySource := make(map[string][]git.ImageReference)
		for _, img := range images {
			bySource[img.Source] = append(bySource[img.Source], img)
		}

		for source, refs := range bySource {
			fmt.Printf("  %s (%d images):\n", source, len(refs))
			for _, ref := range refs {
				relPath, _ := filepath.Rel(absPath, ref.FilePath)
				fmt.Printf("    • %s\n", ref.Image)
				fmt.Printf("      └─ %s:%d\n", relPath, ref.LineNumber)
			}
			fmt.Println()
		}

		// Generate BOM if requested
		if bomOutput != "" {
			bom, err := scanner.GenerateBillOfMaterials()
			if err != nil {
				return fmt.Errorf("failed to generate BOM: %w", err)
			}

			err = os.WriteFile(bomOutput, []byte(bom), 0644)
			if err != nil {
				return fmt.Errorf("failed to write BOM: %w", err)
			}

			fmt.Printf("📄 Bill of Materials written to: %s\n\n", bomOutput)
		}

		// Auto-mirror if requested
		if autoMirror {
			fmt.Println("🔄 Auto-mirroring discovered images...")

			for _, img := range images {
				fmt.Printf("Syncing %s...\n", img.Image)
				// Note: In a real implementation, you would call the sync function here
				// For now, we'll just print what would be synced
			}

			fmt.Printf("\n✅ Would mirror %d images (use 'sync' command to actually mirror)\n", len(images))
			fmt.Println("\nTo mirror these images, run:")
			for _, img := range images {
				fmt.Printf("  registry-mirror sync %s\n", img.Image)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().StringVarP(&scanPath, "path", "p", "", "Path to repository (default: current directory)")
	scanCmd.Flags().BoolVarP(&autoMirror, "auto-mirror", "a", false, "Automatically mirror all discovered images")
	scanCmd.Flags().StringVarP(&bomOutput, "bom", "b", "", "Generate Bill of Materials and save to file")
}
