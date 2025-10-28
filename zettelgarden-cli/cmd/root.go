package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	profile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "zg",
	Short: "Zettelgarden CLI - Command-line interface for Zettelgarden",
	Long: `Zettelgarden CLI is a command-line tool for interacting with Zettelgarden,
designed primarily for LLM agent consumption with structured, parseable output formats.

Features:
  - JSON output optimized for minimal token usage
  - Field selection to reduce output size
  - Full REST API access with authentication
  - Multiple profile support for different environments`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/zettelgarden/config.json)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "profile to use (default is 'default')")

	// Bind flags to viper
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
			os.Exit(1)
		}

		// Search config in home directory with name ".config/zettelgarden/config.json"
		configDir := home + "/.config/zettelgarden"
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("json")
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("ZG")
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// Config file loaded successfully - no need to print for LLM-optimized output
		// Users can use --verbose flag if we add it later
	}
}
