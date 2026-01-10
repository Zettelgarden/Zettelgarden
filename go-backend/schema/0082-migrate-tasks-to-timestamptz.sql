-- Migrate task date columns from TIMESTAMP to TIMESTAMPTZ
-- Existing naive timestamps are assumed to be UTC

ALTER TABLE tasks
    ALTER COLUMN scheduled_date TYPE TIMESTAMPTZ USING scheduled_date AT TIME ZONE 'UTC';

ALTER TABLE tasks
    ALTER COLUMN due_date TYPE TIMESTAMPTZ USING due_date AT TIME ZONE 'UTC';

ALTER TABLE tasks
    ALTER COLUMN reminder_time TYPE TIMESTAMPTZ USING reminder_time AT TIME ZONE 'UTC';

ALTER TABLE tasks
    ALTER COLUMN completed_at TYPE TIMESTAMPTZ USING completed_at AT TIME ZONE 'UTC';

ALTER TABLE tasks
    ALTER COLUMN created_at TYPE TIMESTAMPTZ USING created_at AT TIME ZONE 'UTC';

ALTER TABLE tasks
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ USING updated_at AT TIME ZONE 'UTC';
