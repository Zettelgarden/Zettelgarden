-- Populate user_stats with existing data
-- This backfills the stats table with current aggregate values

INSERT INTO user_stats (user_id, card_count, task_count, file_count, chat_message_count, llm_cost_usd, revenue_cents)
SELECT
    u.id,
    (SELECT COUNT(*) FROM cards c WHERE c.user_id = u.id AND c.is_deleted = FALSE),
    (SELECT COUNT(*) FROM tasks t WHERE t.user_id = u.id AND t.is_deleted = FALSE),
    (SELECT COUNT(*) FROM files f WHERE f.created_by = u.id),
    (SELECT COUNT(*) FROM chat_messages cm
        INNER JOIN chat_conversations cc ON cm.conversation_id = cc.id
        WHERE cc.user_id = u.id),
    COALESCE((SELECT SUM(l.cost_usd) FROM llm_query_log l WHERE l.user_id = u.id), 0),
    COALESCE((SELECT SUM(r.amount_cents) FROM revenue r WHERE r.user_id = u.id), 0)
FROM users u
ON CONFLICT (user_id) DO NOTHING;
