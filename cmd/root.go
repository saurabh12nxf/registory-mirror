package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "registry-mirror",
	Short: "A smart Docker registry mirroring tool",
	Long: `registry-mirror intelligently mirrors and caches container images 
from Docker Hub to your local registry, dramatically speeding up image pulls.

Born from frustration with slow Docker image pulls on home networks,
this tool reduces pull times from minutes to seconds by maintaining
a smart local cache of frequently-used images.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.registry-mirror.yaml)")
	rootCmd.PersistentFlags().Bool("json", false, "output in JSON format")
	rootCmd.PersistentFlags().StringP("registry", "r", "localhost:5000", "local registry address")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".registry-mirror")
	}

	viper.SetEnvPrefix("REGISTRY_MIRROR")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		// Config file found and loaded successfully
		// We don't print this to avoid cluttering output
	}
}
