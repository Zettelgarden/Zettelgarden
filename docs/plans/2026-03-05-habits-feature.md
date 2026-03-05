# Habits Feature Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a habits tracking system with daily check-ins, streak analytics, and task-adjacent linking.

**Architecture:** Separate habits domain with two new tables (habits, habit_logs). Backend provides REST API for CRUD, check-ins, and stats. Frontend adds dedicated /habits page with sidebar widget. Habits optionally link to existing tasks via foreign key.

**Tech Stack:** Go (backend), PostgreSQL with existing patterns, React/TypeScript (frontend), Vitest for testing.

---

## Task 1: Database Migration - Habits Table

**Files:**
- Create: `go-backend/schema/migrations/XXX_create_habits_table.up.sql`
- Create: `go-backend/schema/migrations/XXX_create_habits_table.down.sql`

**Step 1: Write the up migration**

Create file `go-backend/schema/migrations/XXX_create_habits_table.up.sql` (use next migration number):

```sql
CREATE TABLE habits (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    frequency VARCHAR(20) NOT NULL DEFAULT 'daily',
    custom_days JSONB,
    icon VARCHAR(50),
    color VARCHAR(7),
    position INTEGER NOT NULL DEFAULT 0,
    linked_task_id INTEGER REFERENCES tasks(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_habits_user_id ON habits(user_id);
CREATE INDEX idx_habits_position ON habits(user_id, position);
CREATE INDEX idx_habits_linked_task ON habits(linked_task_id);

-- Add habits to the enum table for audit events
INSERT INTO audit_table_types (table_name) VALUES ('habits');
```

**Step 2: Write the down migration**

Create file `go-backend/schema/migrations/XXX_create_habits_table.down.sql`:

```sql
DROP INDEX IF EXISTS idx_habits_linked_task;
DROP INDEX IF EXISTS idx_habits_position;
DROP INDEX IF EXISTS idx_habits_user_id;
DROP TABLE IF EXISTS habits;

DELETE FROM audit_table_types WHERE table_name = 'habits';
```

**Step 3: Run migration to verify it works**

Run: `cd go-backend && go run main.go migrate`
Expected: Tables created successfully, no errors

**Step 4: Commit**

```bash
git add go-backend/schema/migrations/*_create_habits_table.*.sql
git commit -m "feat(habits): add habits table migration"
```

---

## Task 2: Database Migration - Habit Logs Table

**Files:**
- Create: `go-backend/schema/migrations/XXX_create_habit_logs_table.up.sql`
- Create: `go-backend/schema/migrations/XXX_create_habit_logs_table.down.sql`

**Step 1: Write the up migration**

Create file `go-backend/schema/migrations/XXX_create_habit_logs_table.up.sql`:

```sql
CREATE TABLE habit_logs (
    id SERIAL PRIMARY KEY,
    habit_id INTEGER NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_habit_logs_habit_completed ON habit_logs(habit_id, completed_at DESC);
CREATE INDEX idx_habit_logs_user_completed ON habit_logs(user_id, completed_at DESC);
CREATE INDEX idx_habit_logs_habit_date ON habit_logs(habit_id, DATE(completed_at AT TIME ZONE 'UTC'));

-- Add habit_logs to audit events
INSERT INTO audit_table_types (table_name) VALUES ('habit_logs');
```

**Step 2: Write the down migration**

Create file `go-backend/schema/migrations/XXX_create_habit_logs_table.down.sql`:

```sql
DROP INDEX IF EXISTS idx_habit_logs_habit_date;
DROP INDEX IF EXISTS idx_habit_logs_user_completed;
DROP INDEX IF EXISTS idx_habit_logs_habit_completed;
DROP TABLE IF EXISTS habit_logs;

DELETE FROM audit_table_types WHERE table_name = 'habit_logs';
```

**Step 3: Run migration to verify it works**

Run: `cd go-backend && go run main.go migrate`
Expected: Tables created successfully

**Step 4: Commit**

```bash
git add go-backend/schema/migrations/*_create_habit_logs_table.*.sql
git commit -m "feat(habits): add habit_logs table migration"
```

---

## Task 3: Backend Models - Habit Types

**Files:**
- Create: `go-backend/models/habits.go`

**Step 1: Write the failing test**

Create file `go-backend/models/habits_test.go`:

```go
package models

import (
    "testing"
    "time"
)

func TestHabit_Frequency(t *testing.T) {
    habit := Habit{
        Frequency: "daily",
    }
    if habit.Frequency != "daily" {
        t.Errorf("expected daily, got %s", habit.Frequency)
    }
}

func TestHabitLog_IsToday(t *testing.T) {
    now := time.Now().UTC()
    log := HabitLog{
        CompletedAt: now,
    }
    // We'll implement this method later
    // For now just test the struct
    if log.CompletedAt.IsZero() {
        t.Error("completed_at should not be zero")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./models -run TestHabit`
Expected: FAIL - "cannot refer to unexported name..."

**Step 3: Write minimal implementation**

Create file `go-backend/models/habits.go`:

```go
package models

import "time"

// Frequency constants
const (
    FrequencyDaily    = "daily"
    FrequencyWeekly   = "weekly"
    FrequencyCustom   = "custom_days"
)

type Habit struct {
    ID            int        `json:"id"`
    UserID        int        `json:"user_id"`
    Title         string     `json:"title"`
    Description   *string    `json:"description"`
    Frequency     string     `json:"frequency"`
    CustomDays    *string    `json:"custom_days"` // JSON array as string
    Icon          *string    `json:"icon"`
    Color         *string    `json:"color"`
    Position      int        `json:"position"`
    LinkedTaskID  *int       `json:"linked_task_id"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`

    // Computed fields (not in DB)
    TodayCheckedIn bool   `json:"today_checked_in,omitempty"`
    CurrentStreak   int    `json:"current_streak,omitempty"`
    IsDueToday      bool   `json:"is_due_today,omitempty"`
    CheckedInToday  bool   `json:"checked_in_today,omitempty"`
    TodayLogID      *int   `json:"today_log_id,omitempty"`
}

type HabitLog struct {
    ID          int        `json:"id"`
    HabitID     int        `json:"habit_id"`
    UserID      int        `json:"user_id"`
    CompletedAt time.Time  `json:"completed_at"`
    Notes       *string    `json:"notes"`
    CreatedAt   time.Time  `json:"created_at"`
}

type HabitStats struct {
    CurrentStreak      int       `json:"current_streak"`
    LongestStreak      int       `json:"longest_streak"`
    TotalCompletions   int       `json:"total_completions"`
    CompletionRate7d   float64   `json:"completion_rate_7d"`
    CompletionRate30d  float64   `json:"completion_rate_30d"`
    LastCompletedAt    *time.Time `json:"last_completed_at"`
}

type CreateHabitParams struct {
    Title        string  `json:"title"`
    Description  *string `json:"description"`
    Frequency    string  `json:"frequency"`
    CustomDays   *string `json:"custom_days"`
    Icon         *string `json:"icon"`
    Color        *string `json:"color"`
    Position     *int    `json:"position"`
    LinkedTaskID *int    `json:"linked_task_id"`
}

type UpdateHabitParams struct {
    Title        *string `json:"title"`
    Description  *string `json:"description"`
    Frequency    *string `json:"frequency"`
    CustomDays   *string `json:"custom_days"`
    Icon         *string `json:"icon"`
    Color        *string `json:"color"`
    Position     *int    `json:"position"`
    LinkedTaskID *int    `json:"linked_task_id"`
}

type CheckinHabitParams struct {
    Notes *string `json:"notes"`
}
```

**Step 4: Run test to verify it passes**

Run: `cd go-backend && go test ./models -run TestHabit`
Expected: PASS

**Step 5: Commit**

```bash
git add go-backend/models/habits.go go-backend/models/habits_test.go
git commit -m "feat(habits): add habit model types"
```

---

## Task 4: Backend Services - Habit CRUD

**Files:**
- Create: `go-backend/services/habits.go`
- Modify: `go-backend/main.go` (add routes)

**Step 1: Write the failing test**

Add to `go-backend/services/habits_test.go`:

```go
package services

import (
    "database/sql"
    "testing"
    "time"
    "go-backend/models"
)

func TestCreateHabit(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    params := models.CreateHabitParams{
        Title:     "Test Habit",
        Frequency: models.FrequencyDaily,
    }

    id, err := CreateHabit(db, 1, params)
    if err != nil {
        t.Fatalf("failed to create habit: %v", err)
    }
    if id <= 0 {
        t.Error("expected positive habit id")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./services -run TestCreateHabit`
Expected: FAIL - "undefined: CreateHabit"

**Step 3: Write minimal implementation**

Create file `go-backend/services/habits.go`:

```go
package services

import (
    "database/sql"
    "fmt"
    "time"
    "go-backend/models"
)

func CreateHabit(db *sql.DB, userID int, params models.CreateHabitParams) (int, error) {
    var position int
    if params.Position != nil {
        position = *params.Position
    } else {
        // Get max position for user and add 1
        err := db.QueryRow("SELECT COALESCE(MAX(position), 0) + 1 FROM habits WHERE user_id = $1", userID).Scan(&position)
        if err != nil {
            position = 0
        }
    }

    query := `
        INSERT INTO habits (user_id, title, description, frequency, custom_days, icon, color, position, linked_task_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id
    `

    var id int
    err := db.QueryRow(query, userID, params.Title, params.Description, params.Frequency,
        params.CustomDays, params.Icon, params.Color, position, params.LinkedTaskID).Scan(&id)
    if err != nil {
        return 0, fmt.Errorf("failed to create habit: %w", err)
    }

    return id, nil
}

func GetHabit(db *sql.DB, userID int, habitID int) (models.Habit, error) {
    var habit models.Habit
    var description, customDays, icon, color sql.NullString
    var linkedTaskID sql.NullInt64

    query := `
        SELECT id, user_id, title, description, frequency, custom_days, icon, color, position, linked_task_id, created_at, updated_at
        FROM habits
        WHERE id = $1 AND user_id = $2
    `
    err := db.QueryRow(query, habitID, userID).Scan(
        &habit.ID, &habit.UserID, &habit.Title, &description, &habit.Frequency,
        &customDays, &icon, &color, &habit.Position, &linkedTaskID,
        &habit.CreatedAt, &habit.UpdatedAt,
    )
    if err == sql.ErrNoRows {
        return habit, fmt.Errorf("habit not found")
    }
    if err != nil {
        return habit, fmt.Errorf("failed to get habit: %w", err)
    }

    if description.Valid {
        habit.Description = &description.String
    }
    if customDays.Valid {
        habit.CustomDays = &customDays.String
    }
    if icon.Valid {
        habit.Icon = &icon.String
    }
    if color.Valid {
        habit.Color = &color.String
    }
    if linkedTaskID.Valid {
        id := int(linkedTaskID.Int64)
        habit.LinkedTaskID = &id
    }

    return habit, nil
}

func GetHabits(db *sql.DB, userID int) ([]models.Habit, error) {
    query := `
        SELECT id, user_id, title, description, frequency, custom_days, icon, color, position, linked_task_id, created_at, updated_at
        FROM habits
        WHERE user_id = $1
        ORDER BY position ASC
    `
    rows, err := db.Query(query, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get habits: %w", err)
    }
    defer rows.Close()

    var habits []models.Habit
    for rows.Next() {
        var habit models.Habit
        var description, customDays, icon, color sql.NullString
        var linkedTaskID sql.NullInt64

        err := rows.Scan(
            &habit.ID, &habit.UserID, &habit.Title, &description, &habit.Frequency,
            &customDays, &icon, &color, &habit.Position, &linkedTaskID,
            &habit.CreatedAt, &habit.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to scan habit: %w", err)
        }

        if description.Valid {
            habit.Description = &description.String
        }
        if customDays.Valid {
            habit.CustomDays = &customDays.String
        }
        if icon.Valid {
            habit.Icon = &icon.String
        }
        if color.Valid {
            habit.Color = &color.String
        }
        if linkedTaskID.Valid {
            id := int(linkedTaskID.Int64)
            habit.LinkedTaskID = &id
        }

        habits = append(habits, habit)
    }

    return habits, nil
}

func UpdateHabit(db *sql.DB, userID int, habitID int, params models.UpdateHabitParams) error {
    // Build dynamic UPDATE query based on non-nil params
    sets := []string{}
    args := []interface{}{}
    argPos := 1

    if params.Title != nil {
        sets = append(sets, fmt.Sprintf("title = $%d", argPos))
        args = append(args, *params.Title)
        argPos++
    }
    if params.Description != nil {
        sets = append(sets, fmt.Sprintf("description = $%d", argPos))
        args = append(args, *params.Description)
        argPos++
    }
    if params.Frequency != nil {
        sets = append(sets, fmt.Sprintf("frequency = $%d", argPos))
        args = append(args, *params.Frequency)
        argPos++
    }
    if params.CustomDays != nil {
        sets = append(sets, fmt.Sprintf("custom_days = $%d", argPos))
        args = append(args, *params.CustomDays)
        argPos++
    }
    if params.Icon != nil {
        sets = append(sets, fmt.Sprintf("icon = $%d", argPos))
        args = append(args, *params.Icon)
        argPos++
    }
    if params.Color != nil {
        sets = append(sets, fmt.Sprintf("color = $%d", argPos))
        args = append(args, *params.Color)
        argPos++
    }
    if params.Position != nil {
        sets = append(sets, fmt.Sprintf("position = $%d", argPos))
        args = append(args, *params.Position)
        argPos++
    }
    if params.LinkedTaskID != nil {
        sets = append(sets, fmt.Sprintf("linked_task_id = $%d", argPos))
        args = append(args, *params.LinkedTaskID)
        argPos++
    }

    if len(sets) == 0 {
        return nil
    }

    sets = append(sets, fmt.Sprintf("updated_at = $%d", argPos))
    args = append(args, time.Now())
    argPos++

    args = append(args, habitID, userID)

    query := fmt.Sprintf("UPDATE habits SET %s WHERE id = $%d AND user_id = $%d",
        joinStrings(sets, ", "), argPos, argPos+1)

    result, err := db.Exec(query, args...)
    if err != nil {
        return fmt.Errorf("failed to update habit: %w", err)
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("habit not found")
    }

    return nil
}

func DeleteHabit(db *sql.DB, userID int, habitID int) error {
    result, err := db.Exec("DELETE FROM habits WHERE id = $1 AND user_id = $2", habitID, userID)
    if err != nil {
        return fmt.Errorf("failed to delete habit: %w", err)
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("habit not found")
    }

    return nil
}

func joinStrings(strs []string, sep string) string {
    if len(strs) == 0 {
        return ""
    }
    result := strs[0]
    for i := 1; i < len(strs); i++ {
        result += sep + strs[i]
    }
    return result
}
```

**Step 4: Run test to verify it passes**

Run: `cd go-backend && go test ./services -run TestCreateHabit`
Expected: PASS

**Step 5: Commit**

```bash
git add go-backend/services/habits.go go-backend/services/habits_test.go
git commit -m "feat(habits): add habit CRUD service functions"
```

---

## Task 5: Backend Services - Habit Check-ins

**Files:**
- Modify: `go-backend/services/habits.go`

**Step 1: Write the failing test**

Add to `go-backend/services/habits_test.go`:

```go
func TestCheckinHabit(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Create a habit first
    params := models.CreateHabitParams{
        Title:     "Test Habit",
        Frequency: models.FrequencyDaily,
    }
    habitID, _ := CreateHabit(db, 1, params)

    // Check in
    checkinParams := models.CheckinHabitParams{}
    logID, err := CheckinHabit(db, 1, habitID, checkinParams, time.UTC)
    if err != nil {
        t.Fatalf("failed to check in: %v", err)
    }
    if logID <= 0 {
        t.Error("expected positive log id")
    }
}

func TestCheckinHabit_Duplicate(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    habitID, _ := CreateHabit(db, 1, models.CreateHabitParams{
        Title: "Test",
        Frequency: models.FrequencyDaily,
    })

    now := time.Now().UTC()
    CheckinHabit(db, 1, habitID, models.CheckinHabitParams{}, now)

    // Try to check in again same day
    _, err := CheckinHabit(db, 1, habitID, models.CheckinHabitParams{}, now)
    if err == nil {
        t.Error("expected error for duplicate check-in")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./services -run TestCheckinHabit`
Expected: FAIL - "undefined: CheckinHabit"

**Step 3: Write minimal implementation**

Add to `go-backend/services/habits.go`:

```go
func CheckinHabit(db *sql.DB, userID int, habitID int, params models.CheckinHabitParams, timezone string) (int, error) {
    // Get the habit to verify ownership
    habit, err := GetHabit(db, userID, habitID)
    if err != nil {
        return 0, fmt.Errorf("habit not found: %w", err)
    }

    // Check if already checked in today (in user's timezone)
    today := time.Now().UTC().Format("2006-01-02")
    var existingLogID int
    checkQuery := `
        SELECT id FROM habit_logs
        WHERE habit_id = $1 AND user_id = $2
        AND DATE(completed_at AT TIME ZONE $3) = $4
    `
    err = db.QueryRow(checkQuery, habitID, userID, timezone, today).Scan(&existingLogID)
    if err == nil {
        return 0, fmt.Errorf("already checked in today")
    }

    // Create the log entry
    query := `
        INSERT INTO habit_logs (habit_id, user_id, completed_at, notes)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `

    var logID int
    completedAt := time.Now().UTC()
    err = db.QueryRow(query, habitID, userID, completedAt, params.Notes).Scan(&logID)
    if err != nil {
        return 0, fmt.Errorf("failed to create habit log: %w", err)
    }

    return logID, nil
}

func DeleteHabitLog(db *sql.DB, userID int, habitID int, logID int) error {
    query := `
        DELETE FROM habit_logs
        WHERE id = $1 AND habit_id = $2 AND user_id = $3
    `
    result, err := db.Exec(query, logID, habitID, userID)
    if err != nil {
        return fmt.Errorf("failed to delete habit log: %w", err)
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("habit log not found")
    }

    return nil
}

func GetHabitLogs(db *sql.DB, userID int, habitID int, limit, offset int) ([]models.HabitLog, int, error) {
    // Get total count
    var total int
    err := db.QueryRow("SELECT COUNT(*) FROM habit_logs WHERE habit_id = $1 AND user_id = $2", habitID, userID).Scan(&total)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to count habit logs: %w", err)
    }

    query := `
        SELECT id, habit_id, user_id, completed_at, notes, created_at
        FROM habit_logs
        WHERE habit_id = $1 AND user_id = $2
        ORDER BY completed_at DESC
        LIMIT $3 OFFSET $4
    `
    rows, err := db.Query(query, habitID, userID, limit, offset)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to get habit logs: %w", err)
    }
    defer rows.Close()

    var logs []models.HabitLog
    for rows.Next() {
        var log models.HabitLog
        var notes sql.NullString

        err := rows.Scan(&log.ID, &log.HabitID, &log.UserID, &log.CompletedAt, &notes, &log.CreatedAt)
        if err != nil {
            return nil, 0, fmt.Errorf("failed to scan habit log: %w", err)
        }

        if notes.Valid {
            log.Notes = &notes.String
        }

        logs = append(logs, log)
    }

    return logs, total, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd go-backend && go test ./services -run TestCheckinHabit`
Expected: PASS

**Step 5: Commit**

```bash
git add go-backend/services/habits.go go-backend/services/habits_test.go
git commit -m "feat(habits): add check-in service functions"
```

---

## Task 6: Backend Services - Streak Calculation

**Files:**
- Modify: `go-backend/services/habits.go`

**Step 1: Write the failing test**

Add to `go-backend/services/habits_test.go`:

```go
func TestCalculateHabitStats(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    habitID, _ := CreateHabit(db, 1, models.CreateHabitParams{
        Title: "Test",
        Frequency: models.FrequencyDaily,
    })

    // Create 5 consecutive days of check-ins
    now := time.Now().UTC()
    for i := 4; i >= 0; i-- {
        checkinTime := now.AddDate(0, 0, -i)
        db.Exec("INSERT INTO habit_logs (habit_id, user_id, completed_at) VALUES ($1, $2, $3)",
            habitID, 1, checkinTime)
    }

    stats, err := CalculateHabitStats(db, 1, habitID, time.UTC)
    if err != nil {
        t.Fatalf("failed to calculate stats: %v", err)
    }
    if stats.CurrentStreak != 5 {
        t.Errorf("expected streak 5, got %d", stats.CurrentStreak)
    }
    if stats.TotalCompletions != 5 {
        t.Errorf("expected total 5, got %d", stats.TotalCompletions)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./services -run TestCalculateHabitStats`
Expected: FAIL - "undefined: CalculateHabitStats"

**Step 3: Write minimal implementation**

Add to `go-backend/services/habits.go`:

```go
func CalculateHabitStats(db *sql.DB, userID int, habitID int, timezone string) (models.HabitStats, error) {
    var stats models.HabitStats

    // Get total completions
    err := db.QueryRow("SELECT COUNT(*) FROM habit_logs WHERE habit_id = $1 AND user_id = $2", habitID, userID).Scan(&stats.TotalCompletions)
    if err != nil {
        return stats, fmt.Errorf("failed to count completions: %w", err)
    }

    // Get last completion time
    var lastCompleted sql.NullTime
    err = db.QueryRow("SELECT MAX(completed_at) FROM habit_logs WHERE habit_id = $1 AND user_id = $2", habitID, userID).Scan(&lastCompleted)
    if err != nil {
        return stats, fmt.Errorf("failed to get last completion: %w", err)
    }
    if lastCompleted.Valid {
        stats.LastCompletedAt = &lastCompleted.Time
    }

    // Calculate current streak
    stats.CurrentStreak = calculateStreak(db, userID, habitID, timezone)

    // Calculate longest streak
    stats.LongestStreak = calculateLongestStreak(db, userID, habitID, timezone)

    // Calculate completion rates
    stats.CompletionRate7d = calculateCompletionRate(db, userID, habitID, 7, timezone)
    stats.CompletionRate30d = calculateCompletionRate(db, userID, habitID, 30, timezone)

    return stats, nil
}

func calculateStreak(db *sql.DB, userID int, habitID int, timezone string) int {
    query := `
        SELECT DISTINCT DATE(completed_at AT TIME ZONE $1) as completion_date
        FROM habit_logs
        WHERE habit_id = $2 AND user_id = $3
        ORDER BY completion_date DESC
    `
    rows, err := db.Query(query, timezone, habitID, userID)
    if err != nil {
        return 0
    }
    defer rows.Close()

    var dates []time.Time
    for rows.Next() {
        var date time.Time
        if rows.Scan(&date) == nil {
            dates = append(dates, date)
        }
    }

    if len(dates) == 0 {
        return 0
    }

    streak := 1
    for i := 0; i < len(dates)-1; i++ {
        diff := dates[i].Sub(dates[i+1]).Hours() / 24
        if diff <= 1.0 { // Within 1 day
            streak++
        } else {
            break
        }
    }

    return streak
}

func calculateLongestStreak(db *sql.DB, userID int, habitID int, timezone string) int {
    query := `
        SELECT DISTINCT DATE(completed_at AT TIME ZONE $1) as completion_date
        FROM habit_logs
        WHERE habit_id = $2 AND user_id = $3
        ORDER BY completion_date ASC
    `
    rows, err := db.Query(query, timezone, habitID, userID)
    if err != nil {
        return 0
    }
    defer rows.Close()

    var dates []time.Time
    for rows.Next() {
        var date time.Time
        if rows.Scan(&date) == nil {
            dates = append(dates, date)
        }
    }

    if len(dates) == 0 {
        return 0
    }

    longestStreak := 1
    currentStreak := 1

    for i := 1; i < len(dates); i++ {
        diff := dates[i].Sub(dates[i-1]).Hours() / 24
        if diff <= 1.0 {
            currentStreak++
            if currentStreak > longestStreak {
                longestStreak = currentStreak
            }
        } else {
            currentStreak = 1
        }
    }

    return longestStreak
}

func calculateCompletionRate(db *sql.DB, userID int, habitID int, days int, timezone string) float64 {
    query := `
        SELECT COUNT(DISTINCT DATE(completed_at AT TIME ZONE $1))
        FROM habit_logs
        WHERE habit_id = $2 AND user_id = $3
        AND completed_at >= NOW() - INTERVAL '1 day' * $4
    `
    var completedDays int
    err := db.QueryRow(query, timezone, habitID, userID, days).Scan(&completedDays)
    if err != nil {
        return 0
    }

    return float64(completedDays) / float64(days)
}
```

**Step 4: Run test to verify it passes**

Run: `cd go-backend && go test ./services -run TestCalculateHabitStats`
Expected: PASS

**Step 5: Commit**

```bash
git add go-backend/services/habits.go go-backend/services/habits_test.go
git commit -m "feat(habits): add streak calculation and stats"
```

---

## Task 7: Backend Services - Today's Habits

**Files:**
- Modify: `go-backend/services/habits.go`

**Step 1: Write the failing test**

Add to `go-backend/services/habits_test.go`:

```go
func TestGetTodaysHabits(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    habitID, _ := CreateHabit(db, 1, models.CreateHabitParams{
        Title: "Daily Habit",
        Frequency: models.FrequencyDaily,
    })

    habits, err := GetTodaysHabits(db, 1, time.UTC)
    if err != nil {
        t.Fatalf("failed to get today's habits: %v", err)
    }
    if len(habits) != 1 {
        t.Errorf("expected 1 habit, got %d", len(habits))
    }
    if !habits[0].IsDueToday {
        t.Error("expected habit to be due today")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./services -run TestGetTodaysHabits`
Expected: FAIL - "undefined: GetTodaysHabits"

**Step 3: Write minimal implementation**

Add to `go-backend/services/habits.go`:

```go
import "encoding/json"

type HabitWithCheckin struct {
    models.Habit
    IsDueToday     bool  `json:"is_due_today"`
    CheckedInToday bool  `json:"checked_in_today"`
    TodayLogID     *int  `json:"today_log_id,omitempty"`
}

func GetTodaysHabits(db *sql.DB, userID int, timezone string) ([]HabitWithCheckin, error) {
    habits, err := GetHabits(db, userID)
    if err != nil {
        return nil, err
    }

    var result []HabitWithCheckin
    today := time.Now().UTC().Format("2006-01-02")
    currentWeekday := int(time.Now().UTC().Weekday())
    if currentWeekday == 0 {
        currentWeekday = 7 // Convert Sunday from 0 to 7
    }

    for _, habit := range habits {
        var hc HabitWithCheckin
        hc.Habit = habit

        // Determine if habit is due today
        hc.IsDueToday = isHabitDueToday(&habit, currentWeekday)

        if hc.IsDueToday {
            // Check if already checked in today
            var logID sql.NullInt64
            checkQuery := `
                SELECT id FROM habit_logs
                WHERE habit_id = $1 AND user_id = $2
                AND DATE(completed_at AT TIME ZONE $3) = $4
                LIMIT 1
            `
            err := db.QueryRow(checkQuery, habit.ID, userID, timezone, today).Scan(&logID)
            hc.CheckedInToday = (err == nil)
            if logID.Valid {
                id := int(logID.Int64)
                hc.TodayLogID = &id
            }

            result = append(result, hc)
        }
    }

    return result, nil
}

func isHabitDueToday(habit *models.Habit, currentWeekday int) bool {
    switch habit.Frequency {
    case models.FrequencyDaily:
        return true
    case models.FrequencyWeekly, models.FrequencyCustom:
        if habit.CustomDays != nil {
            var customDays []int
            json.Unmarshal([]byte(*habit.CustomDays), &customDays)
            for _, day := range customDays {
                if day == currentWeekday {
                    return true
                }
            }
        }
        return false
    default:
        return true
    }
}
```

**Step 4: Run test to verify it passes**

Run: `cd go-backend && go test ./services -run TestGetTodaysHabits`
Expected: PASS

**Step 5: Commit**

```bash
git add go-backend/services/habits.go go-backend/services/habits_test.go
git commit -m "feat(habits): add GetTodaysHabits function"
```

---

## Task 8: Backend Handlers - Habit Routes

**Files:**
- Create: `go-backend/handlers/habits.go`
- Modify: `go-backend/main.go` (register routes)

**Step 1: Write the failing test**

Create file `go-backend/handlers/habits_test.go`:

```go
package handlers

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "go-backend/models"
)

func TestGetHabitsRoute(t *testing.T) {
    s := setupTestHandler(t)
    defer s.db.Close()

    // Create a test habit
    _, err := services.CreateHabit(s.db, 1, models.CreateHabitParams{
        Title:     "Test Habit",
        Frequency: models.FrequencyDaily,
    })
    if err != nil {
        t.Fatalf("failed to create habit: %v", err)
    }

    req := httptest.NewRequest("GET", "/api/habits", nil)
    req.Header.Set("Authorization", "Bearer "+s.token)
    w := httptest.NewRecorder()

    s.handler.GetHabitsRoute(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("expected status 200, got %d", w.Code)
    }

    var response []models.Habit
    json.NewDecoder(w.Body).Decode(&response)

    if len(response) != 1 {
        t.Errorf("expected 1 habit, got %d", len(response))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `cd go-backend && go test ./handlers -run TestGetHabitsRoute`
Expected: FAIL - handler not defined

**Step 3: Write minimal implementation**

Create file `go-backend/handlers/habits.go`:

```go
package handlers

import (
    "encoding/json"
    "log"
    "net/http"
    "strconv"
    "go-backend/models"
    "go-backend/services"
    "github.com/gorilla/mux"
)

func (s *Handler) GetHabitsRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)

    habits, err := services.GetHabits(s.GetDB(), userID)
    if err != nil {
        log.Printf("Error getting habits for user %d: %v", userID, err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(habits)
}

func (s *Handler) GetHabitRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        log.Printf("Invalid habit id param: %v", err)
        http.Error(w, "Invalid id", http.StatusBadRequest)
        return
    }

    habit, err := services.GetHabit(s.GetDB(), userID, id)
    if err != nil {
        log.Printf("Error getting habit %d: %v", id, err)
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(habit)
}

func (s *Handler) CreateHabitRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)

    var params models.CreateHabitParams
    if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
        log.Printf("Error decoding create habit request: %v", err)
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    habitID, err := services.CreateHabit(s.GetDB(), userID, params)
    if err != nil {
        log.Printf("Error creating habit: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]int{"id": habitID})
}

func (s *Handler) UpdateHabitRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        log.Printf("Invalid habit id param: %v", err)
        http.Error(w, "Invalid id", http.StatusBadRequest)
        return
    }

    var params models.UpdateHabitParams
    if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
        log.Printf("Error decoding update habit request: %v", err)
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    if err := services.UpdateHabit(s.GetDB(), userID, id, params); err != nil {
        log.Printf("Error updating habit %d: %v", id, err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(models.GenericResponse{
        Message: "success",
        Error:   false,
    })
}

func (s *Handler) DeleteHabitRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        log.Printf("Invalid habit id param: %v", err)
        http.Error(w, "Invalid id", http.StatusBadRequest)
        return
    }

    err = services.DeleteHabit(s.GetDB(), userID, id)
    if err != nil {
        log.Printf("Error deleting habit %d: %v", id, err)
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (s *Handler) CheckinHabitRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        log.Printf("Invalid habit id param: %v", err)
        http.Error(w, "Invalid id", http.StatusBadRequest)
        return
    }

    var params models.CheckinHabitParams
    if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
        // Empty body is ok
        params = models.CheckinHabitParams{}
    }

    userTimezone, err := s.GetUserTimezone(userID)
    if err != nil {
        userTimezone = "UTC"
    }

    logID, err := services.CheckinHabit(s.GetDB(), userID, id, params, userTimezone)
    if err != nil {
        if err.Error() == "already checked in today" {
            http.Error(w, err.Error(), http.StatusConflict)
            return
        }
        log.Printf("Error checking in to habit %d: %v", id, err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]int{"id": logID})
}

func (s *Handler) DeleteHabitLogRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        http.Error(w, "Invalid habit ID", http.StatusBadRequest)
        return
    }

    logID, err := strconv.Atoi(mux.Vars(r)["logId"])
    if err != nil {
        http.Error(w, "Invalid log ID", http.StatusBadRequest)
        return
    }

    err = services.DeleteHabitLog(s.GetDB(), userID, id, logID)
    if err != nil {
        log.Printf("Error deleting habit log %d: %v", logID, err)
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (s *Handler) GetHabitLogsRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        http.Error(w, "Invalid habit ID", http.StatusBadRequest)
        return
    }

    limit := 50
    if l := r.URL.Query().Get("limit"); l != "" {
        if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
            limit = parsed
        }
    }

    offset := 0
    if o := r.URL.Query().Get("offset"); o != "" {
        if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
            offset = parsed
        }
    }

    logs, total, err := services.GetHabitLogs(s.GetDB(), userID, id, limit, offset)
    if err != nil {
        log.Printf("Error getting habit logs: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    response := struct {
        Logs  []models.HabitLog `json:"logs"`
        Total int               `json:"total"`
        Limit int               `json:"limit"`
        Offset int              `json:"offset"`
    }{
        Logs:   logs,
        Total:  total,
        Limit:  limit,
        Offset: offset,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (s *Handler) GetHabitStatsRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        http.Error(w, "Invalid habit ID", http.StatusBadRequest)
        return
    }

    userTimezone, err := s.GetUserTimezone(userID)
    if err != nil {
        userTimezone = "UTC"
    }

    stats, err := services.CalculateHabitStats(s.GetDB(), userID, id, userTimezone)
    if err != nil {
        log.Printf("Error getting habit stats: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

func (s *Handler) GetTodaysHabitsRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)

    userTimezone, err := s.GetUserTimezone(userID)
    if err != nil {
        userTimezone = "UTC"
    }

    habits, err := services.GetTodaysHabits(s.GetDB(), userID, userTimezone)
    if err != nil {
        log.Printf("Error getting today's habits: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(habits)
}

func (s *Handler) LinkHabitToTaskRoute(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("current_user").(int)
    id, err := strconv.Atoi(mux.Vars(r)["id"])
    if err != nil {
        http.Error(w, "Invalid habit ID", http.StatusBadRequest)
        return
    }

    var requestBody struct {
        TaskID *int `json:"task_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    params := models.UpdateHabitParams{
        LinkedTaskID: requestBody.TaskID,
    }

    if err := services.UpdateHabit(s.GetDB(), userID, id, params); err != nil {
        log.Printf("Error linking habit to task: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(models.GenericResponse{
        Message: "success",
        Error:   false,
    })
}
```

**Step 4: Register routes in main.go**

Add to `go-backend/main.go` in the route registration section:

```go
// Habits routes
api.HandleFunc("/habits", handlers.LogRoute(handlers.GetHabitsRoute)).Methods("GET")
api.HandleFunc("/habits", handlers.LogRoute(handlers.CreateHabitRoute)).Methods("POST")
api.HandleFunc("/habits/today", handlers.LogRoute(handlers.GetTodaysHabitsRoute)).Methods("GET")
api.HandleFunc("/habits/{id}", handlers.LogRoute(handlers.GetHabitRoute)).Methods("GET")
api.HandleFunc("/habits/{id}", handlers.LogRoute(handlers.UpdateHabitRoute)).Methods("PUT")
api.HandleFunc("/habits/{id}", handlers.LogRoute(handlers.DeleteHabitRoute)).Methods("DELETE")
api.HandleFunc("/habits/{id}/checkin", handlers.LogRoute(handlers.CheckinHabitRoute)).Methods("POST")
api.HandleFunc("/habits/{id}/checkin/{logId}", handlers.LogRoute(handlers.DeleteHabitLogRoute)).Methods("DELETE")
api.HandleFunc("/habits/{id}/logs", handlers.LogRoute(handlers.GetHabitLogsRoute)).Methods("GET")
api.HandleFunc("/habits/{id}/stats", handlers.LogRoute(handlers.GetHabitStatsRoute)).Methods("GET")
api.HandleFunc("/habits/{id}/link", handlers.LogRoute(handlers.LinkHabitToTaskRoute)).Methods("PUT")
```

**Step 5: Run test to verify it passes**

Run: `cd go-backend && go test ./handlers -run TestGetHabitsRoute`
Expected: PASS

**Step 6: Commit**

```bash
git add go-backend/handlers/habits.go go-backend/handlers/habits_test.go go-backend/main.go
git commit -m "feat(habits): add habit HTTP handlers and routes"
```

---

## Task 9: Frontend - Habit Context and Types

**Files:**
- Create: `zettelkasten-front/src/contexts/HabitContext.tsx`
- Create: `zettelkasten-front/src/models/habit.ts`

**Step 1: Write the types**

Create file `zettelkasten-front/src/models/habit.ts`:

```typescript
export interface Habit {
  id: number;
  user_id: number;
  title: string;
  description?: string;
  frequency: 'daily' | 'weekly' | 'custom_days';
  custom_days?: string; // JSON array string
  icon?: string;
  color?: string;
  position: number;
  linked_task_id?: number;
  created_at: string;
  updated_at: string;
  today_checked_in?: boolean;
  current_streak?: number;
}

export interface HabitWithCheckin extends Habit {
  is_due_today: boolean;
  checked_in_today: boolean;
  today_log_id?: number;
}

export interface HabitLog {
  id: number;
  habit_id: number;
  user_id: number;
  completed_at: string;
  notes?: string;
  created_at: string;
}

export interface HabitStats {
  current_streak: number;
  longest_streak: number;
  total_completions: number;
  completion_rate_7d: number;
  completion_rate_30d: number;
  last_completed_at?: string;
}

export interface CreateHabitParams {
  title: string;
  description?: string;
  frequency: 'daily' | 'weekly' | 'custom_days';
  custom_days?: number[];
  icon?: string;
  color?: string;
  linked_task_id?: number;
}

export interface UpdateHabitParams {
  title?: string;
  description?: string;
  frequency?: 'daily' | 'weekly' | 'custom_days';
  custom_days?: number[];
  icon?: string;
  color?: string;
  linked_task_id?: number;
}

export interface CheckinHabitParams {
  notes?: string;
}
```

**Step 2: Write the context**

Create file `zettelkasten-front/src/contexts/HabitContext.tsx`:

```typescript
import React, { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import {
  Habit,
  HabitWithCheckin,
  HabitLog,
  HabitStats,
  CreateHabitParams,
  UpdateHabitParams,
  CheckinHabitParams
} from '../models/habit';
import * as habitApi from '../api/habits';

interface HabitContextType {
  habits: Habit[];
  todaysHabits: HabitWithCheckin[];
  selectedHabit: Habit | null;
  selectedHabitLogs: HabitLog[];
  selectedHabitStats: HabitStats | null;
  loading: boolean;
  error: string | null;
  fetchHabits: () => Promise<void>;
  fetchTodaysHabits: () => Promise<void>;
  fetchHabit: (id: number) => Promise<void>;
  createHabit: (params: CreateHabitParams) => Promise<number>;
  updateHabit: (id: number, params: UpdateHabitParams) => Promise<void>;
  deleteHabit: (id: number) => Promise<void>;
  checkinHabit: (id: number, params?: CheckinHabitParams) => Promise<number>;
  deleteHabitLog: (habitId: number, logId: number) => Promise<void>;
  fetchHabitLogs: (id: number) => Promise<void>;
  fetchHabitStats: (id: number) => Promise<void>;
  linkHabitToTask: (id: number, taskId: number | null) => Promise<void>;
  setSelectedHabit: (habit: Habit | null) => void;
}

const HabitContext = createContext<HabitContextType | undefined>(undefined);

export const HabitProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [habits, setHabits] = useState<Habit[]>([]);
  const [todaysHabits, setTodaysHabits] = useState<HabitWithCheckin[]>([]);
  const [selectedHabit, setSelectedHabit] = useState<Habit | null>(null);
  const [selectedHabitLogs, setSelectedHabitLogs] = useState<HabitLog[]>([]);
  const [selectedHabitStats, setSelectedHabitStats] = useState<HabitStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHabits = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await habitApi.getHabits();
      setHabits(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch habits');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchTodaysHabits = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await habitApi.getTodaysHabits();
      setTodaysHabits(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch today\'s habits');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchHabit = useCallback(async (id: number) => {
    setLoading(true);
    setError(null);
    try {
      const habit = await habitApi.getHabit(id);
      setSelectedHabit(habit);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch habit');
    } finally {
      setLoading(false);
    }
  }, []);

  const createHabit = useCallback(async (params: CreateHabitParams): Promise<number> => {
    setLoading(true);
    setError(null);
    try {
      const id = await habitApi.createHabit(params);
      await fetchHabits();
      await fetchTodaysHabits();
      return id;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create habit');
      throw err;
    } finally {
      setLoading(false);
    }
  }, [fetchHabits, fetchTodaysHabits]);

  const updateHabit = useCallback(async (id: number, params: UpdateHabitParams) => {
    setLoading(true);
    setError(null);
    try {
      await habitApi.updateHabit(id, params);
      await fetchHabits();
      await fetchTodaysHabits();
      if (selectedHabit?.id === id) {
        await fetchHabit(id);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update habit');
      throw err;
    } finally {
      setLoading(false);
    }
  }, [fetchHabits, fetchTodaysHabits, fetchHabit, selectedHabit]);

  const deleteHabit = useCallback(async (id: number) => {
    setLoading(true);
    setError(null);
    try {
      await habitApi.deleteHabit(id);
      await fetchHabits();
      await fetchTodaysHabits();
      if (selectedHabit?.id === id) {
        setSelectedHabit(null);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete habit');
      throw err;
    } finally {
      setLoading(false);
    }
  }, [fetchHabits, fetchTodaysHabits, selectedHabit]);

  const checkinHabit = useCallback(async (id: number, params?: CheckinHabitParams): Promise<number> => {
    setLoading(true);
    setError(null);
    try {
      const logId = await habitApi.checkinHabit(id, params);
      await fetchTodaysHabits();
      if (selectedHabit?.id === id) {
        await fetchHabitStats(id);
      }
      return logId;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to check in');
      throw err;
    } finally {
      setLoading(false);
    }
  }, [fetchTodaysHabits, fetchHabitStats, selectedHabit]);

  const deleteHabitLog = useCallback(async (habitId: number, logId: number) => {
    setLoading(true);
    setError(null);
    try {
      await habitApi.deleteHabitLog(habitId, logId);
      await fetchTodaysHabits();
      if (selectedHabit?.id === habitId) {
        await fetchHabitLogs(habitId);
        await fetchHabitStats(habitId);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete check-in');
      throw err;
    } finally {
      setLoading(false);
    }
  }, [fetchTodaysHabits, fetchHabitLogs, fetchHabitStats, selectedHabit]);

  const fetchHabitLogs = useCallback(async (id: number) => {
    setLoading(true);
    setError(null);
    try {
      const data = await habitApi.getHabitLogs(id);
      setSelectedHabitLogs(data.logs);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch logs');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchHabitStats = useCallback(async (id: number) => {
    setLoading(true);
    setError(null);
    try {
      const stats = await habitApi.getHabitStats(id);
      setSelectedHabitStats(stats);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch stats');
    } finally {
      setLoading(false);
    }
  }, []);

  const linkHabitToTask = useCallback(async (id: number, taskId: number | null) => {
    setLoading(true);
    setError(null);
    try {
      await habitApi.linkHabitToTask(id, taskId);
      await fetchHabits();
      if (selectedHabit?.id === id) {
        await fetchHabit(id);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to link habit');
      throw err;
    } finally {
      setLoading(false);
    }
  }, [fetchHabits, fetchHabit, selectedHabit]);

  const value: HabitContextType = {
    habits,
    todaysHabits,
    selectedHabit,
    selectedHabitLogs,
    selectedHabitStats,
    loading,
    error,
    fetchHabits,
    fetchTodaysHabits,
    fetchHabit,
    createHabit,
    updateHabit,
    deleteHabit,
    checkinHabit,
    deleteHabitLog,
    fetchHabitLogs,
    fetchHabitStats,
    linkHabitToTask,
    setSelectedHabit,
  };

  return <HabitContext.Provider value={value}>{children}</HabitContext.Provider>;
};

export const useHabits = () => {
  const context = useContext(HabitContext);
  if (context === undefined) {
    throw new Error('useHabits must be used within a HabitProvider');
  }
  return context;
};
```

**Step 3: Write tests for context**

Create file `zettelkasten-front/src/contexts/__tests__/HabitContext.test.tsx`:

```typescript
import { renderHook, act, waitFor } from '@testing-library/react';
import { HabitProvider, useHabits } from '../HabitContext';
import * as habitApi from '../../api/habits';

jest.mock('../../api/habits');

describe('HabitContext', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('provides habits context', async () => {
    const mockHabits = [{ id: 1, title: 'Test', frequency: 'daily' as const, user_id: 1, position: 0, created_at: '', updated_at: '' }];
    (habitApi.getHabits as jest.Mock).mockResolvedValue(mockHabits);

    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <HabitProvider>{children}</HabitProvider>
    );

    const { result } = renderHook(() => useHabits(), { wrapper });

    await act(async () => {
      await result.current.fetchHabits();
    });

    expect(result.current.habits).toEqual(mockHabits);
  });
});
```

**Step 4: Run tests**

Run: `cd zettelkasten-front && npm test -- HabitContext.test.tsx`
Expected: Tests pass (or fail with expected errors we'll fix)

**Step 5: Commit**

```bash
git add zettelkasten-front/src/models/habit.ts zettelkasten-front/src/contexts/HabitContext.tsx
git commit -m "feat(habits): add habit types and context"
```

---

## Task 10: Frontend - Habit API Client

**Files:**
- Create: `zettelkasten-front/src/api/habits.ts`

**Step 1: Write the API client**

Create file `zettelkasten-front/src/api/habits.ts`:

```typescript
import axios from 'axios';
import {
  Habit,
  HabitWithCheckin,
  HabitLog,
  HabitStats,
  CreateHabitParams,
  UpdateHabitParams,
  CheckinHabitParams
} from '../models/habit';

const API_BASE = '/api';

export async function getHabits(): Promise<Habit[]> {
  const response = await axios.get<Habit[]>(`${API_BASE}/habits`);
  return response.data;
}

export async function getHabit(id: number): Promise<Habit> {
  const response = await axios.get<Habit>(`${API_BASE}/habits/${id}`);
  return response.data;
}

export async function getTodaysHabits(): Promise<HabitWithCheckin[]> {
  const response = await axios.get<HabitWithCheckin[]>(`${API_BASE}/habits/today`);
  return response.data;
}

export async function createHabit(params: CreateHabitParams): Promise<number> {
  const response = await axios.post<{ id: number }>(`${API_BASE}/habits`, params);
  return response.data.id;
}

export async function updateHabit(id: number, params: UpdateHabitParams): Promise<void> {
  await axios.put(`${API_BASE}/habits/${id}`, params);
}

export async function deleteHabit(id: number): Promise<void> {
  await axios.delete(`${API_BASE}/habits/${id}`);
}

export async function checkinHabit(id: number, params?: CheckinHabitParams): Promise<number> {
  const response = await axios.post<{ id: number }>(`${API_BASE}/habits/${id}/checkin`, params || {});
  return response.data.id;
}

export async function deleteHabitLog(habitId: number, logId: number): Promise<void> {
  await axios.delete(`${API_BASE}/habits/${habitId}/checkin/${logId}`);
}

export async function getHabitLogs(id: number, limit = 50, offset = 0): Promise<{ logs: HabitLog[]; total: number }> {
  const response = await axios.get<{ logs: HabitLog[]; total: number; limit: number; offset: number }>(
    `${API_BASE}/habits/${id}/logs`,
    { params: { limit, offset } }
  );
  return { logs: response.data.logs, total: response.data.total };
}

export async function getHabitStats(id: number): Promise<HabitStats> {
  const response = await axios.get<HabitStats>(`${API_BASE}/habits/${id}/stats`);
  return response.data;
}

export async function linkHabitToTask(id: number, taskId: number | null): Promise<void> {
  await axios.put(`${API_BASE}/habits/${id}/link`, { task_id: taskId });
}
```

**Step 2: Commit**

```bash
git add zettelkasten-front/src/api/habits.ts
git commit -m "feat(habits): add habit API client"
```

---

## Task 11: Frontend - Habits Page

**Files:**
- Create: `zettelkasten-front/src/pages/Habits.tsx`
- Create: `zettelkasten-front/src/components/habits/HabitList.tsx`
- Create: `zettelkasten-front/src/components/habits/HabitDetail.tsx`

**Step 1: Create HabitList component**

Create file `zettelkasten-front/src/components/habits/HabitList.tsx`:

```typescript
import React from 'react';
import { useHabits } from '../../contexts/HabitContext';
import { Habit } from '../../models/habit';

export const HabitList: React.FC = () => {
  const { habits, selectedHabit, setSelectedHabit, deleteHabit, loading } = useHabits();

  const handleDelete = async (id: number) => {
    if (confirm('Are you sure you want to delete this habit?')) {
      try {
        await deleteHabit(id);
      } catch (err) {
        console.error('Failed to delete habit:', err);
      }
    }
  };

  if (loading && habits.length === 0) {
    return <div className="p-4">Loading habits...</div>;
  }

  return (
    <div className="habit-list">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold">Habits</h2>
        <button className="btn-primary">+ New Habit</button>
      </div>
      {habits.length === 0 ? (
        <div className="text-center p-8 text-gray-500">
          No habits yet. Create your first habit to start tracking!
        </div>
      ) : (
        <div className="space-y-2">
          {habits.map((habit) => (
            <div
              key={habit.id}
              className={`p-4 rounded-lg cursor-pointer transition-colors ${
                selectedHabit?.id === habit.id
                  ? 'bg-blue-50 border-l-4 border-blue-500'
                  : 'hover:bg-gray-50'
              }`}
              onClick={() => setSelectedHabit(habit)}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  {habit.icon && <span className="text-2xl">{habit.icon}</span>}
                  <div>
                    <div className="font-medium">{habit.title}</div>
                    <div className="text-sm text-gray-500">
                      {habit.frequency === 'daily' ? 'Daily' : 'Custom schedule'}
                    </div>
                  </div>
                </div>
                <button
                  className="text-red-500 hover:text-red-700 px-2"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDelete(habit.id);
                  }}
                >
                  ×
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
```

**Step 2: Create HabitDetail component**

Create file `zettelkasten-front/src/components/habits/HabitDetail.tsx`:

```typescript
import React, { useEffect } from 'react';
import { useHabits } from '../../contexts/HabitContext';
import { Habit } from '../../models/habit';

export const HabitDetail: React.FC = () => {
  const { selectedHabit, selectedHabitStats, selectedHabitLogs, fetchHabitStats, fetchHabitLogs, checkinHabit, deleteHabitLog } = useHabits();

  useEffect(() => {
    if (selectedHabit) {
      fetchHabitStats(selectedHabit.id);
      fetchHabitLogs(selectedHabit.id);
    }
  }, [selectedHabit, fetchHabitStats, fetchHabitLogs]);

  const handleCheckin = async () => {
    if (!selectedHabit) return;
    try {
      await checkinHabit(selectedHabit.id);
    } catch (err) {
      if (err instanceof Error && err.message.includes('already checked in')) {
        alert('Already checked in today!');
      } else {
        console.error('Failed to check in:', err);
      }
    }
  };

  const handleDeleteLog = async (logId: number) => {
    if (!selectedHabit) return;
    try {
      await deleteHabitLog(selectedHabit.id, logId);
    } catch (err) {
      console.error('Failed to delete log:', err);
    }
  };

  if (!selectedHabit) {
    return (
      <div className="flex items-center justify-center h-full text-gray-500">
        Select a habit to view details
      </div>
    );
  }

  return (
    <div className="habit-detail">
      <div className="mb-6">
        <div className="flex items-center gap-3 mb-2">
          {selectedHabit.icon && <span className="text-3xl">{selectedHabit.icon}</span>}
          <h2 className="text-2xl font-bold">{selectedHabit.title}</h2>
        </div>
        {selectedHabit.description && (
          <p className="text-gray-600">{selectedHabit.description}</p>
        )}
      </div>

      <div className="mb-6">
        <button className="btn-primary w-full py-3 text-lg" onClick={handleCheckin}>
          Check In
        </button>
      </div>

      {selectedHabitStats && (
        <div className="mb-6 p-4 bg-gray-50 rounded-lg">
          <h3 className="font-semibold mb-3">Statistics</h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <div className="text-2xl font-bold text-orange-500">
                🔥 {selectedHabitStats.current_streak}
              </div>
              <div className="text-sm text-gray-500">Current Streak</div>
            </div>
            <div>
              <div className="text-2xl font-bold">{selectedHabitStats.longest_streak}</div>
              <div className="text-sm text-gray-500">Longest Streak</div>
            </div>
            <div>
              <div className="text-2xl font-bold">{selectedHabitStats.total_completions}</div>
              <div className="text-sm text-gray-500">Total Check-ins</div>
            </div>
            <div>
              <div className="text-2xl font-bold">
                {Math.round(selectedHabitStats.completion_rate_7d * 100)}%
              </div>
              <div className="text-sm text-gray-500">7-Day Rate</div>
            </div>
          </div>
        </div>
      )}

      <div>
        <h3 className="font-semibold mb-3">Recent Check-ins</h3>
        {selectedHabitLogs.length === 0 ? (
          <div className="text-gray-500 text-center py-4">No check-ins yet</div>
        ) : (
          <div className="space-y-2">
            {selectedHabitLogs.slice(0, 10).map((log) => (
              <div key={log.id} className="flex justify-between items-center p-3 bg-gray-50 rounded">
                <div>
                  <div className="text-sm">
                    {new Date(log.completed_at).toLocaleDateString()}
                  </div>
                  {log.notes && <div className="text-xs text-gray-500">{log.notes}</div>}
                </div>
                <button
                  className="text-red-500 hover:text-red-700 text-sm"
                  onClick={() => handleDeleteLog(log.id)}
                >
                  Undo
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
```

**Step 3: Create Habits page**

Create file `zettelkasten-front/src/pages/Habits.tsx`:

```typescript
import React, { useEffect } from 'react';
import { useHabits } from '../contexts/HabitContext';
import { HabitList } from '../components/habits/HabitList';
import { HabitDetail } from '../components/habits/HabitDetail';

export const Habits: React.FC = () => {
  const { fetchHabits } = useHabits();

  useEffect(() => {
    fetchHabits();
  }, [fetchHabits]);

  return (
    <div className="container mx-auto px-4 py-6">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="md:col-span-1">
          <HabitList />
        </div>
        <div className="md:col-span-2">
          <HabitDetail />
        </div>
      </div>
    </div>
  );
};
```

**Step 4: Add route to App.tsx**

Add to your routing configuration:

```typescript
import { Habits } from './pages/Habits';

// In your routes:
<Route path="/habits" element={<Habits />} />
```

**Step 5: Commit**

```bash
git add zettelkasten-front/src/pages/Habits.tsx zettelkasten-front/src/components/habits/
git commit -m "feat(habits): add habits page with list and detail views"
```

---

## Task 12: Frontend - Sidebar Widget

**Files:**
- Create: `zettelkasten-front/src/components/habits/TodaysHabitsWidget.tsx`
- Modify: `zettelkasten-front/src/components/Sidebar.tsx` (add widget)

**Step 1: Create TodaysHabitsWidget**

Create file `zettelkasten-front/src/components/habits/TodaysHabitsWidget.tsx`:

```typescript
import React, { useEffect, useState } from 'react';
import { useHabits } from '../../contexts/HabitContext';

export const TodaysHabitsWidget: React.FC = () => {
  const { todaysHabits, fetchTodaysHabits, checkinHabit, loading } = useHabits();
  const [checkingIn, setCheckingIn] = useState<number | null>(null);

  useEffect(() => {
    fetchTodaysHabits();
    // Refresh every minute
    const interval = setInterval(fetchTodaysHabits, 60000);
    return () => clearInterval(interval);
  }, [fetchTodaysHabits]);

  const handleCheckin = async (habitId: number) => {
    setCheckingIn(habitId);
    try {
      await checkinHabit(habitId);
    } catch (err) {
      if (err instanceof Error && !err.message.includes('already checked in')) {
        console.error('Failed to check in:', err);
      }
    } finally {
      setCheckingIn(null);
    }
  };

  if (todaysHabits.length === 0) {
    return null;
  }

  return (
    <div className="sidebar-section">
      <h3 className="sidebar-section-title">Today's Habits</h3>
      <div className="space-y-2">
        {todaysHabits.map((habit) => (
          <div
            key={habit.id}
            className="flex items-center justify-between p-2 rounded hover:bg-gray-100"
          >
            <div className="flex items-center gap-2 flex-1 min-w-0">
              {habit.icon && <span className="text-lg">{habit.icon}</span>}
              <span className="truncate text-sm">{habit.title}</span>
            </div>
            <button
              className={`ml-2 px-3 py-1 rounded text-sm ${
                habit.checked_in_today
                  ? 'bg-green-500 text-white'
                  : 'bg-gray-200 hover:bg-gray-300'
              }`}
              onClick={() => handleCheckin(habit.id)}
              disabled={checkingIn === habit.id || habit.checked_in_today}
            >
              {habit.checked_in_today ? '✓' : checkingIn === habit.id ? '...' : '✓'}
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};
```

**Step 2: Add to Sidebar**

Modify your Sidebar component to include the widget. The exact location depends on your sidebar structure, but add something like:

```typescript
import { TodaysHabitsWidget } from './habits/TodaysHabitsWidget';

// In your Sidebar component, add:
<TodaysHabitsWidget />
```

**Step 3: Wrap App with HabitProvider**

Modify your App.tsx or main entry point:

```typescript
import { HabitProvider } from './contexts/HabitContext';

// Wrap your app:
<HabitProvider>
  {/* existing app content */}
</HabitProvider>
```

**Step 4: Commit**

```bash
git add zettelkasten-front/src/components/habits/TodaysHabitsWidget.tsx
git add zettelkasten-front/src/components/Sidebar.tsx
git add zettelkasten-front/src/App.tsx  # or wherever you add the provider
git commit -m "feat(habits): add today's habits sidebar widget"
```

---

## Task 13: Frontend - Create Habit Dialog

**Files:**
- Create: `zettelkasten-front/src/components/habits/CreateHabitDialog.tsx`

**Step 1: Create dialog component**

Create file `zettelkasten-front/src/components/habits/CreateHabitDialog.tsx`:

```typescript
import React, { useState } from 'react';
import { useHabits } from '../../contexts/HabitContext';
import { CreateHabitParams } from '../../models/habit';

interface CreateHabitDialogProps {
  onClose: () => void;
}

export const CreateHabitDialog: React.FC<CreateHabitDialogProps> = ({ onClose }) => {
  const { createHabit } = useHabits();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [frequency, setFrequency] = useState<'daily' | 'weekly' | 'custom_days'>('daily');
  const [customDays, setCustomDays] = useState<number[]>([]);
  const [icon, setIcon] = useState('');
  const [color, setColor] = useState('#10b981');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const params: CreateHabitParams = {
      title,
      description: description || undefined,
      frequency,
      custom_days: frequency === 'custom_days' ? customDays : undefined,
      icon: icon || undefined,
      color,
    };
    try {
      await createHabit(params);
      onClose();
    } catch (err) {
      console.error('Failed to create habit:', err);
    }
  };

  const toggleDay = (day: number) => {
    if (customDays.includes(day)) {
      setCustomDays(customDays.filter((d) => d !== day));
    } else {
      setCustomDays([...customDays, day]);
    }
  };

  const ICONS = ['💊', '🏃', '📚', '💧', '🧘', '💪', '🎯', '✅'];
  const COLORS = ['#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899'];

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-full max-w-md">
        <h2 className="text-xl font-bold mb-4">Create New Habit</h2>
        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Title *</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full border rounded px-3 py-2"
              required
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full border rounded px-3 py-2"
              rows={2}
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Frequency</label>
            <select
              value={frequency}
              onChange={(e) => setFrequency(e.target.value as any)}
              className="w-full border rounded px-3 py-2"
            >
              <option value="daily">Daily</option>
              <option value="custom_days">Specific Days</option>
            </select>
          </div>

          {frequency === 'custom_days' && (
            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">Select Days</label>
              <div className="flex gap-2">
                {['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].map((day, i) => (
                  <button
                    key={day}
                    type="button"
                    className={`px-3 py-1 rounded text-sm ${
                      customDays.includes(i + 1) ? 'bg-blue-500 text-white' : 'bg-gray-200'
                    }`}
                    onClick={() => toggleDay(i + 1)}
                  >
                    {day}
                  </button>
                ))}
              </div>
            </div>
          )}

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Icon</label>
            <div className="flex gap-2 flex-wrap">
              {ICONS.map((i) => (
                <button
                  key={i}
                  type="button"
                  className={`text-2xl p-2 rounded ${icon === i ? 'bg-gray-200' : ''}`}
                  onClick={() => setIcon(i)}
                >
                  {i}
                </button>
              ))}
            </div>
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1">Color</label>
            <div className="flex gap-2">
              {COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  className={`w-8 h-8 rounded ${color === c ? 'ring-2 ring-offset-2' : ''}`}
                  style={{ backgroundColor: c }}
                  onClick={() => setColor(c)}
                />
              ))}
            </div>
          </div>

          <div className="flex justify-end gap-2">
            <button type="button" className="px-4 py-2 border rounded" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="px-4 py-2 bg-blue-500 text-white rounded">
              Create
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
```

**Step 2: Wire up to HabitList "New Habit" button**

Modify `HabitList.tsx` to use the dialog:

```typescript
import { useState } from 'react';
import { CreateHabitDialog } from './CreateHabitDialog';

// In HabitList component:
const [showCreateDialog, setShowCreateDialog] = useState(false);

// In the JSX:
{showCreateDialog && <CreateHabitDialog onClose={() => setShowCreateDialog(false)} />}
<button className="btn-primary" onClick={() => setShowCreateDialog(true)}>+ New Habit</button>
```

**Step 3: Commit**

```bash
git add zettelkasten-front/src/components/habits/CreateHabitDialog.tsx
git add zettelkasten-front/src/components/habits/HabitList.tsx
git commit -m "feat(habits): add create habit dialog"
```

---

## Task 14: Integration - Add Habits Link to Navigation

**Files:**
- Modify: `zettelkasten-front/src/components/Sidebar.tsx` (or wherever navigation is)

**Step 1: Add habits link to navigation**

Add a link to the habits page in your sidebar/navigation:

```typescript
// Add to navigation links:
<Link to="/habits" className="nav-link">
  📋 Habits
</Link>
```

**Step 2: Commit**

```bash
git add zettelkasten-front/src/components/Sidebar.tsx
git commit -m "feat(habits): add habits link to sidebar navigation"
```

---

## Task 15: End-to-End Testing

**Step 1: Test the full flow**

Manual testing checklist:

1. Navigate to /habits
2. Click "New Habit"
3. Fill in form: title="Test Habit", frequency=daily, select icon and color
4. Click Create
5. Verify habit appears in list
6. Click on habit to see detail view
7. Click "Check In" button
8. Verify stats update (streak = 1)
9. Check sidebar widget - habit appears with green checkmark
10. Try checking in again - should show "already checked in" message
11. Click "Undo" on recent check-in
12. Verify streak resets, sidebar shows unchecked
13. Delete habit
14. Confirm it's removed from list

**Step 2: Test timezone behavior**

1. Create a habit
2. Check in
3. Change user timezone in settings
4. Verify "today" correctly shifts

**Step 3: Test frequency settings**

1. Create habit with "Specific Days" - select Mon/Wed/Fri
2. Verify it only appears in "Today's Habits" on those days

**Step 4: Test task linking**

1. Create a habit
2. Verify linked_task_id can be set via API
3. Create a task and link it

**Step 5: Commit any fixes**

```bash
git add -A
git commit -m "fix(habits): address integration test findings"
```

---

## Final Steps

1. **Run all tests**:
   - Backend: `cd go-backend && go test ./...`
   - Frontend: `cd zettelkasten-front && npm test`

2. **Build and verify**:
   ```bash
   cd zettelkasten-front && npm run build
   cd ../go-backend && go build
   ```

3. **Final commit**:
   ```bash
   git add -A
   git commit -m "feat(habits): complete habits tracking feature

   - Backend: habits CRUD, check-ins, streak calculation, stats API
   - Frontend: habits page, sidebar widget, create dialog
   - Features: daily/custom frequency, notes, icons, colors
   - Analytics: current/longest streak, completion rates
   "
   ```

4. **Push to remote**:
   ```bash
   git push
   ```

---

## Summary

This implementation plan builds a complete habits tracking system:

- **Backend**: 2 new tables, REST API, streak calculation, timezone support
- **Frontend**: Context-based state management, modular components, dual UI entry points
- **Features**: Daily/custom frequency, optional notes, visual customization (icons/colors)
- **Analytics**: Streaks, completion rates, calendar history
- **Integration**: Task-adjacent linking, sidebar widget, dedicated page

Total estimated implementation: ~15 tasks, each committed incrementally.
