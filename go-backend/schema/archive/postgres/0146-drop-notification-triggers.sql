-- Migration: Drop notification-sync triggers ported to Go (SQLite migration Phase 5)
-- Date: 2026-07-25
--
-- Companion to the RSS notification Go port (services/rss_notifications.go +
-- models.DeleteNotificationBySource). The Go server now maintains RSS
-- notifications on BOTH Postgres and SQLite (single code path — design doc
-- Phase 5, decision 3b); these triggers are dropped so Postgres does not
-- double-fire during the cutover window. SQLite never had them.
--
-- What is replaced and where:
--   * rss notification sync (0124) + the rss half of delete_notification (0122)
--     -> services.SyncRSSArticleNotification at every rss_articles write site
--       (insert, star/unstar, read single/feed/folder, cleanup delete) and
--       models.DeleteNotificationBySource.
--
-- Dropped as cleanup with NO Go replacement (these features are abandoned /
-- dead — no Go write path exists for them):
--   * email notification sync (0123): email ingestion is not in the Go server.
--   * email half of delete_notification (0122): same.

-- 0124: RSS article notification triggers + function
DROP TRIGGER IF EXISTS rss_article_notification_trigger ON rss_articles;
DROP TRIGGER IF EXISTS rss_article_delete_notification_trigger ON rss_articles;
DROP FUNCTION IF EXISTS sync_rss_notification();

-- 0123: email notification triggers + function (email ingestion abandoned)
DROP TRIGGER IF EXISTS email_notification_trigger ON emails;
DROP TRIGGER IF EXISTS email_delete_notification_trigger ON emails;
DROP FUNCTION IF EXISTS sync_email_notification();

-- 0122: shared delete_notification() helper (both call sites above are gone)
DROP FUNCTION IF EXISTS delete_notification();
