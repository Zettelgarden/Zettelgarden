--
-- PostgreSQL database dump
--

\restrict kIb9LPvvUVEGj8cyn3RIK6SB4bbMKEaScLPEx7CKV3QDGXfBIoG7DrRAcdK5Vec

-- Dumped from database version 16.11 (Debian 16.11-1.pgdg13+1)
-- Dumped by pg_dump version 16.11 (Debian 16.11-1.pgdg13+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: delete_notification(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.delete_notification() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- Delete notification for this user and source
    -- Note: TG_TABLE_NAME returns schema-qualified name, so we map it to source_type
    DELETE FROM notifications
    WHERE user_id = OLD.user_id
      AND source_id = OLD.id
      AND source_type = CASE
          WHEN TG_TABLE_NAME LIKE '%emails' THEN 'email'
          WHEN TG_TABLE_NAME LIKE '%rss_articles' THEN 'rss'
          WHEN TG_TABLE_NAME LIKE '%tasks' THEN 'task'
          ELSE TG_TABLE_NAME
        END;
    RETURN OLD;
END;
$$;


--
-- Name: FUNCTION delete_notification(); Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON FUNCTION public.delete_notification() IS 'Deletes corresponding notification rows when source record is deleted. Automatically determines source_type from table name.';


--
-- Name: notify_llm_job_insert(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.notify_llm_job_insert() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
	PERFORM pg_notify('llm_job_queue', 'new_job');
	RETURN NEW;
END;
$$;


--
-- Name: FUNCTION notify_llm_job_insert(); Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON FUNCTION public.notify_llm_job_insert() IS 'Sends PostgreSQL NOTIFY event when a new job is added to the queue';


--
-- Name: sync_email_notification(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.sync_email_notification() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    notification_title TEXT;
    notification_preview TEXT;
    notification_importance INT;
    notification_tags TEXT[];
BEGIN
    -- Only create notifications for unprocessed and triaged emails (not archived)
    IF NEW.status NOT IN ('unprocessed', 'triaged') THEN
        -- If archived, delete the notification
        IF NEW.status = 'archived' THEN
            DELETE FROM notifications WHERE user_id = NEW.user_id AND source_type = 'email' AND source_id = NEW.id;
            RETURN NEW;
        END IF;
        RETURN NEW;
    END IF;

    -- Build notification title and preview
    notification_title := COALESCE(NEW.subject, '(No subject)');
    notification_preview := LEFT(COALESCE(NEW.body_text, ''), 200);

    -- Calculate importance score
    -- Importance scores: unprocessed=10 (highest priority), triaged=5 (medium priority)
    -- This ensures new unprocessed emails appear at top of notification list
    IF NEW.status = 'unprocessed' THEN
        notification_importance := 10;
    ELSE
        notification_importance := 5;  -- triaged
    END IF;

    -- Build filter tags
    notification_tags := ARRAY[NEW.status];

    -- Insert or update notification
    INSERT INTO notifications (user_id, source_type, source_id, title, preview, timestamp, importance_score, filter_tags)
    VALUES (NEW.user_id, 'email', NEW.id, notification_title, notification_preview,
            COALESCE(NEW.received_at, NEW.created_at), notification_importance, notification_tags)
    ON CONFLICT (user_id, source_type, source_id)
    DO UPDATE SET
        title = EXCLUDED.title,
        preview = EXCLUDED.preview,
        timestamp = EXCLUDED.timestamp,
        importance_score = EXCLUDED.importance_score,
        filter_tags = EXCLUDED.filter_tags,
        updated_at = NOW();

    RETURN NEW;
END;
$$;


--
-- Name: sync_rss_notification(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.sync_rss_notification() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    feed_record RECORD;
    notification_title TEXT;
    notification_preview TEXT;
    notification_importance INT;
    notification_tags TEXT[];
BEGIN
    -- Only create notifications for starred articles or priority feed articles
    SELECT * INTO feed_record FROM rss_feeds WHERE id = NEW.feed_id;

    -- Safety check: if feed doesn't exist, skip notification processing
    IF feed_record IS NULL THEN
        RETURN NEW;
    END IF;

    IF NOT (NEW.is_starred = TRUE OR (feed_record.priority = TRUE AND NEW.read = FALSE)) THEN
        -- If no longer starred, delete the notification
        IF NEW.is_starred = FALSE THEN
            DELETE FROM notifications WHERE user_id = NEW.user_id AND source_type = 'rss' AND source_id = NEW.id;
        END IF;
        RETURN NEW;
    END IF;

    -- Build notification title and preview
    notification_title := NEW.title;
    notification_preview := LEFT(COALESCE(NEW.content, ''), 200);

    -- Calculate importance score
    -- Starred articles: 10 (highest priority)
    -- Priority feed unread articles: 5 (medium priority)
    IF NEW.is_starred = TRUE THEN
        notification_importance := 10;
    ELSIF feed_record.priority = TRUE THEN
        notification_importance := 5;
    ELSE
        notification_importance := 0;
    END IF;

    -- Build filter tags
    notification_tags := ARRAY[]::TEXT[];
    IF NEW.is_starred = TRUE THEN
        notification_tags := array_append(notification_tags, 'starred');
    END IF;
    IF feed_record.priority = TRUE THEN
        notification_tags := array_append(notification_tags, 'priority');
    END IF;

    -- Insert or update notification
    INSERT INTO notifications (user_id, source_type, source_id, title, preview, timestamp, importance_score, filter_tags)
    VALUES (NEW.user_id, 'rss', NEW.id, notification_title, notification_preview,
            COALESCE(NEW.published_at, NEW.fetched_at), notification_importance, notification_tags)
    ON CONFLICT (user_id, source_type, source_id)
    DO UPDATE SET
        title = EXCLUDED.title,
        preview = EXCLUDED.preview,
        timestamp = EXCLUDED.timestamp,
        importance_score = EXCLUDED.importance_score,
        filter_tags = EXCLUDED.filter_tags,
        updated_at = NOW();

    RETURN NEW;
END;
$$;


--
-- Name: update_conversation_timestamp(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_conversation_timestamp() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE chat_conversations
    SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.conversation_id;
    RETURN NEW;
END;
$$;


--
-- Name: update_llm_jobs_updated_at(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_llm_jobs_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
	NEW.updated_at = NOW();
	RETURN NEW;
END;
$$;


--
-- Name: update_user_card_count_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_user_card_count_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
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
$$;


--
-- Name: update_user_card_count_insert(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_user_card_count_insert() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO user_stats (user_id, card_count, updated_at)
    VALUES (NEW.user_id, 1, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        card_count = user_stats.card_count + 1,
        updated_at = NOW();
    RETURN NEW;
END;
$$;


--
-- Name: update_user_chat_message_count(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_user_chat_message_count() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
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
$$;


--
-- Name: update_user_file_count_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_user_file_count_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO user_stats (user_id, file_count, updated_at)
    VALUES (OLD.created_by, -1, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        file_count = GREATEST(user_stats.file_count - 1, 0),
        updated_at = NOW();
    RETURN OLD;
END;
$$;


--
-- Name: update_user_file_count_insert(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_user_file_count_insert() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO user_stats (user_id, file_count, updated_at)
    VALUES (NEW.created_by, 1, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        file_count = user_stats.file_count + 1,
        updated_at = NOW();
    RETURN NEW;
END;
$$;


--
-- Name: update_user_llm_cost(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_user_llm_cost() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO user_stats (user_id, llm_cost_usd, updated_at)
    VALUES (NEW.user_id, NEW.cost_usd, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        llm_cost_usd = user_stats.llm_cost_usd + NEW.cost_usd,
        updated_at = NOW();
    RETURN NEW;
END;
$$;


--
-- Name: update_user_revenue(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_user_revenue() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO user_stats (user_id, revenue_cents, updated_at)
    VALUES (NEW.user_id, NEW.amount_cents, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        revenue_cents = user_stats.revenue_cents + NEW.amount_cents,
        updated_at = NOW();
    RETURN NEW;
END;
$$;


--
-- Name: update_user_task_count_delete(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_user_task_count_delete() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
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
$$;


--
-- Name: update_user_task_count_insert(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_user_task_count_insert() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO user_stats (user_id, task_count, updated_at)
    VALUES (NEW.user_id, 1, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        task_count = user_stats.task_count + 1,
        updated_at = NOW();
    RETURN NEW;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: admin_audit_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.admin_audit_log (
    id integer NOT NULL,
    admin_user_id integer NOT NULL,
    action text NOT NULL,
    target_type text NOT NULL,
    target_id integer,
    details jsonb DEFAULT '{}'::jsonb NOT NULL,
    ip_address inet,
    user_agent text,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE admin_audit_log; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.admin_audit_log IS 'Audit log of all admin actions for security and compliance';


--
-- Name: COLUMN admin_audit_log.admin_user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.admin_audit_log.admin_user_id IS 'The admin user who performed the action';


--
-- Name: COLUMN admin_audit_log.action; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.admin_audit_log.action IS 'The action performed (e.g., user.update, mailing_list.unsubscribe)';


--
-- Name: COLUMN admin_audit_log.target_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.admin_audit_log.target_type IS 'Type of entity affected (e.g., user, mailing_list)';


--
-- Name: COLUMN admin_audit_log.target_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.admin_audit_log.target_id IS 'ID of the affected entity';


--
-- Name: COLUMN admin_audit_log.details; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.admin_audit_log.details IS 'Additional context about the action (JSON)';


--
-- Name: COLUMN admin_audit_log.ip_address; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.admin_audit_log.ip_address IS 'IP address of the admin for security investigations';


--
-- Name: admin_audit_log_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.admin_audit_log_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: admin_audit_log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.admin_audit_log_id_seq OWNED BY public.admin_audit_log.id;


--
-- Name: agent_activity_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.agent_activity_log (
    id integer NOT NULL,
    agent_id integer,
    action character varying(50) NOT NULL,
    target_type character varying(50) NOT NULL,
    target_id integer,
    details jsonb,
    created_at timestamp without time zone DEFAULT now()
);


--
-- Name: TABLE agent_activity_log; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.agent_activity_log IS 'Audit log of all AI agent actions';


--
-- Name: COLUMN agent_activity_log.agent_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.agent_activity_log.agent_id IS 'The agent user who performed the action (NULL if agent was deleted/revoked)';


--
-- Name: COLUMN agent_activity_log.action; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.agent_activity_log.action IS 'Type of action performed (e.g., create_card, update_card, delete_card)';


--
-- Name: COLUMN agent_activity_log.target_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.agent_activity_log.target_type IS 'Type of target entity (e.g., card, task, file)';


--
-- Name: COLUMN agent_activity_log.target_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.agent_activity_log.target_id IS 'ID of the target entity (nullable for actions without specific targets)';


--
-- Name: COLUMN agent_activity_log.details; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.agent_activity_log.details IS 'Additional JSON details about the action';


--
-- Name: COLUMN agent_activity_log.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.agent_activity_log.created_at IS 'When the action was performed';


--
-- Name: agent_activity_log_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.agent_activity_log_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: agent_activity_log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.agent_activity_log_id_seq OWNED BY public.agent_activity_log.id;


--
-- Name: api_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_keys (
    id integer NOT NULL,
    user_id integer NOT NULL,
    name character varying(255) NOT NULL,
    key_hash character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    last_used_at timestamp with time zone,
    revoked_at timestamp with time zone,
    is_active boolean DEFAULT true,
    description text,
    CONSTRAINT revoked_at_required_when_inactive CHECK ((((is_active = true) AND (revoked_at IS NULL)) OR ((is_active = false) AND (revoked_at IS NOT NULL))))
);


--
-- Name: api_keys_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.api_keys_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: api_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.api_keys_id_seq OWNED BY public.api_keys.id;


--
-- Name: audit_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_events (
    id integer NOT NULL,
    user_id integer NOT NULL,
    entity_id integer NOT NULL,
    entity_type text NOT NULL,
    action text NOT NULL,
    details jsonb NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: audit_events_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.audit_events_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: audit_events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.audit_events_id_seq OWNED BY public.audit_events.id;


--
-- Name: backlinks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.backlinks (
    source_id text,
    target_id text,
    created_at timestamp without time zone,
    updated_at timestamp without time zone,
    source_id_int integer,
    target_id_int integer
);


--
-- Name: card_chunks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.card_chunks (
    id integer NOT NULL,
    card_pk integer,
    user_id integer,
    chunk_text text,
    chunk_id integer
);


--
-- Name: card_chunks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.card_chunks_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: card_chunks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.card_chunks_id_seq OWNED BY public.card_chunks.id;


--
-- Name: card_embeddings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.card_embeddings (
    id integer NOT NULL,
    card_pk integer,
    user_id integer,
    chunk integer
);


--
-- Name: card_embeddings_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.card_embeddings_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: card_embeddings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.card_embeddings_id_seq OWNED BY public.card_embeddings.id;


--
-- Name: card_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.card_tags (
    card_pk integer NOT NULL,
    tag_id integer NOT NULL
);


--
-- Name: card_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.card_templates (
    id integer NOT NULL,
    user_id integer NOT NULL,
    title text NOT NULL,
    body text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    name text DEFAULT ''::text NOT NULL
);


--
-- Name: card_templates_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.card_templates_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: card_templates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.card_templates_id_seq OWNED BY public.card_templates.id;


--
-- Name: card_views; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.card_views (
    id integer NOT NULL,
    card_pk integer,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    user_id integer
);


--
-- Name: card_views_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.card_views_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: card_views_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.card_views_id_seq OWNED BY public.card_views.id;


--
-- Name: cards; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.cards (
    id integer NOT NULL,
    card_id text,
    title text,
    body text,
    is_reference integer DEFAULT 0,
    link text,
    created_at timestamp without time zone,
    updated_at timestamp without time zone,
    user_id integer,
    is_deleted boolean DEFAULT false,
    parent_id integer,
    is_literature_card boolean DEFAULT false,
    is_flashcard boolean DEFAULT false,
    flashcard_state text,
    flashcard_reps integer DEFAULT 0,
    flashcard_lapses integer DEFAULT 0,
    flashcard_last_review timestamp without time zone,
    flashcard_due timestamp without time zone,
    flashcard_stability real DEFAULT 0,
    flashcard_difficulty real DEFAULT 0,
    card_schema_id integer,
    structured_data jsonb,
    created_by_agent_id integer
);


--
-- Name: COLUMN cards.created_by_agent_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.cards.created_by_agent_id IS 'For cards created by AI agents: the agent user who created this card';


--
-- Name: cards_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.cards_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cards_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.cards_id_seq OWNED BY public.cards.id;


--
-- Name: categories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.categories (
    id integer NOT NULL,
    user_id integer,
    name text,
    description text,
    regex text,
    is_active boolean DEFAULT true,
    created_by integer,
    updated_by integer,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: categories_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.categories_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: categories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.categories_id_seq OWNED BY public.categories.id;


--
-- Name: chat_conversations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.chat_conversations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id integer NOT NULL,
    title text,
    model text DEFAULT 'gpt-4o-mini'::text NOT NULL,
    system_prompt text,
    starred boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    primary_card_id integer
);


--
-- Name: chat_instructions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.chat_instructions (
    id integer NOT NULL,
    user_id integer NOT NULL,
    instructions text DEFAULT ''::text,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


--
-- Name: chat_instructions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.chat_instructions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: chat_instructions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.chat_instructions_id_seq OWNED BY public.chat_instructions.id;


--
-- Name: chat_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.chat_messages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    conversation_id uuid NOT NULL,
    role text NOT NULL,
    content text,
    tool_calls jsonb,
    tool_call_id text,
    sequence_number integer NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    referenced_cards jsonb,
    status text DEFAULT 'completed'::text NOT NULL,
    CONSTRAINT chat_messages_role_check CHECK ((role = ANY (ARRAY['user'::text, 'assistant'::text, 'system'::text, 'tool'::text]))),
    CONSTRAINT chat_messages_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'processing'::text, 'completed'::text, 'failed'::text])))
);


--
-- Name: chat_tool_calls; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.chat_tool_calls (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id integer NOT NULL,
    conversation_id uuid,
    message_id uuid,
    tool_name text NOT NULL,
    tool_arguments jsonb,
    tool_result jsonb,
    execution_time_ms integer,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: chat_usage_quotas; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.chat_usage_quotas (
    id integer NOT NULL,
    user_id integer NOT NULL,
    quota_type text NOT NULL,
    current_usage integer DEFAULT 0,
    max_limit integer NOT NULL,
    reset_date date DEFAULT CURRENT_DATE,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chat_usage_quotas_quota_type_check CHECK ((quota_type = ANY (ARRAY['messages_per_day'::text, 'tool_calls_per_day'::text, 'conversations_per_day'::text])))
);


--
-- Name: chat_usage_quotas_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.chat_usage_quotas_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: chat_usage_quotas_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.chat_usage_quotas_id_seq OWNED BY public.chat_usage_quotas.id;


--
-- Name: email_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_accounts (
    id integer NOT NULL,
    user_id integer NOT NULL,
    email_address text NOT NULL,
    jmap_server_url text DEFAULT 'https://api.fastmail.com/jmap/session'::text NOT NULL,
    is_active boolean DEFAULT true,
    last_sync_at timestamp with time zone,
    sync_status text DEFAULT 'active'::text,
    jmap_state text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    imap_server text DEFAULT 'imap.fastmail.com:993'::text,
    imap_server_type text DEFAULT 'imap'::text,
    imap_uid bigint,
    imap_uid_validity bigint,
    app_password_encrypted text,
    CONSTRAINT email_accounts_sync_status_check CHECK ((sync_status = ANY (ARRAY['active'::text, 'syncing'::text, 'error'::text, 'disabled'::text])))
);


--
-- Name: TABLE email_accounts; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.email_accounts IS 'Stores Fastmail account credentials for email synchronization';


--
-- Name: COLUMN email_accounts.jmap_server_url; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_accounts.jmap_server_url IS 'JMAP server endpoint URL';


--
-- Name: COLUMN email_accounts.is_active; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_accounts.is_active IS 'Whether the account is actively syncing';


--
-- Name: COLUMN email_accounts.sync_status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_accounts.sync_status IS 'Current sync status: active, syncing, error, or disabled';


--
-- Name: COLUMN email_accounts.jmap_state; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_accounts.jmap_state IS 'JMAP state token for incremental synchronization';


--
-- Name: COLUMN email_accounts.imap_server; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_accounts.imap_server IS 'IMAP server address (e.g., imap.fastmail.com:993)';


--
-- Name: COLUMN email_accounts.imap_server_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_accounts.imap_server_type IS 'Type of email server (default: imap)';


--
-- Name: COLUMN email_accounts.imap_uid; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_accounts.imap_uid IS 'Last IMAP message UID synced for this account (for incremental sync)';


--
-- Name: COLUMN email_accounts.imap_uid_validity; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_accounts.imap_uid_validity IS 'IMAP UIDVALIDITY value to detect mailbox resets';


--
-- Name: COLUMN email_accounts.app_password_encrypted; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_accounts.app_password_encrypted IS 'Encrypted app password for authentication';


--
-- Name: email_accounts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.email_accounts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: email_accounts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.email_accounts_id_seq OWNED BY public.email_accounts.id;


--
-- Name: email_attachments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_attachments (
    id integer NOT NULL,
    user_id integer NOT NULL,
    email_id integer NOT NULL,
    file_id integer,
    filename text NOT NULL,
    content_type text,
    size bigint,
    s3_key text,
    thumbnail_path text,
    content_id text,
    is_inline boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: TABLE email_attachments; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.email_attachments IS 'Stores email attachment metadata and links to files in S3';


--
-- Name: COLUMN email_attachments.email_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_attachments.email_id IS 'Reference to the email this attachment belongs to';


--
-- Name: COLUMN email_attachments.file_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_attachments.file_id IS 'Reference to the file record if saved to file vault';


--
-- Name: COLUMN email_attachments.filename; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_attachments.filename IS 'Original filename of the attachment';


--
-- Name: COLUMN email_attachments.content_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_attachments.content_type IS 'MIME content type (e.g., image/jpeg, application/pdf)';


--
-- Name: COLUMN email_attachments.size; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_attachments.size IS 'Size of the attachment in bytes';


--
-- Name: COLUMN email_attachments.s3_key; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_attachments.s3_key IS 'S3 key where the attachment is stored';


--
-- Name: COLUMN email_attachments.thumbnail_path; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_attachments.thumbnail_path IS 'S3 key for thumbnail (for images)';


--
-- Name: COLUMN email_attachments.content_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_attachments.content_id IS 'Content-ID header for inline attachments';


--
-- Name: COLUMN email_attachments.is_inline; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_attachments.is_inline IS 'Whether this is an inline attachment (e.g., embedded image)';


--
-- Name: email_attachments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.email_attachments_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: email_attachments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.email_attachments_id_seq OWNED BY public.email_attachments.id;


--
-- Name: email_card_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_card_links (
    id integer NOT NULL,
    email_id integer NOT NULL,
    card_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: TABLE email_card_links; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.email_card_links IS 'Links emails to cards created from email content';


--
-- Name: COLUMN email_card_links.email_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_card_links.email_id IS 'Reference to the source email';


--
-- Name: COLUMN email_card_links.card_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_card_links.card_id IS 'Reference to the created card';


--
-- Name: email_card_links_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.email_card_links_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: email_card_links_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.email_card_links_id_seq OWNED BY public.email_card_links.id;


--
-- Name: email_fact_junction; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_fact_junction (
    id integer NOT NULL,
    user_id integer NOT NULL,
    email_id integer NOT NULL,
    fact_id integer NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


--
-- Name: TABLE email_fact_junction; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.email_fact_junction IS 'Links facts extracted from emails to their source emails';


--
-- Name: COLUMN email_fact_junction.email_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_fact_junction.email_id IS 'Reference to the source email';


--
-- Name: COLUMN email_fact_junction.fact_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_fact_junction.fact_id IS 'Reference to the extracted fact';


--
-- Name: email_fact_junction_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.email_fact_junction_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: email_fact_junction_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.email_fact_junction_id_seq OWNED BY public.email_fact_junction.id;


--
-- Name: email_triage_decisions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_triage_decisions (
    id integer NOT NULL,
    email_id integer NOT NULL,
    decision text NOT NULL,
    confidence double precision NOT NULL,
    reasoning text,
    is_auto_executed boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT email_triage_decisions_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
    CONSTRAINT email_triage_decisions_decision_check CHECK ((decision = ANY (ARRAY['archive'::text, 'delete'::text, 'keep'::text, 'convert_to_card'::text])))
);


--
-- Name: TABLE email_triage_decisions; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.email_triage_decisions IS 'Stores AI-powered triage recommendations for emails';


--
-- Name: COLUMN email_triage_decisions.decision; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_triage_decisions.decision IS 'Recommended action: archive, delete, keep, or convert_to_card';


--
-- Name: COLUMN email_triage_decisions.confidence; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_triage_decisions.confidence IS 'Confidence score between 0 and 1';


--
-- Name: COLUMN email_triage_decisions.reasoning; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_triage_decisions.reasoning IS 'AI-generated explanation for the decision';


--
-- Name: COLUMN email_triage_decisions.is_auto_executed; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.email_triage_decisions.is_auto_executed IS 'Whether the decision was automatically applied';


--
-- Name: email_triage_decisions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.email_triage_decisions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: email_triage_decisions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.email_triage_decisions_id_seq OWNED BY public.email_triage_decisions.id;


--
-- Name: emails; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.emails (
    id integer NOT NULL,
    user_id integer NOT NULL,
    email_account_id integer,
    message_id text NOT NULL,
    thread_id text,
    subject text,
    from_address text,
    from_name text,
    to_addresses text,
    body_text text,
    body_html text,
    received_at timestamp with time zone,
    folder text DEFAULT 'Inbox'::text,
    status text DEFAULT 'unprocessed'::text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    imap_uid bigint,
    is_read boolean DEFAULT false,
    card_id integer,
    CONSTRAINT emails_status_check CHECK ((status = ANY (ARRAY['unprocessed'::text, 'triaged'::text, 'reviewed'::text, 'archived'::text, 'deleted'::text, 'converted'::text])))
);


--
-- Name: TABLE emails; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.emails IS 'Stores synchronized email messages from Fastmail';


--
-- Name: COLUMN emails.message_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.emails.message_id IS 'JMAP message identifier';


--
-- Name: COLUMN emails.thread_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.emails.thread_id IS 'JMAP thread identifier for conversation tracking';


--
-- Name: COLUMN emails.folder; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.emails.folder IS 'Email folder/category (e.g., Inbox, Archive)';


--
-- Name: COLUMN emails.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.emails.status IS 'Processing status: unprocessed, triaged, reviewed, archived, deleted, or converted';


--
-- Name: COLUMN emails.imap_uid; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.emails.imap_uid IS 'IMAP message UID for this email, used for folder operations like archive';


--
-- Name: COLUMN emails.is_read; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.emails.is_read IS 'Whether the email has been read (synced with IMAP \Seen flag)';


--
-- Name: COLUMN emails.card_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.emails.card_id IS 'ID of the card created from this email, if converted';


--
-- Name: emails_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.emails_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: emails_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.emails_id_seq OWNED BY public.emails.id;


--
-- Name: entities; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.entities (
    id integer NOT NULL,
    user_id integer,
    name text,
    description text,
    type text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    card_pk integer
);


--
-- Name: entities_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.entities_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: entities_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.entities_id_seq OWNED BY public.entities.id;


--
-- Name: entity_card_junction; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.entity_card_junction (
    id integer NOT NULL,
    user_id integer,
    entity_id integer,
    card_pk integer,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    chunk_id integer
);


--
-- Name: entity_card_junction_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.entity_card_junction_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: entity_card_junction_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.entity_card_junction_id_seq OWNED BY public.entity_card_junction.id;


--
-- Name: entity_fact_junction; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.entity_fact_junction (
    user_id integer NOT NULL,
    entity_id integer NOT NULL,
    fact_id integer NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


--
-- Name: external_calendars; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.external_calendars (
    id integer NOT NULL,
    user_id integer NOT NULL,
    name text NOT NULL,
    url text NOT NULL,
    sync_enabled boolean DEFAULT true,
    sync_interval_hours integer DEFAULT 1,
    color text DEFAULT '#6366f1'::text,
    last_synced_at timestamp with time zone,
    last_error text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    username text,
    password text,
    CONSTRAINT name_not_empty CHECK ((length(TRIM(BOTH FROM name)) > 0)),
    CONSTRAINT sync_interval_hours_bounds CHECK (((sync_interval_hours >= 1) AND (sync_interval_hours <= 168)))
);


--
-- Name: TABLE external_calendars; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.external_calendars IS 'External calendar subscriptions for importing iCal events';


--
-- Name: COLUMN external_calendars.sync_interval_hours; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_calendars.sync_interval_hours IS 'Hours between automatic syncs (1 = hourly, for future background job)';


--
-- Name: COLUMN external_calendars.color; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_calendars.color IS 'Display color for events from this calendar (hex, default indigo)';


--
-- Name: COLUMN external_calendars.username; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_calendars.username IS 'Username for Basic Auth (if calendar requires authentication)';


--
-- Name: COLUMN external_calendars.password; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_calendars.password IS 'Encrypted password for Basic Auth (AES-256-GCM encrypted)';


--
-- Name: CONSTRAINT name_not_empty ON external_calendars; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON CONSTRAINT name_not_empty ON public.external_calendars IS 'Ensures calendar name is not empty or whitespace only';


--
-- Name: CONSTRAINT sync_interval_hours_bounds ON external_calendars; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON CONSTRAINT sync_interval_hours_bounds ON public.external_calendars IS 'Ensures sync interval is between 1 and 168 hours (1 hour to 7 days)';


--
-- Name: external_calendars_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.external_calendars_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: external_calendars_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.external_calendars_id_seq OWNED BY public.external_calendars.id;


--
-- Name: external_events; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.external_events (
    id integer NOT NULL,
    user_id integer NOT NULL,
    external_calendar_id integer,
    title text NOT NULL,
    description text,
    start_time timestamp with time zone NOT NULL,
    end_time timestamp with time zone NOT NULL,
    all_day boolean DEFAULT false,
    location text,
    external_uid text,
    external_url text,
    recurrence_rule text,
    color text,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    last_synced_at timestamp with time zone,
    card_pk integer,
    recurrence_id text,
    recurrence_instance integer
);


--
-- Name: TABLE external_events; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.external_events IS 'Imported calendar events from external iCal feeds';


--
-- Name: COLUMN external_events.external_uid; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_events.external_uid IS 'UID from iCal feed for deduplication';


--
-- Name: COLUMN external_events.color; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_events.color IS 'Override color for this specific event (falls back to calendar color if null)';


--
-- Name: COLUMN external_events.card_pk; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_events.card_pk IS 'Optional link to a card. Allows calendar events to be associated with specific cards for context and reference.';


--
-- Name: COLUMN external_events.recurrence_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_events.recurrence_id IS 'Identifier for the recurring event series (same as external_uid for non-recurring events)';


--
-- Name: COLUMN external_events.recurrence_instance; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.external_events.recurrence_instance IS 'Instance index for recurring events (0-based, NULL for non-recurring events)';


--
-- Name: external_events_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.external_events_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: external_events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.external_events_id_seq OWNED BY public.external_events.id;


--
-- Name: fact_card_junction; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.fact_card_junction (
    fact_id integer NOT NULL,
    card_pk integer NOT NULL,
    user_id integer NOT NULL,
    is_origin boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


--
-- Name: facts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.facts (
    id integer NOT NULL,
    user_id integer NOT NULL,
    card_pk integer NOT NULL,
    fact text NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: facts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.facts_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: facts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.facts_id_seq OWNED BY public.facts.id;


--
-- Name: file_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.file_tags (
    id integer NOT NULL,
    user_id integer NOT NULL,
    name character varying(100) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: file_tags_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.file_tags_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: file_tags_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.file_tags_id_seq OWNED BY public.file_tags.id;


--
-- Name: files; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.files (
    id integer NOT NULL,
    name text,
    type text,
    path text,
    filename text,
    size integer,
    created_by integer,
    updated_by integer,
    card_pk integer,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    is_deleted boolean DEFAULT false,
    user_id integer,
    thumbnail_path text,
    description text,
    extracted_text text
);


--
-- Name: files_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.files_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: files_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.files_id_seq OWNED BY public.files.id;


--
-- Name: files_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.files_tags (
    file_id integer NOT NULL,
    tag_id integer NOT NULL
);


--
-- Name: flashcard_reviews; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.flashcard_reviews (
    id integer,
    card_pk integer,
    user_id integer,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    rating integer
);


--
-- Name: habit_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.habit_logs (
    id integer NOT NULL,
    habit_id integer NOT NULL,
    user_id integer NOT NULL,
    completed_at timestamp with time zone NOT NULL,
    notes text,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: TABLE habit_logs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.habit_logs IS 'Individual habit completion records with timestamps and notes';


--
-- Name: COLUMN habit_logs.habit_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habit_logs.habit_id IS 'Reference to the habit being logged';


--
-- Name: COLUMN habit_logs.user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habit_logs.user_id IS 'User who completed the habit';


--
-- Name: COLUMN habit_logs.completed_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habit_logs.completed_at IS 'When the habit was completed';


--
-- Name: COLUMN habit_logs.notes; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habit_logs.notes IS 'Optional notes about this specific completion';


--
-- Name: COLUMN habit_logs.created_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habit_logs.created_at IS 'Record creation timestamp';


--
-- Name: habit_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.habit_logs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: habit_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.habit_logs_id_seq OWNED BY public.habit_logs.id;


--
-- Name: habits; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.habits (
    id integer NOT NULL,
    user_id integer NOT NULL,
    title character varying(255) NOT NULL,
    description text,
    frequency character varying(20) DEFAULT 'daily'::character varying NOT NULL,
    custom_days jsonb,
    icon character varying(50),
    color character varying(7),
    "position" integer DEFAULT 0,
    linked_task_id integer,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT habits_frequency_check CHECK (((frequency)::text = ANY ((ARRAY['daily'::character varying, 'weekly'::character varying, 'monthly'::character varying, 'custom_days'::character varying])::text[])))
);


--
-- Name: TABLE habits; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.habits IS 'User-defined habits for recurring behaviors and goals';


--
-- Name: COLUMN habits.title; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habits.title IS 'Name/title of the habit';


--
-- Name: COLUMN habits.description; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habits.description IS 'Optional description or notes about the habit';


--
-- Name: COLUMN habits.frequency; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habits.frequency IS 'How often the habit repeats: daily, weekly, monthly, or custom';


--
-- Name: COLUMN habits.custom_days; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habits.custom_days IS 'JSONB for custom schedule (e.g., specific days of week for weekly)';


--
-- Name: COLUMN habits.icon; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habits.icon IS 'Optional icon name for UI display';


--
-- Name: COLUMN habits.color; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habits.color IS 'Optional hex color code for UI display (e.g., #FF5733)';


--
-- Name: COLUMN habits."position"; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habits."position" IS 'Sort order for display in UI';


--
-- Name: COLUMN habits.linked_task_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.habits.linked_task_id IS 'Optional reference to a recurring task';


--
-- Name: habits_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.habits_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: habits_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.habits_id_seq OWNED BY public.habits.id;


--
-- Name: inactive_cards; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inactive_cards (
    id integer NOT NULL,
    card_pk integer NOT NULL,
    user_id integer NOT NULL,
    card_updated_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: inactive_cards_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.inactive_cards_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: inactive_cards_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.inactive_cards_id_seq OWNED BY public.inactive_cards.id;


--
-- Name: keywords; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.keywords (
    id integer NOT NULL,
    card_pk integer,
    user_id integer,
    keyword text
);


--
-- Name: keywords_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.keywords_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: keywords_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.keywords_id_seq OWNED BY public.keywords.id;


--
-- Name: llm_jobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.llm_jobs (
    id integer NOT NULL,
    user_id integer NOT NULL,
    job_type character varying(50) NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    priority integer DEFAULT 5 NOT NULL,
    payload jsonb NOT NULL,
    result jsonb,
    error_message text,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    started_at timestamp without time zone,
    completed_at timestamp without time zone,
    retry_count integer DEFAULT 0 NOT NULL,
    max_retries integer DEFAULT 3 NOT NULL,
    timeout_seconds integer DEFAULT 300 NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    last_heartbeat timestamp without time zone,
    correlation_id character varying(36),
    CONSTRAINT llm_jobs_status_check CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'running'::character varying, 'completed'::character varying, 'failed'::character varying, 'cancelled'::character varying])::text[]))),
    CONSTRAINT llm_jobs_type_check CHECK (((job_type)::text = ANY ((ARRAY['summarization'::character varying, 'entity_extraction'::character varying, 'fact_entity_extraction'::character varying, 'chat'::character varying, 'memory'::character varying, 'email'::character varying])::text[])))
);


--
-- Name: TABLE llm_jobs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.llm_jobs IS 'Queue for asynchronous LLM operations with status tracking and retry logic';


--
-- Name: COLUMN llm_jobs.job_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.llm_jobs.job_type IS 'Type of LLM job: embedding, summarization, entity_extraction, chat, memory, email';


--
-- Name: COLUMN llm_jobs.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.llm_jobs.status IS 'Job status: pending, running, completed, failed, cancelled';


--
-- Name: COLUMN llm_jobs.priority; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.llm_jobs.priority IS 'Job priority (lower number = higher priority, default 5)';


--
-- Name: COLUMN llm_jobs.payload; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.llm_jobs.payload IS 'Job-specific input data as JSONB';


--
-- Name: COLUMN llm_jobs.result; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.llm_jobs.result IS 'Job result data as JSONB';


--
-- Name: COLUMN llm_jobs.retry_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.llm_jobs.retry_count IS 'Number of retry attempts made';


--
-- Name: COLUMN llm_jobs.max_retries; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.llm_jobs.max_retries IS 'Maximum retry attempts before marking as failed';


--
-- Name: COLUMN llm_jobs.timeout_seconds; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.llm_jobs.timeout_seconds IS 'Maximum time allowed for job execution';


--
-- Name: COLUMN llm_jobs.updated_at; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.llm_jobs.updated_at IS 'Last time the job was updated (status change, retry, etc.)';


--
-- Name: COLUMN llm_jobs.last_heartbeat; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.llm_jobs.last_heartbeat IS 'Last heartbeat timestamp from running job, used to detect stuck/abandoned jobs';


--
-- Name: COLUMN llm_jobs.correlation_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.llm_jobs.correlation_id IS 'Correlation ID for distributed tracing - links job to request context';


--
-- Name: llm_jobs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.llm_jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: llm_jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.llm_jobs_id_seq OWNED BY public.llm_jobs.id;


--
-- Name: llm_models; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.llm_models (
    id integer NOT NULL,
    provider_id integer,
    name character varying(255) NOT NULL,
    model_identifier character varying(255) NOT NULL,
    description text,
    is_active boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: llm_models_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.llm_models_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: llm_models_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.llm_models_id_seq OWNED BY public.llm_models.id;


--
-- Name: llm_providers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.llm_providers (
    id integer NOT NULL,
    name character varying(255) NOT NULL,
    base_url character varying(255),
    api_key_required boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    user_id integer,
    api_key text
);


--
-- Name: llm_providers_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.llm_providers_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: llm_providers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.llm_providers_id_seq OWNED BY public.llm_providers.id;


--
-- Name: llm_query_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.llm_query_log (
    id integer NOT NULL,
    user_id integer,
    model text,
    prompt_tokens integer,
    completion_tokens integer,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    cost_usd numeric(10,4),
    request_type text
);


--
-- Name: llm_query_log_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.llm_query_log_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: llm_query_log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.llm_query_log_id_seq OWNED BY public.llm_query_log.id;


--
-- Name: mailing_list; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mailing_list (
    id integer NOT NULL,
    email text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    welcome_email_sent boolean DEFAULT false,
    subscribed boolean DEFAULT true
);


--
-- Name: mailing_list_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.mailing_list_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: mailing_list_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.mailing_list_id_seq OWNED BY public.mailing_list.id;


--
-- Name: mailing_list_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mailing_list_messages (
    id integer NOT NULL,
    subject text NOT NULL,
    body text NOT NULL,
    sent_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    total_recipients integer
);


--
-- Name: mailing_list_messages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.mailing_list_messages_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: mailing_list_messages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.mailing_list_messages_id_seq OWNED BY public.mailing_list_messages.id;


--
-- Name: mailing_list_recipients; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.mailing_list_recipients (
    id integer NOT NULL,
    message_id integer NOT NULL,
    recipient_email text NOT NULL,
    recipient_type text NOT NULL,
    sent_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: mailing_list_recipients_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.mailing_list_recipients_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: mailing_list_recipients_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.mailing_list_recipients_id_seq OWNED BY public.mailing_list_recipients.id;


--
-- Name: migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.migrations (
    id integer NOT NULL,
    migration_name character varying(255) NOT NULL,
    applied_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: migrations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.migrations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: migrations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.migrations_id_seq OWNED BY public.migrations.id;


--
-- Name: notification_preferences; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notification_preferences (
    user_id integer NOT NULL,
    show_unprocessed_emails boolean DEFAULT true,
    show_starred_articles boolean DEFAULT true,
    show_priority_tasks boolean DEFAULT true,
    show_priority_feeds boolean DEFAULT true,
    items_per_page integer DEFAULT 50,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT notification_preferences_items_per_page_check CHECK (((items_per_page >= 10) AND (items_per_page <= 200)))
);


--
-- Name: TABLE notification_preferences; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.notification_preferences IS 'User preferences for notification filtering and display';


--
-- Name: COLUMN notification_preferences.show_unprocessed_emails; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.notification_preferences.show_unprocessed_emails IS 'Include unprocessed emails in unified inbox';


--
-- Name: COLUMN notification_preferences.show_starred_articles; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.notification_preferences.show_starred_articles IS 'Include starred RSS articles in unified inbox';


--
-- Name: COLUMN notification_preferences.show_priority_tasks; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.notification_preferences.show_priority_tasks IS 'Include priority tasks in unified inbox';


--
-- Name: COLUMN notification_preferences.show_priority_feeds; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.notification_preferences.show_priority_feeds IS 'Include priority RSS feeds in unified inbox';


--
-- Name: COLUMN notification_preferences.items_per_page; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.notification_preferences.items_per_page IS 'Number of items to display per page (10-200)';


--
-- Name: notifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notifications (
    id integer NOT NULL,
    user_id integer NOT NULL,
    source_type character varying(20) NOT NULL,
    source_id integer NOT NULL,
    title text NOT NULL,
    preview text,
    "timestamp" timestamp with time zone NOT NULL,
    importance_score integer DEFAULT 0,
    is_read boolean DEFAULT false,
    is_archived boolean DEFAULT false,
    filter_tags text[],
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT notifications_importance_score_check CHECK ((importance_score >= 0)),
    CONSTRAINT notifications_source_type_check CHECK (((source_type)::text = ANY ((ARRAY['email'::character varying, 'rss'::character varying, 'task'::character varying])::text[])))
);


--
-- Name: TABLE notifications; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.notifications IS 'Unified view of important items from email, RSS, and tasks';


--
-- Name: COLUMN notifications.source_type; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.notifications.source_type IS 'Type of source: email, rss, or task';


--
-- Name: COLUMN notifications.importance_score; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.notifications.importance_score IS 'Computed score for sorting';


--
-- Name: COLUMN notifications.filter_tags; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.notifications.filter_tags IS 'Tags for filtering';


--
-- Name: notifications_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.notifications_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: notifications_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.notifications_id_seq OWNED BY public.notifications.id;


--
-- Name: starred_cards; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.starred_cards (
    id integer NOT NULL,
    card_pk integer,
    user_id integer NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: pinned_cards_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.pinned_cards_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: pinned_cards_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.pinned_cards_id_seq OWNED BY public.starred_cards.id;


--
-- Name: starred_searches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.starred_searches (
    id integer NOT NULL,
    user_id integer NOT NULL,
    title text NOT NULL,
    search_term text NOT NULL,
    search_config jsonb NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE starred_searches; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.starred_searches IS 'Stores user-pinned searches with their configuration';


--
-- Name: pinned_searches_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.pinned_searches_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: pinned_searches_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.pinned_searches_id_seq OWNED BY public.starred_searches.id;


--
-- Name: revenue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.revenue (
    id integer NOT NULL,
    user_id integer NOT NULL,
    stripe_subscription_id text NOT NULL,
    stripe_invoice_id text NOT NULL,
    amount_cents integer NOT NULL,
    currency text DEFAULT 'usd'::text NOT NULL,
    description text,
    payment_date timestamp with time zone DEFAULT now(),
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: revenue_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.revenue_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: revenue_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.revenue_id_seq OWNED BY public.revenue.id;


--
-- Name: rss_articles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rss_articles (
    id integer NOT NULL,
    user_id integer NOT NULL,
    feed_id integer NOT NULL,
    title text NOT NULL,
    content text,
    author text,
    url text NOT NULL,
    published_at timestamp with time zone,
    fetched_at timestamp with time zone DEFAULT now(),
    read boolean DEFAULT false,
    card_id integer,
    is_starred boolean DEFAULT false
);


--
-- Name: TABLE rss_articles; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.rss_articles IS 'Articles fetched from RSS feeds';


--
-- Name: COLUMN rss_articles.card_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.rss_articles.card_id IS 'ID of the card created from this RSS article, if converted';


--
-- Name: COLUMN rss_articles.is_starred; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.rss_articles.is_starred IS 'Whether the article is starred by the user';


--
-- Name: rss_articles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.rss_articles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: rss_articles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.rss_articles_id_seq OWNED BY public.rss_articles.id;


--
-- Name: rss_feeds; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rss_feeds (
    id integer NOT NULL,
    user_id integer NOT NULL,
    url text NOT NULL,
    name text NOT NULL,
    folder text,
    auto_tags text DEFAULT ''::text,
    fetch_interval integer DEFAULT 60,
    last_fetched_at timestamp with time zone,
    last_error text,
    enabled boolean DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    priority boolean DEFAULT false
);


--
-- Name: TABLE rss_feeds; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.rss_feeds IS 'RSS feed subscriptions per user';


--
-- Name: COLUMN rss_feeds.auto_tags; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.rss_feeds.auto_tags IS 'Comma-separated tags to apply when converting articles to cards';


--
-- Name: COLUMN rss_feeds.fetch_interval; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.rss_feeds.fetch_interval IS 'Fetch interval in minutes';


--
-- Name: COLUMN rss_feeds.priority; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.rss_feeds.priority IS 'Manual priority flag for smart feed - feeds marked as priority always rank higher';


--
-- Name: rss_feeds_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.rss_feeds_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: rss_feeds_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.rss_feeds_id_seq OWNED BY public.rss_feeds.id;


--
-- Name: rss_folders; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rss_folders (
    id integer NOT NULL,
    user_id integer NOT NULL,
    name text NOT NULL,
    order_index integer DEFAULT 0
);


--
-- Name: TABLE rss_folders; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.rss_folders IS 'User-defined folders for organizing feeds';


--
-- Name: rss_folders_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.rss_folders_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: rss_folders_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.rss_folders_id_seq OWNED BY public.rss_folders.id;


--
-- Name: rss_seen_articles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rss_seen_articles (
    id integer NOT NULL,
    feed_id integer NOT NULL,
    url text NOT NULL,
    first_seen_at timestamp with time zone DEFAULT now()
);


--
-- Name: TABLE rss_seen_articles; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.rss_seen_articles IS 'Tracks all article URLs ever seen per feed to prevent re-syncing after cleanup';


--
-- Name: rss_seen_articles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.rss_seen_articles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: rss_seen_articles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.rss_seen_articles_id_seq OWNED BY public.rss_seen_articles.id;


--
-- Name: scheduled_job_runs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.scheduled_job_runs (
    id integer NOT NULL,
    job_name character varying(255) NOT NULL,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    status character varying(20) NOT NULL,
    error_message text,
    retry_count integer DEFAULT 0,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT scheduled_job_runs_status_check CHECK (((status)::text = ANY ((ARRAY['running'::character varying, 'completed'::character varying, 'failed'::character varying, 'cancelled'::character varying])::text[])))
);


--
-- Name: TABLE scheduled_job_runs; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.scheduled_job_runs IS 'Execution history for scheduled jobs - tracks start/end times, status, and errors';


--
-- Name: COLUMN scheduled_job_runs.job_name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.scheduled_job_runs.job_name IS 'Identifier for the type of job (e.g., "daily-backup", "hourly-sync")';


--
-- Name: COLUMN scheduled_job_runs.status; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.scheduled_job_runs.status IS 'Current state: running, completed, failed, or cancelled';


--
-- Name: COLUMN scheduled_job_runs.retry_count; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.scheduled_job_runs.retry_count IS 'Number of retry attempts for this job execution';


--
-- Name: scheduled_job_runs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.scheduled_job_runs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: scheduled_job_runs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.scheduled_job_runs_id_seq OWNED BY public.scheduled_job_runs.id;


--
-- Name: schema_definitions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_definitions (
    id integer NOT NULL,
    name text NOT NULL,
    owner_id integer NOT NULL,
    fields jsonb NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    is_deleted boolean DEFAULT false,
    slug text,
    CONSTRAINT schema_definitions_name_check CHECK ((char_length(name) <= 255))
);


--
-- Name: schema_definitions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.schema_definitions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: schema_definitions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.schema_definitions_id_seq OWNED BY public.schema_definitions.id;


--
-- Name: spreadsheets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.spreadsheets (
    id integer NOT NULL,
    user_id integer NOT NULL,
    card_id integer NOT NULL,
    name character varying(255) NOT NULL,
    rows integer DEFAULT 5 NOT NULL,
    cols integer DEFAULT 5 NOT NULL,
    data jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE spreadsheets; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.spreadsheets IS 'Spreadsheet data attached to cards, migrated from card body JSON blocks';


--
-- Name: COLUMN spreadsheets.user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.spreadsheets.user_id IS 'Owner of the spreadsheet';


--
-- Name: COLUMN spreadsheets.card_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.spreadsheets.card_id IS 'Card that contains this spreadsheet';


--
-- Name: COLUMN spreadsheets.name; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.spreadsheets.name IS 'Display name of the spreadsheet';


--
-- Name: COLUMN spreadsheets.rows; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.spreadsheets.rows IS 'Number of rows in the spreadsheet';


--
-- Name: COLUMN spreadsheets.cols; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.spreadsheets.cols IS 'Number of columns in the spreadsheet';


--
-- Name: COLUMN spreadsheets.data; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.spreadsheets.data IS 'Cell data stored as JSONB';


--
-- Name: spreadsheets_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.spreadsheets_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: spreadsheets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.spreadsheets_id_seq OWNED BY public.spreadsheets.id;


--
-- Name: stripe_plans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stripe_plans (
    id integer NOT NULL,
    stripe_product_id text NOT NULL,
    stripe_price_id text NOT NULL,
    name text,
    description text,
    active boolean NOT NULL,
    unit_amount integer,
    currency text,
    "interval" text,
    interval_count integer,
    trial_days integer,
    metadata text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: stripe_plans_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.stripe_plans_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: stripe_plans_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.stripe_plans_id_seq OWNED BY public.stripe_plans.id;


--
-- Name: summarizations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.summarizations (
    id integer NOT NULL,
    user_id integer NOT NULL,
    input_text text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    result text,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    card_pk integer,
    prompt_tokens integer DEFAULT 0,
    completion_tokens integer DEFAULT 0,
    total_tokens integer DEFAULT 0,
    cost double precision DEFAULT 0,
    model text DEFAULT ''::text,
    llm_job_id integer,
    CONSTRAINT summarizations_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'processing'::text, 'complete'::text, 'failed'::text])))
);


--
-- Name: COLUMN summarizations.card_pk; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.summarizations.card_pk IS 'Reference to the card being analyzed. NULL indicates a manual/standalone summarization created without a card, which is excluded from card-specific queries.';


--
-- Name: COLUMN summarizations.llm_job_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.summarizations.llm_job_id IS 'Reference to the LLM job processing this summarization';


--
-- Name: summarizations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.summarizations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: summarizations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.summarizations_id_seq OWNED BY public.summarizations.id;


--
-- Name: summary_arguments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.summary_arguments (
    id integer NOT NULL,
    user_id integer NOT NULL,
    card_pk integer,
    summarization_id integer NOT NULL,
    thesis_id integer NOT NULL,
    argument text NOT NULL,
    importance integer NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: summary_arguments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.summary_arguments_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: summary_arguments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.summary_arguments_id_seq OWNED BY public.summary_arguments.id;


--
-- Name: summary_sections; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.summary_sections (
    id integer NOT NULL,
    user_id integer NOT NULL,
    card_pk integer,
    summarization_id integer NOT NULL,
    section_title text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    section_order integer DEFAULT 0
);


--
-- Name: summary_sections_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.summary_sections_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: summary_sections_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.summary_sections_id_seq OWNED BY public.summary_sections.id;


--
-- Name: summary_theses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.summary_theses (
    id integer NOT NULL,
    user_id integer NOT NULL,
    card_pk integer,
    summarization_id integer NOT NULL,
    section_id integer NOT NULL,
    thesis text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: summary_theses_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.summary_theses_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: summary_theses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.summary_theses_id_seq OWNED BY public.summary_theses.id;


--
-- Name: tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tags (
    id integer NOT NULL,
    name text NOT NULL,
    color text,
    user_id integer,
    is_deleted boolean DEFAULT false,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: tags_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.tags_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: tags_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.tags_id_seq OWNED BY public.tags.id;


--
-- Name: task_dependencies; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_dependencies (
    id integer NOT NULL,
    task_id integer NOT NULL,
    blocking_task_id integer NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    CONSTRAINT task_dependencies_check CHECK ((task_id <> blocking_task_id))
);


--
-- Name: task_dependencies_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_dependencies_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_dependencies_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_dependencies_id_seq OWNED BY public.task_dependencies.id;


--
-- Name: task_statuses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_statuses (
    id integer NOT NULL,
    user_id integer NOT NULL,
    name character varying(50) NOT NULL,
    display_name character varying(100) NOT NULL,
    color character varying(7) NOT NULL,
    icon character varying(10),
    "position" integer NOT NULL,
    is_default boolean DEFAULT false,
    is_complete_state boolean DEFAULT false,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


--
-- Name: task_statuses_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.task_statuses_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: task_statuses_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.task_statuses_id_seq OWNED BY public.task_statuses.id;


--
-- Name: task_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.task_tags (
    task_pk integer NOT NULL,
    tag_id integer NOT NULL
);


--
-- Name: tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tasks (
    id integer NOT NULL,
    card_pk integer,
    user_id integer NOT NULL,
    scheduled_date timestamp with time zone,
    due_date timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    title text NOT NULL,
    is_complete boolean DEFAULT false NOT NULL,
    is_deleted boolean DEFAULT false NOT NULL,
    priority text,
    status character varying(50) DEFAULT 'todo'::character varying,
    reminder_time timestamp with time zone,
    reminder_sent boolean DEFAULT false,
    description text,
    external_uid text,
    external_calendar_id integer,
    parent_task_id integer,
    sort_order integer
);


--
-- Name: COLUMN tasks.external_uid; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tasks.external_uid IS 'UID from VTODO component in iCal feed, used for upsert identification';


--
-- Name: COLUMN tasks.external_calendar_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tasks.external_calendar_id IS 'Reference to external calendar subscription this task was imported from';


--
-- Name: COLUMN tasks.parent_task_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.tasks.parent_task_id IS 'References parent task for subtask hierarchy. NULL for root tasks. Single level nesting only.';


--
-- Name: tasks_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.tasks_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: tasks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.tasks_id_seq OWNED BY public.tasks.id;


--
-- Name: unsorted_cards; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.unsorted_cards (
    id integer NOT NULL,
    title text,
    body text,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: unsorted_cards_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.unsorted_cards_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: unsorted_cards_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.unsorted_cards_id_seq OWNED BY public.unsorted_cards.id;


--
-- Name: user_llm_configurations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_llm_configurations (
    id integer NOT NULL,
    user_id integer,
    model_id integer,
    api_key character varying(255),
    custom_settings jsonb DEFAULT '{}'::jsonb,
    is_default boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: user_llm_configurations_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_llm_configurations_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_llm_configurations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_llm_configurations_id_seq OWNED BY public.user_llm_configurations.id;


--
-- Name: user_memories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_memories (
    id integer NOT NULL,
    user_id integer NOT NULL,
    memory text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: user_memories_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_memories_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_memories_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_memories_id_seq OWNED BY public.user_memories.id;


--
-- Name: user_stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_stats (
    user_id integer NOT NULL,
    card_count integer DEFAULT 0 NOT NULL,
    task_count integer DEFAULT 0 NOT NULL,
    file_count integer DEFAULT 0 NOT NULL,
    chat_message_count integer DEFAULT 0 NOT NULL,
    llm_cost_usd numeric(10,4) DEFAULT 0.0000 NOT NULL,
    revenue_cents integer DEFAULT 0 NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE user_stats; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.user_stats IS 'Cached aggregate statistics for users - updated via triggers';


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id integer NOT NULL,
    username text,
    email text,
    password text,
    created_at timestamp without time zone,
    updated_at timestamp without time zone,
    is_admin boolean DEFAULT false NOT NULL,
    email_validated boolean DEFAULT false NOT NULL,
    stripe_customer_id text,
    stripe_subscription_id text,
    stripe_subscription_status text,
    stripe_subscription_frequency text,
    stripe_current_plan text,
    last_login timestamp without time zone,
    can_upload_files boolean DEFAULT true,
    max_file_storage integer DEFAULT 100000000,
    dashboard_card_pk integer DEFAULT 0,
    last_seen timestamp without time zone,
    memory_has_changed boolean DEFAULT true,
    auth_provider text DEFAULT 'local'::text,
    github_id text,
    has_seen_getting_started boolean DEFAULT false,
    stripe_cancel_at_period_end boolean DEFAULT false,
    timezone text DEFAULT 'UTC'::text,
    last_memory_job_id integer,
    caldav_url text,
    caldav_token text,
    is_agent boolean DEFAULT false NOT NULL,
    owner_user_id integer,
    api_key_hash character(60),
    CONSTRAINT check_agent_has_api_key CHECK (((NOT is_agent) OR (api_key_hash IS NOT NULL))),
    CONSTRAINT check_agent_not_admin CHECK ((NOT ((is_agent = true) AND (is_admin = true))))
);


--
-- Name: COLUMN users.caldav_url; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.users.caldav_url IS 'CalDAV server URL for calendar sync (e.g., https://calendar.google.com/dav/user@example.com/user)';


--
-- Name: COLUMN users.caldav_token; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.users.caldav_token IS 'Secure token for accessing iCal feed at /api/user/calendar.ics?token=XYZ';


--
-- Name: COLUMN users.is_agent; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.users.is_agent IS 'Whether this user is an AI agent';


--
-- Name: COLUMN users.owner_user_id; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.users.owner_user_id IS 'For agents: the user who owns this agent';


--
-- Name: COLUMN users.api_key_hash; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.users.api_key_hash IS 'For agents: hashed API key for authentication';


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: admin_audit_log id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_audit_log ALTER COLUMN id SET DEFAULT nextval('public.admin_audit_log_id_seq'::regclass);


--
-- Name: agent_activity_log id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_activity_log ALTER COLUMN id SET DEFAULT nextval('public.agent_activity_log_id_seq'::regclass);


--
-- Name: api_keys id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys ALTER COLUMN id SET DEFAULT nextval('public.api_keys_id_seq'::regclass);


--
-- Name: audit_events id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_events ALTER COLUMN id SET DEFAULT nextval('public.audit_events_id_seq'::regclass);


--
-- Name: card_chunks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_chunks ALTER COLUMN id SET DEFAULT nextval('public.card_chunks_id_seq'::regclass);


--
-- Name: card_embeddings id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_embeddings ALTER COLUMN id SET DEFAULT nextval('public.card_embeddings_id_seq'::regclass);


--
-- Name: card_templates id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_templates ALTER COLUMN id SET DEFAULT nextval('public.card_templates_id_seq'::regclass);


--
-- Name: card_views id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_views ALTER COLUMN id SET DEFAULT nextval('public.card_views_id_seq'::regclass);


--
-- Name: cards id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cards ALTER COLUMN id SET DEFAULT nextval('public.cards_id_seq'::regclass);


--
-- Name: categories id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categories ALTER COLUMN id SET DEFAULT nextval('public.categories_id_seq'::regclass);


--
-- Name: chat_instructions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_instructions ALTER COLUMN id SET DEFAULT nextval('public.chat_instructions_id_seq'::regclass);


--
-- Name: chat_usage_quotas id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_usage_quotas ALTER COLUMN id SET DEFAULT nextval('public.chat_usage_quotas_id_seq'::regclass);


--
-- Name: email_accounts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_accounts ALTER COLUMN id SET DEFAULT nextval('public.email_accounts_id_seq'::regclass);


--
-- Name: email_attachments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_attachments ALTER COLUMN id SET DEFAULT nextval('public.email_attachments_id_seq'::regclass);


--
-- Name: email_card_links id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_card_links ALTER COLUMN id SET DEFAULT nextval('public.email_card_links_id_seq'::regclass);


--
-- Name: email_fact_junction id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_fact_junction ALTER COLUMN id SET DEFAULT nextval('public.email_fact_junction_id_seq'::regclass);


--
-- Name: email_triage_decisions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_triage_decisions ALTER COLUMN id SET DEFAULT nextval('public.email_triage_decisions_id_seq'::regclass);


--
-- Name: emails id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.emails ALTER COLUMN id SET DEFAULT nextval('public.emails_id_seq'::regclass);


--
-- Name: entities id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entities ALTER COLUMN id SET DEFAULT nextval('public.entities_id_seq'::regclass);


--
-- Name: entity_card_junction id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_card_junction ALTER COLUMN id SET DEFAULT nextval('public.entity_card_junction_id_seq'::regclass);


--
-- Name: external_calendars id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_calendars ALTER COLUMN id SET DEFAULT nextval('public.external_calendars_id_seq'::regclass);


--
-- Name: external_events id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_events ALTER COLUMN id SET DEFAULT nextval('public.external_events_id_seq'::regclass);


--
-- Name: facts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.facts ALTER COLUMN id SET DEFAULT nextval('public.facts_id_seq'::regclass);


--
-- Name: file_tags id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_tags ALTER COLUMN id SET DEFAULT nextval('public.file_tags_id_seq'::regclass);


--
-- Name: files id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.files ALTER COLUMN id SET DEFAULT nextval('public.files_id_seq'::regclass);


--
-- Name: habit_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.habit_logs ALTER COLUMN id SET DEFAULT nextval('public.habit_logs_id_seq'::regclass);


--
-- Name: habits id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.habits ALTER COLUMN id SET DEFAULT nextval('public.habits_id_seq'::regclass);


--
-- Name: inactive_cards id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inactive_cards ALTER COLUMN id SET DEFAULT nextval('public.inactive_cards_id_seq'::regclass);


--
-- Name: keywords id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.keywords ALTER COLUMN id SET DEFAULT nextval('public.keywords_id_seq'::regclass);


--
-- Name: llm_jobs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_jobs ALTER COLUMN id SET DEFAULT nextval('public.llm_jobs_id_seq'::regclass);


--
-- Name: llm_models id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_models ALTER COLUMN id SET DEFAULT nextval('public.llm_models_id_seq'::regclass);


--
-- Name: llm_providers id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_providers ALTER COLUMN id SET DEFAULT nextval('public.llm_providers_id_seq'::regclass);


--
-- Name: llm_query_log id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_query_log ALTER COLUMN id SET DEFAULT nextval('public.llm_query_log_id_seq'::regclass);


--
-- Name: mailing_list id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mailing_list ALTER COLUMN id SET DEFAULT nextval('public.mailing_list_id_seq'::regclass);


--
-- Name: mailing_list_messages id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mailing_list_messages ALTER COLUMN id SET DEFAULT nextval('public.mailing_list_messages_id_seq'::regclass);


--
-- Name: mailing_list_recipients id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mailing_list_recipients ALTER COLUMN id SET DEFAULT nextval('public.mailing_list_recipients_id_seq'::regclass);


--
-- Name: migrations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.migrations ALTER COLUMN id SET DEFAULT nextval('public.migrations_id_seq'::regclass);


--
-- Name: notifications id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications ALTER COLUMN id SET DEFAULT nextval('public.notifications_id_seq'::regclass);


--
-- Name: revenue id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revenue ALTER COLUMN id SET DEFAULT nextval('public.revenue_id_seq'::regclass);


--
-- Name: rss_articles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_articles ALTER COLUMN id SET DEFAULT nextval('public.rss_articles_id_seq'::regclass);


--
-- Name: rss_feeds id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_feeds ALTER COLUMN id SET DEFAULT nextval('public.rss_feeds_id_seq'::regclass);


--
-- Name: rss_folders id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_folders ALTER COLUMN id SET DEFAULT nextval('public.rss_folders_id_seq'::regclass);


--
-- Name: rss_seen_articles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_seen_articles ALTER COLUMN id SET DEFAULT nextval('public.rss_seen_articles_id_seq'::regclass);


--
-- Name: scheduled_job_runs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.scheduled_job_runs ALTER COLUMN id SET DEFAULT nextval('public.scheduled_job_runs_id_seq'::regclass);


--
-- Name: schema_definitions id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_definitions ALTER COLUMN id SET DEFAULT nextval('public.schema_definitions_id_seq'::regclass);


--
-- Name: spreadsheets id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.spreadsheets ALTER COLUMN id SET DEFAULT nextval('public.spreadsheets_id_seq'::regclass);


--
-- Name: starred_cards id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.starred_cards ALTER COLUMN id SET DEFAULT nextval('public.pinned_cards_id_seq'::regclass);


--
-- Name: starred_searches id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.starred_searches ALTER COLUMN id SET DEFAULT nextval('public.pinned_searches_id_seq'::regclass);


--
-- Name: stripe_plans id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stripe_plans ALTER COLUMN id SET DEFAULT nextval('public.stripe_plans_id_seq'::regclass);


--
-- Name: summarizations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summarizations ALTER COLUMN id SET DEFAULT nextval('public.summarizations_id_seq'::regclass);


--
-- Name: summary_arguments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_arguments ALTER COLUMN id SET DEFAULT nextval('public.summary_arguments_id_seq'::regclass);


--
-- Name: summary_sections id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_sections ALTER COLUMN id SET DEFAULT nextval('public.summary_sections_id_seq'::regclass);


--
-- Name: summary_theses id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_theses ALTER COLUMN id SET DEFAULT nextval('public.summary_theses_id_seq'::regclass);


--
-- Name: tags id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tags ALTER COLUMN id SET DEFAULT nextval('public.tags_id_seq'::regclass);


--
-- Name: task_dependencies id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_dependencies ALTER COLUMN id SET DEFAULT nextval('public.task_dependencies_id_seq'::regclass);


--
-- Name: task_statuses id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_statuses ALTER COLUMN id SET DEFAULT nextval('public.task_statuses_id_seq'::regclass);


--
-- Name: tasks id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks ALTER COLUMN id SET DEFAULT nextval('public.tasks_id_seq'::regclass);


--
-- Name: unsorted_cards id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unsorted_cards ALTER COLUMN id SET DEFAULT nextval('public.unsorted_cards_id_seq'::regclass);


--
-- Name: user_llm_configurations id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_llm_configurations ALTER COLUMN id SET DEFAULT nextval('public.user_llm_configurations_id_seq'::regclass);


--
-- Name: user_memories id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_memories ALTER COLUMN id SET DEFAULT nextval('public.user_memories_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: admin_audit_log admin_audit_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_audit_log
    ADD CONSTRAINT admin_audit_log_pkey PRIMARY KEY (id);


--
-- Name: agent_activity_log agent_activity_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_activity_log
    ADD CONSTRAINT agent_activity_log_pkey PRIMARY KEY (id);


--
-- Name: api_keys api_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_pkey PRIMARY KEY (id);


--
-- Name: audit_events audit_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_events
    ADD CONSTRAINT audit_events_pkey PRIMARY KEY (id);


--
-- Name: card_chunks card_chunks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_chunks
    ADD CONSTRAINT card_chunks_pkey PRIMARY KEY (id);


--
-- Name: card_embeddings card_embeddings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_embeddings
    ADD CONSTRAINT card_embeddings_pkey PRIMARY KEY (id);


--
-- Name: card_tags card_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_tags
    ADD CONSTRAINT card_tags_pkey PRIMARY KEY (card_pk, tag_id);


--
-- Name: card_templates card_templates_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_templates
    ADD CONSTRAINT card_templates_pkey PRIMARY KEY (id);


--
-- Name: card_views card_views_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_views
    ADD CONSTRAINT card_views_pkey PRIMARY KEY (id);


--
-- Name: cards cards_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cards
    ADD CONSTRAINT cards_pkey PRIMARY KEY (id);


--
-- Name: categories categories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categories
    ADD CONSTRAINT categories_pkey PRIMARY KEY (id);


--
-- Name: chat_conversations chat_conversations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_conversations
    ADD CONSTRAINT chat_conversations_pkey PRIMARY KEY (id);


--
-- Name: chat_instructions chat_instructions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_instructions
    ADD CONSTRAINT chat_instructions_pkey PRIMARY KEY (id);


--
-- Name: chat_instructions chat_instructions_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_instructions
    ADD CONSTRAINT chat_instructions_user_id_key UNIQUE (user_id);


--
-- Name: chat_messages chat_messages_conversation_id_sequence_number_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_messages
    ADD CONSTRAINT chat_messages_conversation_id_sequence_number_key UNIQUE (conversation_id, sequence_number);


--
-- Name: chat_messages chat_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_messages
    ADD CONSTRAINT chat_messages_pkey PRIMARY KEY (id);


--
-- Name: chat_tool_calls chat_tool_calls_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_tool_calls
    ADD CONSTRAINT chat_tool_calls_pkey PRIMARY KEY (id);


--
-- Name: chat_usage_quotas chat_usage_quotas_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_usage_quotas
    ADD CONSTRAINT chat_usage_quotas_pkey PRIMARY KEY (id);


--
-- Name: chat_usage_quotas chat_usage_quotas_user_id_quota_type_reset_date_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_usage_quotas
    ADD CONSTRAINT chat_usage_quotas_user_id_quota_type_reset_date_key UNIQUE (user_id, quota_type, reset_date);


--
-- Name: email_accounts email_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_accounts
    ADD CONSTRAINT email_accounts_pkey PRIMARY KEY (id);


--
-- Name: email_accounts email_accounts_user_id_email_address_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_accounts
    ADD CONSTRAINT email_accounts_user_id_email_address_key UNIQUE (user_id, email_address);


--
-- Name: email_attachments email_attachments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_attachments
    ADD CONSTRAINT email_attachments_pkey PRIMARY KEY (id);


--
-- Name: email_card_links email_card_links_email_id_card_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_card_links
    ADD CONSTRAINT email_card_links_email_id_card_id_key UNIQUE (email_id, card_id);


--
-- Name: email_card_links email_card_links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_card_links
    ADD CONSTRAINT email_card_links_pkey PRIMARY KEY (id);


--
-- Name: email_fact_junction email_fact_junction_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_fact_junction
    ADD CONSTRAINT email_fact_junction_pkey PRIMARY KEY (id);


--
-- Name: email_fact_junction email_fact_junction_user_id_email_id_fact_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_fact_junction
    ADD CONSTRAINT email_fact_junction_user_id_email_id_fact_id_key UNIQUE (user_id, email_id, fact_id);


--
-- Name: email_triage_decisions email_triage_decisions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_triage_decisions
    ADD CONSTRAINT email_triage_decisions_pkey PRIMARY KEY (id);


--
-- Name: emails emails_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.emails
    ADD CONSTRAINT emails_pkey PRIMARY KEY (id);


--
-- Name: emails emails_user_id_message_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.emails
    ADD CONSTRAINT emails_user_id_message_id_key UNIQUE (user_id, message_id);


--
-- Name: entities entities_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entities
    ADD CONSTRAINT entities_pkey PRIMARY KEY (id);


--
-- Name: entities entities_user_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entities
    ADD CONSTRAINT entities_user_id_name_key UNIQUE (user_id, name);


--
-- Name: entity_card_junction entity_card_junction_entity_id_card_pk_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_card_junction
    ADD CONSTRAINT entity_card_junction_entity_id_card_pk_key UNIQUE (entity_id, card_pk);


--
-- Name: entity_card_junction entity_card_junction_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_card_junction
    ADD CONSTRAINT entity_card_junction_pkey PRIMARY KEY (id);


--
-- Name: entity_fact_junction entity_fact_junction_entity_id_fact_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_fact_junction
    ADD CONSTRAINT entity_fact_junction_entity_id_fact_id_key UNIQUE (entity_id, fact_id);


--
-- Name: external_calendars external_calendars_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_calendars
    ADD CONSTRAINT external_calendars_pkey PRIMARY KEY (id);


--
-- Name: external_calendars external_calendars_user_id_url_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_calendars
    ADD CONSTRAINT external_calendars_user_id_url_key UNIQUE (user_id, url);


--
-- Name: external_events external_events_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_events
    ADD CONSTRAINT external_events_pkey PRIMARY KEY (id);


--
-- Name: external_events external_events_user_id_external_uid_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_events
    ADD CONSTRAINT external_events_user_id_external_uid_key UNIQUE (user_id, external_uid);


--
-- Name: fact_card_junction fact_card_junction_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fact_card_junction
    ADD CONSTRAINT fact_card_junction_pkey PRIMARY KEY (fact_id, card_pk);


--
-- Name: facts facts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.facts
    ADD CONSTRAINT facts_pkey PRIMARY KEY (id);


--
-- Name: file_tags file_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_tags
    ADD CONSTRAINT file_tags_pkey PRIMARY KEY (id);


--
-- Name: files files_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_pkey PRIMARY KEY (id);


--
-- Name: files_tags files_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.files_tags
    ADD CONSTRAINT files_tags_pkey PRIMARY KEY (file_id, tag_id);


--
-- Name: habit_logs habit_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.habit_logs
    ADD CONSTRAINT habit_logs_pkey PRIMARY KEY (id);


--
-- Name: habits habits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.habits
    ADD CONSTRAINT habits_pkey PRIMARY KEY (id);


--
-- Name: inactive_cards inactive_cards_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inactive_cards
    ADD CONSTRAINT inactive_cards_pkey PRIMARY KEY (id);


--
-- Name: keywords keywords_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.keywords
    ADD CONSTRAINT keywords_pkey PRIMARY KEY (id);


--
-- Name: llm_jobs llm_jobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_jobs
    ADD CONSTRAINT llm_jobs_pkey PRIMARY KEY (id);


--
-- Name: llm_models llm_models_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_models
    ADD CONSTRAINT llm_models_pkey PRIMARY KEY (id);


--
-- Name: llm_models llm_models_provider_id_model_identifier_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_models
    ADD CONSTRAINT llm_models_provider_id_model_identifier_key UNIQUE (provider_id, model_identifier);


--
-- Name: llm_providers llm_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_providers
    ADD CONSTRAINT llm_providers_pkey PRIMARY KEY (id);


--
-- Name: llm_query_log llm_query_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_query_log
    ADD CONSTRAINT llm_query_log_pkey PRIMARY KEY (id);


--
-- Name: mailing_list_messages mailing_list_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mailing_list_messages
    ADD CONSTRAINT mailing_list_messages_pkey PRIMARY KEY (id);


--
-- Name: mailing_list mailing_list_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mailing_list
    ADD CONSTRAINT mailing_list_pkey PRIMARY KEY (id);


--
-- Name: mailing_list_recipients mailing_list_recipients_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mailing_list_recipients
    ADD CONSTRAINT mailing_list_recipients_pkey PRIMARY KEY (id);


--
-- Name: migrations migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.migrations
    ADD CONSTRAINT migrations_pkey PRIMARY KEY (id);


--
-- Name: notification_preferences notification_preferences_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_preferences
    ADD CONSTRAINT notification_preferences_pkey PRIMARY KEY (user_id);


--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


--
-- Name: notifications notifications_user_id_source_type_source_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_user_id_source_type_source_id_key UNIQUE (user_id, source_type, source_id);


--
-- Name: starred_cards pinned_cards_card_pk_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.starred_cards
    ADD CONSTRAINT pinned_cards_card_pk_user_id_key UNIQUE (card_pk, user_id);


--
-- Name: starred_cards pinned_cards_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.starred_cards
    ADD CONSTRAINT pinned_cards_pkey PRIMARY KEY (id);


--
-- Name: starred_searches pinned_searches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.starred_searches
    ADD CONSTRAINT pinned_searches_pkey PRIMARY KEY (id);


--
-- Name: revenue revenue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revenue
    ADD CONSTRAINT revenue_pkey PRIMARY KEY (id);


--
-- Name: rss_articles rss_articles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_articles
    ADD CONSTRAINT rss_articles_pkey PRIMARY KEY (id);


--
-- Name: rss_articles rss_articles_user_id_url_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_articles
    ADD CONSTRAINT rss_articles_user_id_url_key UNIQUE (user_id, url);


--
-- Name: rss_feeds rss_feeds_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_feeds
    ADD CONSTRAINT rss_feeds_pkey PRIMARY KEY (id);


--
-- Name: rss_feeds rss_feeds_user_id_url_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_feeds
    ADD CONSTRAINT rss_feeds_user_id_url_key UNIQUE (user_id, url);


--
-- Name: rss_folders rss_folders_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_folders
    ADD CONSTRAINT rss_folders_pkey PRIMARY KEY (id);


--
-- Name: rss_folders rss_folders_user_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_folders
    ADD CONSTRAINT rss_folders_user_id_name_key UNIQUE (user_id, name);


--
-- Name: rss_seen_articles rss_seen_articles_feed_id_url_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_seen_articles
    ADD CONSTRAINT rss_seen_articles_feed_id_url_key UNIQUE (feed_id, url);


--
-- Name: rss_seen_articles rss_seen_articles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_seen_articles
    ADD CONSTRAINT rss_seen_articles_pkey PRIMARY KEY (id);


--
-- Name: scheduled_job_runs scheduled_job_runs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.scheduled_job_runs
    ADD CONSTRAINT scheduled_job_runs_pkey PRIMARY KEY (id);


--
-- Name: schema_definitions schema_definitions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_definitions
    ADD CONSTRAINT schema_definitions_pkey PRIMARY KEY (id);


--
-- Name: spreadsheets spreadsheets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.spreadsheets
    ADD CONSTRAINT spreadsheets_pkey PRIMARY KEY (id);


--
-- Name: stripe_plans stripe_plans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stripe_plans
    ADD CONSTRAINT stripe_plans_pkey PRIMARY KEY (id);


--
-- Name: stripe_plans stripe_price_id_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stripe_plans
    ADD CONSTRAINT stripe_price_id_unique UNIQUE (stripe_price_id);


--
-- Name: summarizations summarizations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summarizations
    ADD CONSTRAINT summarizations_pkey PRIMARY KEY (id);


--
-- Name: summary_arguments summary_arguments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_arguments
    ADD CONSTRAINT summary_arguments_pkey PRIMARY KEY (id);


--
-- Name: summary_sections summary_sections_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_sections
    ADD CONSTRAINT summary_sections_pkey PRIMARY KEY (id);


--
-- Name: summary_theses summary_theses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_theses
    ADD CONSTRAINT summary_theses_pkey PRIMARY KEY (id);


--
-- Name: tags tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT tags_pkey PRIMARY KEY (id);


--
-- Name: task_dependencies task_dependencies_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_dependencies
    ADD CONSTRAINT task_dependencies_pkey PRIMARY KEY (id);


--
-- Name: task_dependencies task_dependencies_task_id_blocking_task_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_dependencies
    ADD CONSTRAINT task_dependencies_task_id_blocking_task_id_key UNIQUE (task_id, blocking_task_id);


--
-- Name: task_statuses task_statuses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_statuses
    ADD CONSTRAINT task_statuses_pkey PRIMARY KEY (id);


--
-- Name: task_statuses task_statuses_user_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_statuses
    ADD CONSTRAINT task_statuses_user_id_name_key UNIQUE (user_id, name);


--
-- Name: task_statuses task_statuses_user_id_position_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_statuses
    ADD CONSTRAINT task_statuses_user_id_position_key UNIQUE (user_id, "position");


--
-- Name: task_tags task_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_tags
    ADD CONSTRAINT task_tags_pkey PRIMARY KEY (task_pk, tag_id);


--
-- Name: tasks tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_pkey PRIMARY KEY (id);


--
-- Name: file_tags unique_user_tag; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_tags
    ADD CONSTRAINT unique_user_tag UNIQUE (user_id, name);


--
-- Name: unsorted_cards unsorted_cards_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.unsorted_cards
    ADD CONSTRAINT unsorted_cards_pkey PRIMARY KEY (id);


--
-- Name: user_llm_configurations user_llm_configurations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_llm_configurations
    ADD CONSTRAINT user_llm_configurations_pkey PRIMARY KEY (id);


--
-- Name: user_llm_configurations user_llm_configurations_user_id_model_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_llm_configurations
    ADD CONSTRAINT user_llm_configurations_user_id_model_id_key UNIQUE (user_id, model_id);


--
-- Name: user_memories user_memories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_memories
    ADD CONSTRAINT user_memories_pkey PRIMARY KEY (id);


--
-- Name: user_memories user_memories_user_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_memories
    ADD CONSTRAINT user_memories_user_id_key UNIQUE (user_id);


--
-- Name: user_stats user_stats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_stats
    ADD CONSTRAINT user_stats_pkey PRIMARY KEY (user_id);


--
-- Name: users users_caldav_token_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_caldav_token_key UNIQUE (caldav_token);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_admin_audit_log_action; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_admin_audit_log_action ON public.admin_audit_log USING btree (action);


--
-- Name: idx_admin_audit_log_admin_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_admin_audit_log_admin_user_id ON public.admin_audit_log USING btree (admin_user_id);


--
-- Name: idx_admin_audit_log_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_admin_audit_log_created_at ON public.admin_audit_log USING btree (created_at DESC);


--
-- Name: idx_admin_audit_log_target; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_admin_audit_log_target ON public.admin_audit_log USING btree (target_type, target_id);


--
-- Name: idx_agent_activity_action; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_activity_action ON public.agent_activity_log USING btree (action);


--
-- Name: idx_agent_activity_agent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_activity_agent ON public.agent_activity_log USING btree (agent_id);


--
-- Name: idx_agent_activity_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_agent_activity_created ON public.agent_activity_log USING btree (created_at DESC);


--
-- Name: idx_api_keys_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_api_keys_user_id ON public.api_keys USING btree (user_id);


--
-- Name: idx_card_templates_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_card_templates_user_id ON public.card_templates USING btree (user_id);


--
-- Name: idx_cards_card_schema_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_cards_card_schema_id ON public.cards USING btree (card_schema_id);


--
-- Name: idx_cards_created_by_agent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_cards_created_by_agent ON public.cards USING btree (created_by_agent_id) WHERE (created_by_agent_id IS NOT NULL);


--
-- Name: idx_cards_user_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_cards_user_created ON public.cards USING btree (user_id, created_at) WHERE (is_deleted = false);


--
-- Name: idx_chat_conversations_primary_card_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_conversations_primary_card_id ON public.chat_conversations USING btree (primary_card_id);


--
-- Name: idx_chat_conversations_updated_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_conversations_updated_at ON public.chat_conversations USING btree (updated_at DESC);


--
-- Name: idx_chat_conversations_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_conversations_user_id ON public.chat_conversations USING btree (user_id);


--
-- Name: idx_chat_instructions_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_instructions_user_id ON public.chat_instructions USING btree (user_id);


--
-- Name: idx_chat_messages_conversation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_messages_conversation_id ON public.chat_messages USING btree (conversation_id);


--
-- Name: idx_chat_messages_conversation_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_messages_conversation_status ON public.chat_messages USING btree (conversation_id, status);


--
-- Name: idx_chat_messages_referenced_cards; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_messages_referenced_cards ON public.chat_messages USING gin (referenced_cards);


--
-- Name: idx_chat_messages_sequence; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_messages_sequence ON public.chat_messages USING btree (conversation_id, sequence_number);


--
-- Name: idx_chat_messages_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_messages_status ON public.chat_messages USING btree (status);


--
-- Name: idx_chat_tool_calls_conversation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_tool_calls_conversation_id ON public.chat_tool_calls USING btree (conversation_id);


--
-- Name: idx_chat_tool_calls_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_tool_calls_user_id ON public.chat_tool_calls USING btree (user_id);


--
-- Name: idx_chat_usage_quotas_user_reset; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_chat_usage_quotas_user_reset ON public.chat_usage_quotas USING btree (user_id, reset_date);


--
-- Name: idx_email_accounts_active_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_accounts_active_user ON public.email_accounts USING btree (user_id) WHERE (is_active = true);


--
-- Name: idx_email_accounts_is_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_accounts_is_active ON public.email_accounts USING btree (is_active);


--
-- Name: idx_email_accounts_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_accounts_user_id ON public.email_accounts USING btree (user_id);


--
-- Name: idx_email_attachments_email_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_attachments_email_id ON public.email_attachments USING btree (email_id);


--
-- Name: idx_email_attachments_file_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_attachments_file_id ON public.email_attachments USING btree (file_id);


--
-- Name: idx_email_attachments_inline; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_attachments_inline ON public.email_attachments USING btree (email_id, is_inline);


--
-- Name: idx_email_attachments_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_attachments_user_id ON public.email_attachments USING btree (user_id);


--
-- Name: idx_email_card_links_card_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_card_links_card_id ON public.email_card_links USING btree (card_id);


--
-- Name: idx_email_card_links_email_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_card_links_email_id ON public.email_card_links USING btree (email_id);


--
-- Name: idx_email_fact_junction_email_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_fact_junction_email_id ON public.email_fact_junction USING btree (email_id);


--
-- Name: idx_email_fact_junction_fact_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_fact_junction_fact_id ON public.email_fact_junction USING btree (fact_id);


--
-- Name: idx_email_fact_junction_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_fact_junction_user_id ON public.email_fact_junction USING btree (user_id);


--
-- Name: idx_email_triage_decisions_email_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_email_triage_decisions_email_id ON public.email_triage_decisions USING btree (email_id);


--
-- Name: idx_emails_card_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_emails_card_id ON public.emails USING btree (card_id);


--
-- Name: idx_emails_imap_uid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_emails_imap_uid ON public.emails USING btree (imap_uid) WHERE (imap_uid IS NOT NULL);


--
-- Name: idx_emails_received_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_emails_received_at ON public.emails USING btree (received_at DESC);


--
-- Name: idx_emails_user_account; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_emails_user_account ON public.emails USING btree (user_id, email_account_id);


--
-- Name: idx_emails_user_folder; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_emails_user_folder ON public.emails USING btree (user_id, folder);


--
-- Name: idx_emails_user_is_read_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_emails_user_is_read_status ON public.emails USING btree (user_id, is_read, status) WHERE ((is_read = false) AND (status = 'unprocessed'::text));


--
-- Name: idx_emails_user_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_emails_user_status ON public.emails USING btree (user_id, status);


--
-- Name: idx_external_calendars_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_external_calendars_user ON public.external_calendars USING btree (user_id);


--
-- Name: idx_external_events_calendar; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_external_events_calendar ON public.external_events USING btree (external_calendar_id);


--
-- Name: idx_external_events_card_pk; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_external_events_card_pk ON public.external_events USING btree (card_pk);


--
-- Name: idx_external_events_recurrence_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_external_events_recurrence_id ON public.external_events USING btree (recurrence_id);


--
-- Name: idx_external_events_recurrence_series; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_external_events_recurrence_series ON public.external_events USING btree (user_id, recurrence_id);


--
-- Name: idx_external_events_user_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_external_events_user_time ON public.external_events USING btree (user_id, start_time, end_time);


--
-- Name: idx_file_tags_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_file_tags_user_id ON public.file_tags USING btree (user_id);


--
-- Name: idx_files_extracted_text; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_files_extracted_text ON public.files USING gin (to_tsvector('english'::regconfig, extracted_text));


--
-- Name: idx_files_tags_file_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_files_tags_file_id ON public.files_tags USING btree (file_id);


--
-- Name: idx_files_tags_tag_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_files_tags_tag_id ON public.files_tags USING btree (tag_id);


--
-- Name: idx_habit_logs_habit_completed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_habit_logs_habit_completed ON public.habit_logs USING btree (habit_id, completed_at DESC);


--
-- Name: idx_habit_logs_habit_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_habit_logs_habit_date ON public.habit_logs USING btree (habit_id, date((completed_at AT TIME ZONE 'UTC'::text)));


--
-- Name: idx_habit_logs_user_completed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_habit_logs_user_completed ON public.habit_logs USING btree (user_id, completed_at DESC);


--
-- Name: idx_habits_linked_task; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_habits_linked_task ON public.habits USING btree (linked_task_id);


--
-- Name: idx_habits_position; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_habits_position ON public.habits USING btree (user_id, "position");


--
-- Name: idx_habits_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_habits_user_id ON public.habits USING btree (user_id);


--
-- Name: idx_llm_jobs_correlation_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_llm_jobs_correlation_id ON public.llm_jobs USING btree (correlation_id) WHERE (correlation_id IS NOT NULL);


--
-- Name: idx_llm_jobs_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_llm_jobs_created_at ON public.llm_jobs USING btree (created_at DESC);


--
-- Name: idx_llm_jobs_priority; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_llm_jobs_priority ON public.llm_jobs USING btree (priority) WHERE ((status)::text = 'pending'::text);


--
-- Name: idx_llm_jobs_running_heartbeat; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_llm_jobs_running_heartbeat ON public.llm_jobs USING btree (id, started_at, last_heartbeat) WHERE ((status)::text = 'running'::text);


--
-- Name: idx_llm_jobs_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_llm_jobs_status ON public.llm_jobs USING btree (status) WHERE ((status)::text = ANY ((ARRAY['pending'::character varying, 'running'::character varying])::text[]));


--
-- Name: idx_llm_jobs_updated_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_llm_jobs_updated_at ON public.llm_jobs USING btree (updated_at DESC);


--
-- Name: idx_llm_jobs_user_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_llm_jobs_user_status ON public.llm_jobs USING btree (user_id, status);


--
-- Name: idx_notifications_filter_tags; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_filter_tags ON public.notifications USING gin (filter_tags);


--
-- Name: idx_notifications_user_timestamp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_user_timestamp ON public.notifications USING btree (user_id, "timestamp" DESC);


--
-- Name: idx_notifications_user_unread; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_notifications_user_unread ON public.notifications USING btree (user_id, is_read, is_archived) WHERE ((is_read = false) AND (is_archived = false));


--
-- Name: idx_rss_articles_card_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rss_articles_card_id ON public.rss_articles USING btree (card_id);


--
-- Name: idx_rss_articles_read; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rss_articles_read ON public.rss_articles USING btree (user_id, read);


--
-- Name: idx_rss_articles_starred; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rss_articles_starred ON public.rss_articles USING btree (user_id, is_starred);


--
-- Name: idx_rss_articles_user_feed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rss_articles_user_feed ON public.rss_articles USING btree (user_id, feed_id);


--
-- Name: idx_rss_feeds_enabled; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rss_feeds_enabled ON public.rss_feeds USING btree (enabled);


--
-- Name: idx_rss_feeds_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rss_feeds_user ON public.rss_feeds USING btree (user_id);


--
-- Name: idx_rss_folders_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rss_folders_user ON public.rss_folders USING btree (user_id);


--
-- Name: idx_rss_seen_articles_feed_url; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rss_seen_articles_feed_url ON public.rss_seen_articles USING btree (feed_id, url);


--
-- Name: idx_scheduled_job_runs_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_scheduled_job_runs_created_at ON public.scheduled_job_runs USING btree (created_at);


--
-- Name: idx_scheduled_job_runs_job_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_scheduled_job_runs_job_name ON public.scheduled_job_runs USING btree (job_name);


--
-- Name: idx_scheduled_job_runs_started_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_scheduled_job_runs_started_at ON public.scheduled_job_runs USING btree (started_at DESC);


--
-- Name: idx_scheduled_job_runs_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_scheduled_job_runs_status ON public.scheduled_job_runs USING btree (status);


--
-- Name: idx_schema_definitions_owner_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_schema_definitions_owner_id ON public.schema_definitions USING btree (owner_id);


--
-- Name: idx_schema_definitions_owner_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_schema_definitions_owner_name ON public.schema_definitions USING btree (owner_id, lower(TRIM(BOTH FROM name))) WHERE (is_deleted = false);


--
-- Name: idx_schema_definitions_owner_slug; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_schema_definitions_owner_slug ON public.schema_definitions USING btree (owner_id, slug);


--
-- Name: idx_spreadsheets_card_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_spreadsheets_card_id ON public.spreadsheets USING btree (card_id);


--
-- Name: idx_spreadsheets_user_card; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_spreadsheets_user_card ON public.spreadsheets USING btree (user_id, card_id);


--
-- Name: idx_spreadsheets_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_spreadsheets_user_id ON public.spreadsheets USING btree (user_id);


--
-- Name: idx_summarizations_llm_job_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_summarizations_llm_job_id ON public.summarizations USING btree (llm_job_id);


--
-- Name: idx_summarizations_user_card_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_summarizations_user_card_created ON public.summarizations USING btree (user_id, card_pk, created_at DESC);


--
-- Name: idx_summarizations_user_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_summarizations_user_created ON public.summarizations USING btree (user_id, created_at DESC);


--
-- Name: idx_task_dependencies_blocking_task_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_dependencies_blocking_task_id ON public.task_dependencies USING btree (blocking_task_id);


--
-- Name: idx_task_dependencies_task_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_dependencies_task_id ON public.task_dependencies USING btree (task_id);


--
-- Name: idx_task_statuses_position; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_statuses_position ON public.task_statuses USING btree (user_id, "position");


--
-- Name: idx_task_statuses_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_task_statuses_user ON public.task_statuses USING btree (user_id);


--
-- Name: idx_tasks_external_uid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_external_uid ON public.tasks USING btree (external_uid) WHERE (external_uid IS NOT NULL);


--
-- Name: idx_tasks_parent_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_parent_id ON public.tasks USING btree (parent_task_id);


--
-- Name: idx_tasks_sort_order; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_sort_order ON public.tasks USING btree (user_id, sort_order) WHERE (sort_order IS NOT NULL);


--
-- Name: idx_tasks_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_status ON public.tasks USING btree (status);


--
-- Name: idx_tasks_user_completed; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_user_completed ON public.tasks USING btree (user_id, completed_at) WHERE ((is_deleted = false) AND (completed_at IS NOT NULL));


--
-- Name: idx_tasks_user_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tasks_user_created ON public.tasks USING btree (user_id, created_at) WHERE (is_deleted = false);


--
-- Name: idx_unique_active_key_name_per_user; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_unique_active_key_name_per_user ON public.api_keys USING btree (user_id, name) WHERE (is_active = true);


--
-- Name: idx_user_stats_card_count; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_stats_card_count ON public.user_stats USING btree (card_count);


--
-- Name: idx_user_stats_revenue; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_stats_revenue ON public.user_stats USING btree (revenue_cents DESC);


--
-- Name: idx_users_agent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_agent ON public.users USING btree (is_agent) WHERE (is_agent = true);


--
-- Name: idx_users_caldav_token; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_caldav_token ON public.users USING btree (caldav_token) WHERE (caldav_token IS NOT NULL);


--
-- Name: idx_users_last_memory_job_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_last_memory_job_id ON public.users USING btree (last_memory_job_id);


--
-- Name: idx_users_owner; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_owner ON public.users USING btree (owner_user_id) WHERE (is_agent = true);


--
-- Name: starred_cards_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX starred_cards_user_id_idx ON public.starred_cards USING btree (user_id);


--
-- Name: starred_searches_user_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX starred_searches_user_id_idx ON public.starred_searches USING btree (user_id);


--
-- Name: emails email_delete_notification_trigger; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER email_delete_notification_trigger AFTER DELETE ON public.emails FOR EACH ROW EXECUTE FUNCTION public.delete_notification();


--
-- Name: emails email_notification_trigger; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER email_notification_trigger AFTER INSERT OR UPDATE ON public.emails FOR EACH ROW EXECUTE FUNCTION public.sync_email_notification();


--
-- Name: llm_jobs llm_job_insert_trigger; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER llm_job_insert_trigger AFTER INSERT ON public.llm_jobs FOR EACH STATEMENT WHEN ((pg_trigger_depth() = 0)) EXECUTE FUNCTION public.notify_llm_job_insert();


--
-- Name: rss_articles rss_article_delete_notification_trigger; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER rss_article_delete_notification_trigger AFTER DELETE ON public.rss_articles FOR EACH ROW EXECUTE FUNCTION public.delete_notification();


--
-- Name: rss_articles rss_article_notification_trigger; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER rss_article_notification_trigger AFTER INSERT OR UPDATE ON public.rss_articles FOR EACH ROW EXECUTE FUNCTION public.sync_rss_notification();


--
-- Name: cards trg_cards_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_cards_delete AFTER DELETE OR UPDATE OF is_deleted ON public.cards FOR EACH ROW EXECUTE FUNCTION public.update_user_card_count_delete();


--
-- Name: cards trg_cards_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_cards_insert AFTER INSERT ON public.cards FOR EACH ROW EXECUTE FUNCTION public.update_user_card_count_insert();


--
-- Name: chat_messages trg_chat_messages_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_chat_messages_insert AFTER INSERT ON public.chat_messages FOR EACH ROW EXECUTE FUNCTION public.update_user_chat_message_count();


--
-- Name: files trg_files_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_files_delete AFTER DELETE ON public.files FOR EACH ROW EXECUTE FUNCTION public.update_user_file_count_delete();


--
-- Name: files trg_files_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_files_insert AFTER INSERT ON public.files FOR EACH ROW EXECUTE FUNCTION public.update_user_file_count_insert();


--
-- Name: llm_query_log trg_llm_query_log_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_llm_query_log_insert AFTER INSERT ON public.llm_query_log FOR EACH ROW EXECUTE FUNCTION public.update_user_llm_cost();


--
-- Name: revenue trg_revenue_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_revenue_insert AFTER INSERT ON public.revenue FOR EACH ROW EXECUTE FUNCTION public.update_user_revenue();


--
-- Name: tasks trg_tasks_delete; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_tasks_delete AFTER DELETE OR UPDATE OF is_deleted ON public.tasks FOR EACH ROW EXECUTE FUNCTION public.update_user_task_count_delete();


--
-- Name: tasks trg_tasks_insert; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_tasks_insert AFTER INSERT ON public.tasks FOR EACH ROW EXECUTE FUNCTION public.update_user_task_count_insert();


--
-- Name: llm_jobs trigger_llm_jobs_updated_at; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_llm_jobs_updated_at BEFORE UPDATE ON public.llm_jobs FOR EACH ROW EXECUTE FUNCTION public.update_llm_jobs_updated_at();


--
-- Name: chat_messages trigger_update_conversation_timestamp; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trigger_update_conversation_timestamp AFTER INSERT ON public.chat_messages FOR EACH ROW EXECUTE FUNCTION public.update_conversation_timestamp();


--
-- Name: admin_audit_log admin_audit_log_admin_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_audit_log
    ADD CONSTRAINT admin_audit_log_admin_user_id_fkey FOREIGN KEY (admin_user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: agent_activity_log agent_activity_log_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.agent_activity_log
    ADD CONSTRAINT agent_activity_log_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: api_keys api_keys_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT api_keys_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: card_chunks card_chunks_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_chunks
    ADD CONSTRAINT card_chunks_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id);


--
-- Name: card_chunks card_chunks_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_chunks
    ADD CONSTRAINT card_chunks_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: card_embeddings card_embeddings_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_embeddings
    ADD CONSTRAINT card_embeddings_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id);


--
-- Name: card_embeddings card_embeddings_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_embeddings
    ADD CONSTRAINT card_embeddings_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: card_tags card_tags_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_tags
    ADD CONSTRAINT card_tags_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id) ON DELETE CASCADE;


--
-- Name: card_tags card_tags_tag_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_tags
    ADD CONSTRAINT card_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.tags(id) ON DELETE CASCADE;


--
-- Name: card_templates card_templates_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_templates
    ADD CONSTRAINT card_templates_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: card_views card_views_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_views
    ADD CONSTRAINT card_views_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id);


--
-- Name: card_views card_views_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_views
    ADD CONSTRAINT card_views_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: card_views card_views_user_id_fkey1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.card_views
    ADD CONSTRAINT card_views_user_id_fkey1 FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: cards cards_card_schema_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cards
    ADD CONSTRAINT cards_card_schema_id_fkey FOREIGN KEY (card_schema_id) REFERENCES public.schema_definitions(id);


--
-- Name: cards cards_created_by_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cards
    ADD CONSTRAINT cards_created_by_agent_id_fkey FOREIGN KEY (created_by_agent_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: cards cards_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.cards
    ADD CONSTRAINT cards_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: categories categories_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categories
    ADD CONSTRAINT categories_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: categories categories_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categories
    ADD CONSTRAINT categories_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: categories categories_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categories
    ADD CONSTRAINT categories_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: chat_conversations chat_conversations_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_conversations
    ADD CONSTRAINT chat_conversations_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: chat_instructions chat_instructions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_instructions
    ADD CONSTRAINT chat_instructions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: chat_messages chat_messages_conversation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_messages
    ADD CONSTRAINT chat_messages_conversation_id_fkey FOREIGN KEY (conversation_id) REFERENCES public.chat_conversations(id) ON DELETE CASCADE;


--
-- Name: chat_tool_calls chat_tool_calls_conversation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_tool_calls
    ADD CONSTRAINT chat_tool_calls_conversation_id_fkey FOREIGN KEY (conversation_id) REFERENCES public.chat_conversations(id) ON DELETE CASCADE;


--
-- Name: chat_tool_calls chat_tool_calls_message_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_tool_calls
    ADD CONSTRAINT chat_tool_calls_message_id_fkey FOREIGN KEY (message_id) REFERENCES public.chat_messages(id) ON DELETE CASCADE;


--
-- Name: chat_tool_calls chat_tool_calls_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_tool_calls
    ADD CONSTRAINT chat_tool_calls_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: chat_usage_quotas chat_usage_quotas_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_usage_quotas
    ADD CONSTRAINT chat_usage_quotas_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: email_accounts email_accounts_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_accounts
    ADD CONSTRAINT email_accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: email_attachments email_attachments_email_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_attachments
    ADD CONSTRAINT email_attachments_email_id_fkey FOREIGN KEY (email_id) REFERENCES public.emails(id) ON DELETE CASCADE;


--
-- Name: email_attachments email_attachments_file_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_attachments
    ADD CONSTRAINT email_attachments_file_id_fkey FOREIGN KEY (file_id) REFERENCES public.files(id) ON DELETE SET NULL;


--
-- Name: email_attachments email_attachments_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_attachments
    ADD CONSTRAINT email_attachments_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: email_card_links email_card_links_card_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_card_links
    ADD CONSTRAINT email_card_links_card_id_fkey FOREIGN KEY (card_id) REFERENCES public.cards(id) ON DELETE CASCADE;


--
-- Name: email_card_links email_card_links_email_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_card_links
    ADD CONSTRAINT email_card_links_email_id_fkey FOREIGN KEY (email_id) REFERENCES public.emails(id) ON DELETE CASCADE;


--
-- Name: email_fact_junction email_fact_junction_email_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_fact_junction
    ADD CONSTRAINT email_fact_junction_email_id_fkey FOREIGN KEY (email_id) REFERENCES public.emails(id) ON DELETE CASCADE;


--
-- Name: email_fact_junction email_fact_junction_fact_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_fact_junction
    ADD CONSTRAINT email_fact_junction_fact_id_fkey FOREIGN KEY (fact_id) REFERENCES public.facts(id) ON DELETE CASCADE;


--
-- Name: email_fact_junction email_fact_junction_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_fact_junction
    ADD CONSTRAINT email_fact_junction_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: email_triage_decisions email_triage_decisions_email_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_triage_decisions
    ADD CONSTRAINT email_triage_decisions_email_id_fkey FOREIGN KEY (email_id) REFERENCES public.emails(id) ON DELETE CASCADE;


--
-- Name: emails emails_card_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.emails
    ADD CONSTRAINT emails_card_id_fkey FOREIGN KEY (card_id) REFERENCES public.cards(id) ON DELETE SET NULL;


--
-- Name: emails emails_email_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.emails
    ADD CONSTRAINT emails_email_account_id_fkey FOREIGN KEY (email_account_id) REFERENCES public.email_accounts(id) ON DELETE SET NULL;


--
-- Name: emails emails_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.emails
    ADD CONSTRAINT emails_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: entities entities_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entities
    ADD CONSTRAINT entities_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: entity_card_junction entity_card_junction_entity_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_card_junction
    ADD CONSTRAINT entity_card_junction_entity_id_fkey FOREIGN KEY (entity_id) REFERENCES public.entities(id);


--
-- Name: entity_card_junction entity_card_junction_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_card_junction
    ADD CONSTRAINT entity_card_junction_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: entity_fact_junction entity_fact_junction_entity_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_fact_junction
    ADD CONSTRAINT entity_fact_junction_entity_id_fkey FOREIGN KEY (entity_id) REFERENCES public.entities(id) ON DELETE CASCADE;


--
-- Name: entity_fact_junction entity_fact_junction_fact_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entity_fact_junction
    ADD CONSTRAINT entity_fact_junction_fact_id_fkey FOREIGN KEY (fact_id) REFERENCES public.facts(id) ON DELETE CASCADE;


--
-- Name: external_calendars external_calendars_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_calendars
    ADD CONSTRAINT external_calendars_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: external_events external_events_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_events
    ADD CONSTRAINT external_events_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id) ON DELETE SET NULL;


--
-- Name: external_events external_events_external_calendar_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_events
    ADD CONSTRAINT external_events_external_calendar_id_fkey FOREIGN KEY (external_calendar_id) REFERENCES public.external_calendars(id) ON DELETE SET NULL;


--
-- Name: external_events external_events_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.external_events
    ADD CONSTRAINT external_events_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: fact_card_junction fact_card_junction_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fact_card_junction
    ADD CONSTRAINT fact_card_junction_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id) ON DELETE CASCADE;


--
-- Name: fact_card_junction fact_card_junction_fact_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fact_card_junction
    ADD CONSTRAINT fact_card_junction_fact_id_fkey FOREIGN KEY (fact_id) REFERENCES public.facts(id) ON DELETE CASCADE;


--
-- Name: facts facts_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.facts
    ADD CONSTRAINT facts_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id) ON DELETE CASCADE;


--
-- Name: facts facts_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.facts
    ADD CONSTRAINT facts_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: file_tags file_tags_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_tags
    ADD CONSTRAINT file_tags_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: files files_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: files_tags files_tags_file_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.files_tags
    ADD CONSTRAINT files_tags_file_id_fkey FOREIGN KEY (file_id) REFERENCES public.files(id) ON DELETE CASCADE;


--
-- Name: files_tags files_tags_tag_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.files_tags
    ADD CONSTRAINT files_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.file_tags(id) ON DELETE CASCADE;


--
-- Name: files files_updated_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: entities fk_card_pk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.entities
    ADD CONSTRAINT fk_card_pk FOREIGN KEY (card_pk) REFERENCES public.cards(id);


--
-- Name: chat_conversations fk_primary_card_id; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.chat_conversations
    ADD CONSTRAINT fk_primary_card_id FOREIGN KEY (primary_card_id) REFERENCES public.cards(id) ON DELETE SET NULL;


--
-- Name: flashcard_reviews flashcard_reviews_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flashcard_reviews
    ADD CONSTRAINT flashcard_reviews_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id);


--
-- Name: flashcard_reviews flashcard_reviews_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.flashcard_reviews
    ADD CONSTRAINT flashcard_reviews_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: habit_logs habit_logs_habit_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.habit_logs
    ADD CONSTRAINT habit_logs_habit_id_fkey FOREIGN KEY (habit_id) REFERENCES public.habits(id) ON DELETE CASCADE;


--
-- Name: habit_logs habit_logs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.habit_logs
    ADD CONSTRAINT habit_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: habits habits_linked_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.habits
    ADD CONSTRAINT habits_linked_task_id_fkey FOREIGN KEY (linked_task_id) REFERENCES public.tasks(id) ON DELETE SET NULL;


--
-- Name: habits habits_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.habits
    ADD CONSTRAINT habits_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: keywords keywords_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.keywords
    ADD CONSTRAINT keywords_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id);


--
-- Name: llm_jobs llm_jobs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_jobs
    ADD CONSTRAINT llm_jobs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: llm_models llm_models_provider_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_models
    ADD CONSTRAINT llm_models_provider_id_fkey FOREIGN KEY (provider_id) REFERENCES public.llm_providers(id);


--
-- Name: llm_providers llm_providers_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_providers
    ADD CONSTRAINT llm_providers_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: llm_query_log llm_query_log_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.llm_query_log
    ADD CONSTRAINT llm_query_log_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: mailing_list_recipients mailing_list_recipients_message_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.mailing_list_recipients
    ADD CONSTRAINT mailing_list_recipients_message_id_fkey FOREIGN KEY (message_id) REFERENCES public.mailing_list_messages(id);


--
-- Name: notification_preferences notification_preferences_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_preferences
    ADD CONSTRAINT notification_preferences_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: notifications notifications_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: starred_cards pinned_cards_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.starred_cards
    ADD CONSTRAINT pinned_cards_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id) ON DELETE CASCADE;


--
-- Name: starred_searches pinned_searches_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.starred_searches
    ADD CONSTRAINT pinned_searches_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: revenue revenue_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.revenue
    ADD CONSTRAINT revenue_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: rss_articles rss_articles_card_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_articles
    ADD CONSTRAINT rss_articles_card_id_fkey FOREIGN KEY (card_id) REFERENCES public.cards(id) ON DELETE SET NULL;


--
-- Name: rss_articles rss_articles_feed_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_articles
    ADD CONSTRAINT rss_articles_feed_id_fkey FOREIGN KEY (feed_id) REFERENCES public.rss_feeds(id) ON DELETE CASCADE;


--
-- Name: rss_articles rss_articles_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_articles
    ADD CONSTRAINT rss_articles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: rss_feeds rss_feeds_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_feeds
    ADD CONSTRAINT rss_feeds_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: rss_folders rss_folders_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_folders
    ADD CONSTRAINT rss_folders_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: rss_seen_articles rss_seen_articles_feed_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rss_seen_articles
    ADD CONSTRAINT rss_seen_articles_feed_id_fkey FOREIGN KEY (feed_id) REFERENCES public.rss_feeds(id) ON DELETE CASCADE;


--
-- Name: schema_definitions schema_definitions_owner_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_definitions
    ADD CONSTRAINT schema_definitions_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: spreadsheets spreadsheets_card_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.spreadsheets
    ADD CONSTRAINT spreadsheets_card_id_fkey FOREIGN KEY (card_id) REFERENCES public.cards(id) ON DELETE CASCADE;


--
-- Name: spreadsheets spreadsheets_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.spreadsheets
    ADD CONSTRAINT spreadsheets_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: summarizations summarizations_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summarizations
    ADD CONSTRAINT summarizations_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id) ON DELETE CASCADE;


--
-- Name: summarizations summarizations_llm_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summarizations
    ADD CONSTRAINT summarizations_llm_job_id_fkey FOREIGN KEY (llm_job_id) REFERENCES public.llm_jobs(id) ON DELETE CASCADE;


--
-- Name: summarizations summarizations_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summarizations
    ADD CONSTRAINT summarizations_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: summary_arguments summary_arguments_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_arguments
    ADD CONSTRAINT summary_arguments_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id) ON DELETE CASCADE;


--
-- Name: summary_arguments summary_arguments_summarization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_arguments
    ADD CONSTRAINT summary_arguments_summarization_id_fkey FOREIGN KEY (summarization_id) REFERENCES public.summarizations(id) ON DELETE CASCADE;


--
-- Name: summary_arguments summary_arguments_thesis_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_arguments
    ADD CONSTRAINT summary_arguments_thesis_id_fkey FOREIGN KEY (thesis_id) REFERENCES public.summary_theses(id) ON DELETE CASCADE;


--
-- Name: summary_arguments summary_arguments_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_arguments
    ADD CONSTRAINT summary_arguments_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: summary_sections summary_sections_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_sections
    ADD CONSTRAINT summary_sections_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id) ON DELETE CASCADE;


--
-- Name: summary_sections summary_sections_summarization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_sections
    ADD CONSTRAINT summary_sections_summarization_id_fkey FOREIGN KEY (summarization_id) REFERENCES public.summarizations(id) ON DELETE CASCADE;


--
-- Name: summary_sections summary_sections_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_sections
    ADD CONSTRAINT summary_sections_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: summary_theses summary_theses_card_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_theses
    ADD CONSTRAINT summary_theses_card_pk_fkey FOREIGN KEY (card_pk) REFERENCES public.cards(id) ON DELETE CASCADE;


--
-- Name: summary_theses summary_theses_section_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_theses
    ADD CONSTRAINT summary_theses_section_id_fkey FOREIGN KEY (section_id) REFERENCES public.summary_sections(id) ON DELETE CASCADE;


--
-- Name: summary_theses summary_theses_summarization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_theses
    ADD CONSTRAINT summary_theses_summarization_id_fkey FOREIGN KEY (summarization_id) REFERENCES public.summarizations(id) ON DELETE CASCADE;


--
-- Name: summary_theses summary_theses_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.summary_theses
    ADD CONSTRAINT summary_theses_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: task_dependencies task_dependencies_blocking_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_dependencies
    ADD CONSTRAINT task_dependencies_blocking_task_id_fkey FOREIGN KEY (blocking_task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;


--
-- Name: task_dependencies task_dependencies_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_dependencies
    ADD CONSTRAINT task_dependencies_task_id_fkey FOREIGN KEY (task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;


--
-- Name: task_statuses task_statuses_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_statuses
    ADD CONSTRAINT task_statuses_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: task_tags task_tags_tag_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_tags
    ADD CONSTRAINT task_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.tags(id) ON DELETE CASCADE;


--
-- Name: task_tags task_tags_task_pk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.task_tags
    ADD CONSTRAINT task_tags_task_pk_fkey FOREIGN KEY (task_pk) REFERENCES public.tasks(id) ON DELETE CASCADE;


--
-- Name: tasks tasks_external_calendar_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_external_calendar_id_fkey FOREIGN KEY (external_calendar_id) REFERENCES public.external_calendars(id) ON DELETE SET NULL;


--
-- Name: tasks tasks_parent_task_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_parent_task_id_fkey FOREIGN KEY (parent_task_id) REFERENCES public.tasks(id) ON DELETE CASCADE;


--
-- Name: user_llm_configurations user_llm_configurations_model_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_llm_configurations
    ADD CONSTRAINT user_llm_configurations_model_id_fkey FOREIGN KEY (model_id) REFERENCES public.llm_models(id);


--
-- Name: user_llm_configurations user_llm_configurations_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_llm_configurations
    ADD CONSTRAINT user_llm_configurations_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_memories user_memories_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_memories
    ADD CONSTRAINT user_memories_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_stats user_stats_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_stats
    ADD CONSTRAINT user_stats_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: users users_last_memory_job_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_last_memory_job_id_fkey FOREIGN KEY (last_memory_job_id) REFERENCES public.llm_jobs(id) ON DELETE SET NULL;


--
-- Name: users users_owner_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_owner_user_id_fkey FOREIGN KEY (owner_user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict kIb9LPvvUVEGj8cyn3RIK6SB4bbMKEaScLPEx7CKV3QDGXfBIoG7DrRAcdK5Vec

