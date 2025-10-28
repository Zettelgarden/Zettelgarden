package main

import (
	"os"

	"github.com/zettelgarden/zettelgarden-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
