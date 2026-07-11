package cmd

import (
	"fmt"
	"strings"
	"time"
)

// Card formatting

func (c Card) FormatHuman() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Card: %s\n", c.Title))
	sb.WriteString(fmt.Sprintf("ID: %s (#%d)\n", c.CardID, c.ID))
	sb.WriteString(fmt.Sprintf("Created: %s\n", formatDate(c.CreatedAt)))
	if c.UpdatedAt != "" && c.UpdatedAt != c.CreatedAt {
		sb.WriteString(fmt.Sprintf("Updated: %s\n", formatDate(c.UpdatedAt)))
	}
	if c.Link != "" {
		sb.WriteString(fmt.Sprintf("Link: %s\n", c.Link))
	}
	if c.Body != "" {
		sb.WriteString("\n")
		// Indent body for readability
		for _, line := range strings.Split(c.Body, "\n") {
			sb.WriteString("  ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (c Card) FormatListHeader() string {
	return "  ID     Title"
}

func (c Card) FormatListItem() string {
	return fmt.Sprintf("%6s  %s", c.CardID, c.Title)
}

// Task formatting

func (t Task) FormatHuman() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Task: %s\n", t.Title))
	sb.WriteString(fmt.Sprintf("ID: %d\n", t.ID))

	status := "incomplete"
	if t.IsComplete {
		status = "complete"
	}
	sb.WriteString(fmt.Sprintf("Status: %s\n", status))

	if t.Priority != nil && *t.Priority != "" {
		sb.WriteString(fmt.Sprintf("Priority: %s\n", *t.Priority))
	}
	if t.Status != nil && *t.Status != "" {
		sb.WriteString(fmt.Sprintf("Workflow: %s\n", *t.Status))
	}
	if t.ScheduledAt != nil && *t.ScheduledAt != "" {
		sb.WriteString(fmt.Sprintf("Scheduled: %s\n", formatDate(*t.ScheduledAt)))
	}
	sb.WriteString(fmt.Sprintf("Created: %s\n", formatDate(t.CreatedAt)))

	if t.Description != "" {
		sb.WriteString("\n")
		for _, line := range strings.Split(t.Description, "\n") {
			sb.WriteString("  ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (t Task) FormatListHeader() string {
	return "  ID  Status  Priority  Scheduled   Title"
}

func (t Task) FormatListItem() string {
	status := "[ ]"
	if t.IsComplete {
		status = "[x]"
	}
	priority := "-"
	if t.Priority != nil && *t.Priority != "" {
		priority = *t.Priority
	}
	scheduled := "-"
	if t.ScheduledAt != nil && *t.ScheduledAt != "" {
		scheduled = formatDate(*t.ScheduledAt)
	}
	return fmt.Sprintf("%4d  %s  %-8s  %-10s  %s", t.ID, status, priority, scheduled, t.Title)
}

// CardTemplate formatting

func (t CardTemplate) FormatHuman() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Template: %s\n", t.Name))
	sb.WriteString(fmt.Sprintf("ID: %d\n", t.ID))
	if t.Title != "" {
		sb.WriteString(fmt.Sprintf("Title template: %s\n", t.Title))
	}
	sb.WriteString(fmt.Sprintf("Created: %s\n", formatDate(t.CreatedAt)))
	if t.Body != "" {
		sb.WriteString("\nBody template:\n")
		for _, line := range strings.Split(t.Body, "\n") {
			sb.WriteString("  ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (t CardTemplate) FormatListHeader() string {
	return "  ID  Name"
}

func (t CardTemplate) FormatListItem() string {
	return fmt.Sprintf("%4d  %s", t.ID, t.Name)
}

// Helper functions

func formatDate(s string) string {
	// Try to parse and reformat as date only
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Format("2006-01-02")
	}
	// Already in a reasonable format or unparseable
	if len(s) > 10 {
		return s[:10]
	}
	return s
}

func formatDateTime(s string) string {
	// Try to parse and reformat with time
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Format("2006-01-02 15:04")
	}
	// Already in a reasonable format or unparseable
	if len(s) > 16 {
		return s[:16]
	}
	return s
}
