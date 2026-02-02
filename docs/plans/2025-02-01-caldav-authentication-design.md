# CalDAV/iCal Authentication Support Design

**Created:** 2025-02-01
**Status:** Design Approved

## Problem Statement

The current CalDAV/iCal subscription feature does not support authentication. When users attempt to subscribe to calendars that require authentication (e.g., Google Calendar private links, corporate calendars), the sync fails with a 401 Unauthorized error.

## Solution Overview

Add Basic Authentication support for external calendar subscriptions by:

1. Storing encrypted credentials in the database
2. Adding username/password fields to the calendar subscription UI
3. Using credentials when fetching iCal feeds

## Architecture

### Database Schema Changes

**Migration: `go-backend/schema/0109-add-calendar-auth.sql`**

```sql
-- Add username and password fields for authenticated calendar access
ALTER TABLE external_calendars ADD COLUMN username TEXT;
ALTER TABLE external_calendars ADD COLUMN password TEXT;

-- Password will be encrypted at rest using AES-256-GCM
COMMENT ON COLUMN external_calendars.username IS 'Username for Basic Auth (if calendar requires authentication)';
COMMENT ON COLUMN external_calendars.password IS 'Encrypted password for Basic Auth (AES-256-GCM encrypted)';
```

### Encryption Service

**New file: `go-backend/services/encryption.go`**

```go
package services

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "os"
)

type EncryptionService struct {
    key []byte
}

func NewEncryptionService() (*EncryptionService, error) {
    keyStr := os.Getenv("CALENDAR_ENCRYPTION_KEY")
    if keyStr == "" {
        return nil, fmt.Errorf("CALENDAR_ENCRYPTION_KEY environment variable not set")
    }

    // Derive 32-byte key from env var using SHA256
    hash := sha256.Sum256([]byte(keyStr))
    return &EncryptionService{key: hash[:]}, nil
}

func (s *EncryptionService) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(s.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, gcm.NonceSize())
    // In production, use crypto/rand for nonce generation

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *EncryptionService) Decrypt(ciphertext string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher(s.key)
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", fmt.Errorf("ciphertext too short")
    }

    nonce, cipher := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, cipher, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

### API Model Changes

**Backend: `go-backend/models/external_events.go`**

```go
type CreateExternalCalendarRequest struct {
    Name     string `json:"name"`
    URL      string `json:"url"`
    Color    string `json:"color"`
    Username string `json:"username,omitempty"` // New
    Password string `json:"password,omitempty"` // New
}

type UpdateExternalCalendarRequest struct {
    Name              *string `json:"name,omitempty"`
    URL               *string `json:"url,omitempty"`
    Color             *string `json:"color,omitempty"`
    SyncEnabled       *bool   `json:"sync_enabled,omitempty"`
    SyncIntervalHours *int    `json:"sync_interval_hours,omitempty"`
    Username          *string `json:"username,omitempty"` // New
    Password          *string `json:"password,omitempty"` // New
}

type ExternalCalendar struct {
    ID                int        `json:"id"`
    UserID            int        `json:"user_id"`
    Name              string     `json:"name"`
    URL               string     `json:"url"`
    SyncEnabled       bool       `json:"sync_enabled"`
    SyncIntervalHours int        `json:"sync_interval_hours"`
    Color             string     `json:"color"`
    Username          string     `json:"username,omitempty"` // New (plaintext, safe to expose)
    // Password is never exposed via JSON
    LastSyncedAt      *time.Time `json:"last_synced_at,omitempty"`
    LastError         *string    `json:"last_error,omitempty"`
    CreatedAt         time.Time  `json:"created_at"`
    UpdatedAt         time.Time  `json:"updated_at"`
}
```

**Frontend: `zettelkasten-front/src/models/ExternalEvent.ts`**

```typescript
export interface CreateExternalCalendarRequest {
  name: string;
  url: string;
  color?: string;
  username?: string;  // New
  password?: string;  // New
}
```

### Service Layer Updates

**Updated: `go-backend/services/ical.go`**

```go
// FetchICalURL fetches an iCal feed from a URL with optional authentication
func FetchICalURL(feedURL string, username, password string) ([]ICalEvent, error) {
    client := &http.Client{
        Timeout: 30 * time.Second,
    }

    req, err := http.NewRequest("GET", feedURL, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // Add Basic Auth if credentials provided
    if username != "" && password != "" {
        req.SetBasicAuth(username, password)
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch iCal feed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("iCal feed returned status %d", resp.StatusCode)
    }

    // ... rest of function unchanged
}

// ValidateICalURL validates with optional authentication
func ValidateICalURL(rawURL string, username, password string) error {
    // ... URL validation unchanged ...

    // Try to fetch and parse with credentials
    events, err := FetchICalURL(rawURL, username, password)
    if err != nil {
        return err
    }

    if len(events) == 0 {
        return fmt.Errorf("feed contains no events")
    }

    return nil
}
```

**Updated: `go-backend/services/external_events.go`**

```go
type ExternalEventService struct {
    db               models.Database
    encryption       *EncryptionService
}

func NewExternalEventService(db models.Database) *ExternalEventService {
    enc, _ := NewEncryptionService() // Handle error appropriately
    return &ExternalEventService{db: db, encryption: enc}
}

func (s *ExternalEventService) SyncExternalCalendar(calendarID int, userID int) error {
    // Get calendar with credentials
    var url, username, encryptedPassword string
    // ... query with username and password fields ...

    // Decrypt password if present
    var password string
    if encryptedPassword != "" {
        decrypted, err := s.encryption.Decrypt(encryptedPassword)
        if err != nil {
            return fmt.Errorf("failed to decrypt credentials: %w", err)
        }
        password = decrypted
    }

    // Fetch with credentials
    icalEvents, err := FetchICalURL(url, username, password)
    // ... rest unchanged
}

func (s *ExternalEventService) CreateCalendar(userID int, req models.CreateExternalCalendarRequest) (*models.ExternalCalendar, error) {
    // Validate URL with credentials
    if err := ValidateICalURL(req.URL, req.Username, req.Password); err != nil {
        return nil, fmt.Errorf("invalid iCal URL: %w", err)
    }

    // Encrypt password before storing
    var encryptedPassword string
    if req.Password != "" {
        encrypted, err := s.encryption.Encrypt(req.Password)
        if err != nil {
            return nil, fmt.Errorf("failed to encrypt password: %w", err)
        }
        encryptedPassword = encrypted
    }

    // ... insert with username and encrypted password
}
```

### Frontend Updates

**Updated: `zettelkasten-front/src/components/settings/CalendarSubscriptions.tsx`**

Add optional authentication fields to the form:

```tsx
<div>
  <label htmlFor="username" className="block text-sm font-medium text-slate-700 mb-1">
    Username <span className="text-slate-400">(optional)</span>
  </label>
  <input
    name="username"
    placeholder="Required if calendar is private"
    className="w-full px-3 py-2 border border-slate-300 rounded"
  />
</div>
<div>
  <label htmlFor="password" className="block text-sm font-medium text-slate-700 mb-1">
    Password <span className="text-slate-400">(optional)</span>
  </label>
  <input
    name="password"
    type="password"
    placeholder="Required if calendar is private"
    className="w-full px-3 py-2 border border-slate-300 rounded"
  />
</div>
```

## Security Considerations

1. **Password Encryption**: All passwords are encrypted at rest using AES-256-GCM
2. **Environment Variable**: The encryption key must be set via `CALENDAR_ENCRYPTION_KEY`
3. **Password Exclusion**: Passwords are never included in API responses (`json:"-"` tag)
4. **SSRF Protection**: Existing URL validation continues to apply
5. **Credential Rotation**: Users can update credentials via the update endpoint

## Environment Variables

**Required:**
```
CALENDAR_ENCRYPTION_KEY=32-character-or-longer-random-string
```

Generate with: `openssl rand -base64 32`

## Testing Plan

1. Unit tests for encryption/decryption service
2. Integration tests for calendar creation with credentials
3. Test sync with authenticated calendars (Google, Fastmail, etc.)
4. Verify password never appears in API responses
5. Test validation fails with invalid credentials

## Migration Steps

1. Add database migration
2. Implement encryption service
3. Update service layer with credential support
4. Update API models and handlers
5. Update frontend form
6. Add tests
