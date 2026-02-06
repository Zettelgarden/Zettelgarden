-- Add external tracking columns to tasks table
-- This allows tasks imported from VTODO components to be tracked and updated
-- when the external calendar feed is re-synced.

-- Add external_uid to track the source VTODO's UID for upsert operations
ALTER TABLE tasks ADD COLUMN external_uid TEXT;

-- Add external_calendar_id to link the task to its source calendar subscription
ALTER TABLE tasks ADD COLUMN external_calendar_id INTEGER REFERENCES external_calendars(id) ON DELETE SET NULL;

-- Add index on external_uid for efficient upsert lookups
CREATE INDEX idx_tasks_external_uid ON tasks(external_uid) WHERE external_uid IS NOT NULL;

-- Add comment to document the purpose
COMMENT ON COLUMN tasks.external_uid IS 'UID from VTODO component in iCal feed, used for upsert identification';
COMMENT ON COLUMN tasks.external_calendar_id IS 'Reference to external calendar subscription this task was imported from';
