package main

import (
	"os"

	"github.com/saurabh12nxf/registry-mirror/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
