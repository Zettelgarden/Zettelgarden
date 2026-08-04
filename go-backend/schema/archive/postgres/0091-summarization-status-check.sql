-- Add CHECK constraint for summarizations.status to enforce valid enum values
-- Valid statuses: pending, processing, complete, failed
ALTER TABLE summarizations
ADD CONSTRAINT summarizations_status_check
CHECK (status IN ('pending', 'processing', 'complete', 'failed'));
