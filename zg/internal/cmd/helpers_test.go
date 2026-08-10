package cmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/nick-zettelgarden/zg/internal/config"
)

// writeTestConfig writes a temp config pointing at the given API URL and
// points the CLI at it via SetCfgFile. It returns a cleanup func that resets
// the CLI config overrides so tests don't leak state into each other.
func writeTestConfig(t *testing.T, apiURL string) {
	t.Helper()
	t.Setenv(config.EnvNoKeyring, "1") // keep token resolution deterministic
	t.Setenv(config.EnvToken, "")      // neutralize any ambient ZETTELGARDEN_TOKEN
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	content := `{"api_url": "` + apiURL + `", "token": "test-token"}`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	SetCfgFile(configPath)
	t.Cleanup(func() {
		SetCfgFile("")
		SetAPIURL("")
		SetAPIToken("")
	})
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns
// everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// newCardListCmd builds a fresh `card list` command with the same flag
// bindings as init(), so tests can set flags without cross-test leakage.
func newCardListCmd() *cobra.Command {
	c := &cobra.Command{Use: "list"}
	c.Flags().IntVarP(&listLimit, "limit", "l", 20, "Limit results")
	c.Flags().IntVarP(&listOffset, "offset", "o", 0, "Offset results")
	c.Flags().BoolVar(&listStarred, "starred", false, "Show only starred cards")
	c.Flags().BoolVar(&listFull, "full", false, "Show full body content")
	return c
}

// newTaskListCmd builds a fresh `task list` command mirroring init().
func newTaskListCmd() *cobra.Command {
	c := &cobra.Command{Use: "list"}
	c.Flags().IntVarP(&taskListLimit, "limit", "l", 50, "Limit results")
	c.Flags().BoolVar(&taskListCompleted, "completed", false, "Show only completed tasks")
	c.Flags().BoolVar(&taskListIncomplete, "incomplete", false, "Show only incomplete tasks")
	c.Flags().StringVarP(&taskListPriority, "priority", "p", "", "Filter by priority")
	c.Flags().StringVar(&taskListScheduled, "scheduled-date", "", "Filter by scheduled date")
	c.Flags().StringVar(&taskListStatus, "status", "", "Filter by status")
	return c
}

// newTaskUpdateCmd builds a fresh `task update` command mirroring init().
func newTaskUpdateCmd() *cobra.Command {
	c := &cobra.Command{Use: "update"}
	c.Flags().StringVarP(&taskUpdateTitle, "title", "t", "", "New title")
	c.Flags().StringVarP(&taskUpdateDescription, "description", "d", "", "New description")
	c.Flags().BoolVar(&taskUpdateComplete, "complete", false, "Mark as complete")
	c.Flags().BoolVar(&taskUpdateIncomplete, "incomplete", false, "Mark as incomplete")
	c.Flags().StringVarP(&taskUpdatePriority, "priority", "p", "", "New priority")
	c.Flags().StringVar(&taskUpdateScheduled, "scheduled-date", "", "New scheduled date")
	c.Flags().StringVar(&taskUpdateStatus, "status", "", "New status")
	return c
}
