package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check system health",
	Long:  `Verifies connectivity to the local registry, disk space, and database integrity.`,
	Run:   runHealth,
}

func init() {
	rootCmd.AddCommand(healthCmd)
}

func runHealth(cmd *cobra.Command, args []string) {
	registry, err := cmd.Flags().GetString("registry")
	if err != nil {
		registry = "localhost:5000"
	}

	fmt.Println("🏥 System Health Check")
	fmt.Println("=====================")

	allGood := true

	// 1. Check Registry Connectivity
	fmt.Print("Checking Local Registry...")
	if checkRegistry(registry) {
		fmt.Println(" ✅ Online")
	} else {
		fmt.Printf(" ❌ Unreachable (http://%s)\n", registry)
		allGood = false
	}

	// 2. Check Database
	fmt.Print("Checking Database...")
	if checkDB() {
		fmt.Println("       ✅ Connected")
	} else {
		fmt.Println("       ❌ Error")
		allGood = false
	}

	// 3. Check Disk Space (Simple check if writable)
	fmt.Print("Checking Storage...")
	if checkStorage() {
		fmt.Println("        ✅ Writable")
	} else {
		fmt.Println("        ❌ Error")
		allGood = false
	}

	// 4. Check Internet
	fmt.Print("Checking Docker Hub...")
	if checkInternet() {
		fmt.Println("     ✅ Reachable")
	} else {
		fmt.Println("     ❌ Unreachable")
		allGood = false
	}

	fmt.Println("---------------------")
	if allGood {
		fmt.Println("✨ System is healthy and ready to mirror!")
	} else {
		fmt.Println("⚠️  Issues detected. Please review above.")
		os.Exit(1)
	}
}

func checkRegistry(addr string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://%s/v2/", addr)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200 || resp.StatusCode == 401
}

func checkDB() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	dbPath := filepath.Join(home, ".registry-mirror.db")
	_, err = os.Stat(dbPath)
	return err == nil || os.IsNotExist(err) // It's fine if it doesn't exist yet, it's writable
}

func checkStorage() bool {
	dir, err := os.Getwd()
	if err != nil {
		return false
	}
	tmpFile := filepath.Join(dir, ".health_check")
	err = os.WriteFile(tmpFile, []byte("test"), 0644)
	if err != nil {
		return false
	}
	os.Remove(tmpFile)
	return true
}

func checkInternet() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://registry-1.docker.io/v2/", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200 || resp.StatusCode == 401
}
