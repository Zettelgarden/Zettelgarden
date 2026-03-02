package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/nick-zettelgarden/zg/internal/cmd"
)

var rootCmd = &cobra.Command{
	Use:   "zg",
	Short: "Zettelgarden CLI tool",
	Long:  `A standalone CLI tool for Zettelgarden card and task operations.`,
}

func main() {
	// Add card commands
	rootCmd.AddCommand(cmd.GetCardCmd())
	// Add task commands
	rootCmd.AddCommand(cmd.GetTaskCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
