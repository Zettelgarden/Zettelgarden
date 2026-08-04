DO $$
BEGIN
    -- Check if the constraint exists using a query on pg_constraint
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'tags_name_key' AND conrelid = 'tags'::regclass
    ) THEN
        -- If it exists, drop the constraint
        ALTER TABLE tags DROP CONSTRAINT tags_name_key;
    END IF;
END $$;
