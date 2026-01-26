-- =============================================
-- TRIGGERS TO MAINTAIN user_stats TABLE
-- =============================================

-- Function to update user_stats when cards are inserted
CREATE OR REPLACE FUNCTION update_user_card_count_insert()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_stats (user_id, card_count, updated_at)
    VALUES (NEW.user_id, 1, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        card_count = user_stats.card_count + 1,
        updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update user_stats when cards are deleted or is_deleted changes
CREATE OR REPLACE FUNCTION update_user_card_count_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'DELETE') THEN
        INSERT INTO user_stats (user_id, card_count, updated_at)
        VALUES (OLD.user_id, -1, NOW())
        ON CONFLICT (user_id) DO UPDATE SET
            card_count = GREATEST(user_stats.card_count - 1, 0),
            updated_at = NOW();
        RETURN OLD;
    ELSIF (TG_OP = 'UPDATE') THEN
        -- Check if is_deleted status changed
        IF OLD.is_deleted = FALSE AND NEW.is_deleted = TRUE THEN
            -- Card was soft deleted - decrement count
            INSERT INTO user_stats (user_id, card_count, updated_at)
            VALUES (OLD.user_id, -1, NOW())
            ON CONFLICT (user_id) DO UPDATE SET
                card_count = GREATEST(user_stats.card_count - 1, 0),
                updated_at = NOW();
        ELSIF OLD.is_deleted = TRUE AND NEW.is_deleted = FALSE THEN
            -- Card was restored - increment count
            INSERT INTO user_stats (user_id, card_count, updated_at)
            VALUES (NEW.user_id, 1, NOW())
            ON CONFLICT (user_id) DO UPDATE SET
                card_count = user_stats.card_count + 1,
                updated_at = NOW();
        END IF;
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Triggers for cards
DROP TRIGGER IF EXISTS trg_cards_insert ON cards;
CREATE TRIGGER trg_cards_insert
AFTER INSERT ON cards
FOR EACH ROW
EXECUTE FUNCTION update_user_card_count_insert();

DROP TRIGGER IF EXISTS trg_cards_delete ON cards;
CREATE TRIGGER trg_cards_delete
AFTER UPDATE OF is_deleted OR DELETE ON cards
FOR EACH ROW
EXECUTE FUNCTION update_user_card_count_delete();

-- Function to update user_stats when tasks are inserted
CREATE OR REPLACE FUNCTION update_user_task_count_insert()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_stats (user_id, task_count, updated_at)
    VALUES (NEW.user_id, 1, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        task_count = user_stats.task_count + 1,
        updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update user_stats when tasks are deleted or is_deleted changes
CREATE OR REPLACE FUNCTION update_user_task_count_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'DELETE') THEN
        INSERT INTO user_stats (user_id, task_count, updated_at)
        VALUES (OLD.user_id, -1, NOW())
        ON CONFLICT (user_id) DO UPDATE SET
            task_count = GREATEST(user_stats.task_count - 1, 0),
            updated_at = NOW();
        RETURN OLD;
    ELSIF (TG_OP = 'UPDATE') THEN
        -- Check if is_deleted status changed
        IF OLD.is_deleted = FALSE AND NEW.is_deleted = TRUE THEN
            -- Task was soft deleted - decrement count
            INSERT INTO user_stats (user_id, task_count, updated_at)
            VALUES (OLD.user_id, -1, NOW())
            ON CONFLICT (user_id) DO UPDATE SET
                task_count = GREATEST(user_stats.task_count - 1, 0),
                updated_at = NOW();
        ELSIF OLD.is_deleted = TRUE AND NEW.is_deleted = FALSE THEN
            -- Task was restored - increment count
            INSERT INTO user_stats (user_id, task_count, updated_at)
            VALUES (NEW.user_id, 1, NOW())
            ON CONFLICT (user_id) DO UPDATE SET
                task_count = user_stats.task_count + 1,
                updated_at = NOW();
        END IF;
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Triggers for tasks
DROP TRIGGER IF EXISTS trg_tasks_insert ON tasks;
CREATE TRIGGER trg_tasks_insert
AFTER INSERT ON tasks
FOR EACH ROW
EXECUTE FUNCTION update_user_task_count_insert();

DROP TRIGGER IF EXISTS trg_tasks_delete ON tasks;
CREATE TRIGGER trg_tasks_delete
AFTER UPDATE OF is_deleted OR DELETE ON tasks
FOR EACH ROW
EXECUTE FUNCTION update_user_task_count_delete();

-- Function to update user_stats when files are inserted
CREATE OR REPLACE FUNCTION update_user_file_count_insert()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_stats (user_id, file_count, updated_at)
    VALUES (NEW.created_by, 1, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        file_count = user_stats.file_count + 1,
        updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update user_stats when files are deleted
CREATE OR REPLACE FUNCTION update_user_file_count_delete()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_stats (user_id, file_count, updated_at)
    VALUES (OLD.created_by, -1, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        file_count = GREATEST(user_stats.file_count - 1, 0),
        updated_at = NOW();
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Triggers for files
DROP TRIGGER IF EXISTS trg_files_insert ON files;
CREATE TRIGGER trg_files_insert
AFTER INSERT ON files
FOR EACH ROW
EXECUTE FUNCTION update_user_file_count_insert();

DROP TRIGGER IF EXISTS trg_files_delete ON files;
CREATE TRIGGER trg_files_delete
AFTER DELETE ON files
FOR EACH ROW
EXECUTE FUNCTION update_user_file_count_delete();

-- Function to update user_stats when chat messages are inserted
CREATE OR REPLACE FUNCTION update_user_chat_message_count()
RETURNS TRIGGER AS $$
DECLARE
    v_user_id INTEGER;
BEGIN
    -- Get the user_id from the conversation
    SELECT user_id INTO v_user_id
    FROM chat_conversations
    WHERE id = NEW.conversation_id;

    IF v_user_id IS NOT NULL THEN
        INSERT INTO user_stats (user_id, chat_message_count, updated_at)
        VALUES (v_user_id, 1, NOW())
        ON CONFLICT (user_id) DO UPDATE SET
            chat_message_count = user_stats.chat_message_count + 1,
            updated_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for chat_messages
DROP TRIGGER IF EXISTS trg_chat_messages_insert ON chat_messages;
CREATE TRIGGER trg_chat_messages_insert
AFTER INSERT ON chat_messages
FOR EACH ROW
EXECUTE FUNCTION update_user_chat_message_count();

-- Function to update user_stats when llm_query_log entries are added
CREATE OR REPLACE FUNCTION update_user_llm_cost()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_stats (user_id, llm_cost_usd, updated_at)
    VALUES (NEW.user_id, NEW.cost_usd, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        llm_cost_usd = user_stats.llm_cost_usd + NEW.cost_usd,
        updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for llm_query_log
DROP TRIGGER IF EXISTS trg_llm_query_log_insert ON llm_query_log;
CREATE TRIGGER trg_llm_query_log_insert
AFTER INSERT ON llm_query_log
FOR EACH ROW
EXECUTE FUNCTION update_user_llm_cost();

-- Function to update user_stats when revenue is recorded
CREATE OR REPLACE FUNCTION update_user_revenue()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_stats (user_id, revenue_cents, updated_at)
    VALUES (NEW.user_id, NEW.amount_cents, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        revenue_cents = user_stats.revenue_cents + NEW.amount_cents,
        updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for revenue
DROP TRIGGER IF EXISTS trg_revenue_insert ON revenue;
CREATE TRIGGER trg_revenue_insert
AFTER INSERT ON revenue
FOR EACH ROW
EXECUTE FUNCTION update_user_revenue();
