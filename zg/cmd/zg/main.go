package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/nick-zettelgarden/zg/internal/cmd"
	"github.com/nick-zettelgarden/zg/internal/output"
)

var (
	cfgFile  string
	apiURL   string
	apiToken string
	pretty   bool
)

var rootCmd = &cobra.Command{
	Use:   "zg",
	Short: "Zettelgarden CLI tool",
	Long:  `A standalone CLI tool for Zettelgarden card and task operations.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if pretty {
			output.SetPretty(true)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path")
	rootCmd.PersistentFlags().StringVar(&apiURL, "url", "", "Override API URL")
	rootCmd.PersistentFlags().StringVar(&apiToken, "token", "", "Override auth token")
	rootCmd.PersistentFlags().BoolVar(&pretty, "pretty", false, "Pretty-print JSON output")
}

func main() {
	rootCmd.AddCommand(cmd.GetCardCmd())
	rootCmd.AddCommand(cmd.GetTaskCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
