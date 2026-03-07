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
	PersistentPreRun: func(command *cobra.Command, args []string) {
		if pretty {
			output.SetPretty(true)
		}
		// Pass global flag values to cmd package
		cmd.SetCfgFile(cfgFile)
		cmd.SetAPIURL(apiURL)
		cmd.SetAPIToken(apiToken)
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
	rootCmd.AddCommand(cmd.GetTemplateCmd())
	rootCmd.AddCommand(cmd.GetHabitCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
