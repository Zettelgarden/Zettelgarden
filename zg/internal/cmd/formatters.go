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

// Habit formatting

func (h Habit) FormatHuman() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Habit: %s\n", h.Title))
	sb.WriteString(fmt.Sprintf("ID: %d\n", h.ID))
	sb.WriteString(fmt.Sprintf("Frequency: %s\n", h.Frequency))
	if h.Description != nil && *h.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", *h.Description))
	}
	if h.Icon != nil && *h.Icon != "" {
		sb.WriteString(fmt.Sprintf("Icon: %s\n", *h.Icon))
	}
	if h.Color != nil && *h.Color != "" {
		sb.WriteString(fmt.Sprintf("Color: %s\n", *h.Color))
	}
	sb.WriteString(fmt.Sprintf("Created: %s\n", formatDate(h.CreatedAt)))
	return sb.String()
}

func (h Habit) FormatListHeader() string {
	return "  ID  Frequency  Title"
}

func (h Habit) FormatListItem() string {
	icon := ""
	if h.Icon != nil && *h.Icon != "" {
		icon = *h.Icon + " "
	}
	return fmt.Sprintf("%4d  %-9s  %s%s", h.ID, h.Frequency, icon, h.Title)
}

// HabitWithCheckin formatting

func (h HabitWithCheckin) FormatHuman() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Habit: %s\n", h.Title))
	sb.WriteString(fmt.Sprintf("ID: %d\n", h.ID))
	sb.WriteString(fmt.Sprintf("Frequency: %s\n", h.Frequency))
	sb.WriteString(fmt.Sprintf("Due today: %t\n", h.IsDueToday))
	sb.WriteString(fmt.Sprintf("Checked in: %t\n", h.CheckedInToday))
	if h.Description != nil && *h.Description != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", *h.Description))
	}
	sb.WriteString(fmt.Sprintf("Created: %s\n", formatDate(h.CreatedAt)))
	return sb.String()
}

func (h HabitWithCheckin) FormatListHeader() string {
	return "  ID  Done  Title"
}

func (h HabitWithCheckin) FormatListItem() string {
	done := "[ ]"
	if h.CheckedInToday {
		done = "[x]"
	} else if !h.IsDueToday {
		done = " - " // Not due today
	}
	icon := ""
	if h.Icon != nil && *h.Icon != "" {
		icon = *h.Icon + " "
	}
	return fmt.Sprintf("%4d  %s  %s%s", h.ID, done, icon, h.Title)
}

// HabitStats formatting

func (h HabitStats) FormatHuman() string {
	var sb strings.Builder
	sb.WriteString("Habit Statistics\n")
	sb.WriteString("----------------\n")
	sb.WriteString(fmt.Sprintf("Current streak: %d days\n", h.CurrentStreak))
	sb.WriteString(fmt.Sprintf("Longest streak: %d days\n", h.LongestStreak))
	sb.WriteString(fmt.Sprintf("Total completions: %d\n", h.TotalCompletions))
	sb.WriteString(fmt.Sprintf("7-day completion rate: %.0f%%\n", h.CompletionRate7d*100))
	sb.WriteString(fmt.Sprintf("30-day completion rate: %.0f%%\n", h.CompletionRate30d*100))
	if h.LastCompletedAt != nil && *h.LastCompletedAt != "" {
		sb.WriteString(fmt.Sprintf("Last completed: %s\n", formatDate(*h.LastCompletedAt)))
	}
	return sb.String()
}

// HabitLog formatting

func (h HabitLog) FormatHuman() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Log #%d\n", h.ID))
	sb.WriteString(fmt.Sprintf("Completed: %s\n", formatDateTime(h.CompletedAt)))
	if h.Notes != nil && *h.Notes != "" {
		sb.WriteString(fmt.Sprintf("Notes: %s\n", *h.Notes))
	}
	return sb.String()
}

func (h HabitLog) FormatListHeader() string {
	return "  Log ID  Completed            Notes"
}

func (h HabitLog) FormatListItem() string {
	notes := "-"
	if h.Notes != nil && *h.Notes != "" {
		if len(*h.Notes) > 30 {
			notes = (*h.Notes)[:27] + "..."
		} else {
			notes = *h.Notes
		}
	}
	return fmt.Sprintf("%7d  %s  %s", h.ID, formatDateTime(h.CompletedAt), notes)
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
