package handlers

import (
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"go-backend/pkg/config"
	"go-backend/server"
	"go-backend/services"
	"go-backend/settings"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

const (
	// MaxSummarizationRequestsPerMinute is the maximum number of summarization requests per user per minute
	MaxSummarizationRequestsPerMinute = 10

	// MaxConcurrentSummarizationsPerUser is the maximum number of concurrent summarization jobs per user
	MaxConcurrentSummarizationsPerUser = 3

	// SummarizationRateLimitWindow is the time window for rate limiting (1 minute)
	SummarizationRateLimitWindow = time.Minute
)

type Handler struct {
	DB             *sql.DB
	Server         *server.Server
	messageMutexes sync.Map // map[string]*sync.Mutex - per-message mutexes

	// Rate limiting and concurrency control for summarization
	summarizationRateLimits   sync.Map // map[int][]time.Time - request timestamps per user
	summarizationRateLimitMu  sync.Map // map[int]*sync.Mutex - per-user rate limit mutex
	summarizationActiveJobs   sync.Map // map[int]int - active job count per user
	summarizationActiveJobsMu sync.Map // map[int]*sync.Mutex - per-user active jobs mutex

	// JobRunner executes LLM jobs inline and records each run in the
	// llm_jobs audit table. See services.JobRunner.
	JobRunner *services.JobRunner

	// GitHub OAuth configuration (enabled by default; set GITHUB_AUTH_ENABLED
	// =false to disable, e.g. when a generic OIDC provider replaces it).
	GitHubConfig config.GitHubConfig

	// OIDC / SSO configuration (opt-in) and a lazily-discovered, cached
	// provider+oauth2 config. Discovery happens on first use of the OIDC
	// routes; the cache is process-local and never invalidated, so changing
	// OIDC_* env vars requires a restart.
	OIDCConfig   config.OIDCConfig
	oidcProvider *oidc.Provider
	oidcOAuth2   *oauth2.Config
	oidcInitMu   sync.Mutex

	// Stripe billing configuration (enabled by default; set STRIPE_ENABLED
	// =false to disable — billing routes then return 404).
	StripeConfig config.StripeConfig

	// oidcClientOverride, when non-nil, replaces the real discovery-based OIDC
	// client in CallbackOIDCRoute (test seam; production handlers leave it nil).
	// See oidc.go for the interface and the real implementation.
	oidcClientOverride oidcClient

	// Settings is the file-backed admin settings manager (config.yaml next
	// to the SQLite DB). Non-secret admin settings; env seeds it on first
	// boot. See Zettelgarden-6er.15.
	Settings *settings.Manager
}

// GetDB returns the appropriate database connection for database operations.
// During testing (when Testing=true and Tx is set), it returns the test transaction.
// Otherwise, it returns the standard database connection.
//
// This allows the same code to work in both test and production environments:
// - Tests use transactions that are rolled back after each test for isolation
// - Production uses the actual database connection
//
// CONVENTION: handler request-scoped reads and writes MUST go through GetDB()
// so they participate in the (rolled-back) per-test transaction and do not leak
// across tests. Reserve raw h.DB (the pool) for the few paths that legitimately
// need to commit independently of the request transaction:
//   - fire-and-forget audit timestamps (last_login / last_seen / last_used_at);
//   - external callbacks with no request transaction (Stripe webhooks in
//     billing.go, GitHub OAuth callbacks in oauth.go);
//   - constructing an LLM client (services.NewDefaultClient), whose DB field is
//     typed *sql.DB; and a handful of service helpers that take *sql.DB
//     (services.GetEntities) — both are reads/AI-logging, never test pollution.
//
// Writes that need atomicity should wrap themselves via BeginTx/ShouldCommitTx
// (see ReorderTaskStatusesRoute), NOT call db.Begin() on a GetDB() handle
// (which may be an already-open *sql.Tx in tests).
func (h *Handler) GetDB() models.Database {
	if h.Server != nil && h.Server.Testing && h.Server.Tx != nil {
		return h.Server.Tx
	}
	return h.DB
}

// ShouldCommitTx returns true if the handler should commit its transaction.
// Returns false during testing since the test framework manages transaction lifecycle.
func (h *Handler) ShouldCommitTx() bool {
	return !(h.Server != nil && h.Server.Testing)
}

// BeginTx returns a transaction for the handler to use.
// During testing, this returns the test's transaction (commit should be skipped).
func (h *Handler) BeginTx() (*sql.Tx, error) {
	if h.Server != nil && h.Server.Testing && h.Server.Tx != nil {
		return h.Server.Tx, nil
	}
	return h.DB.Begin()
}

// getMessageMutex gets or creates a mutex for a specific message
func (s *Handler) getMessageMutex(messageID string) *sync.Mutex {
	mu, _ := s.messageMutexes.LoadOrStore(messageID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// parseRSSArticleFilters extracts and validates common RSS article query parameters
// Returns a filters map that can be passed to service layer functions
func parseRSSArticleFilters(query url.Values) map[string]interface{} {
	filters := make(map[string]interface{})

	if folder := query.Get("folder"); folder != "" {
		filters["folder"] = folder
	}
	if unread := query.Get("unread"); unread == "true" {
		filters["unread"] = true
	}
	if starred := query.Get("starred"); starred == "true" {
		filters["starred"] = true
	}
	if feedID := query.Get("feed_id"); feedID != "" {
		if id, err := strconv.Atoi(feedID); err == nil && id >= 0 {
			filters["feed_id"] = id
		}
	}
	if limit := query.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l >= 0 {
			filters["limit"] = l
		}
	}
	if offset := query.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filters["offset"] = o
		}
	}

	return filters
}

// cleanupMessageMutex removes a mutex after a message is completed/failed
func (s *Handler) cleanupMessageMutex(messageID string) {
	s.messageMutexes.Delete(messageID)
}

// checkSummarizationRateLimit checks if a user has exceeded their rate limit for summarization requests
func (h *Handler) checkSummarizationRateLimit(userID int) bool {
	// Get or create mutex for this user
	muInt, _ := h.summarizationRateLimitMu.LoadOrStore(userID, &sync.Mutex{})
	mu := muInt.(*sync.Mutex)

	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-SummarizationRateLimitWindow)

	// Load existing timestamps
	timestampsInt, _ := h.summarizationRateLimits.LoadOrStore(userID, []time.Time{})
	timestamps := timestampsInt.([]time.Time)

	// Filter out timestamps outside the window
	var validTimestamps []time.Time
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check if limit is exceeded
	if len(validTimestamps) >= MaxSummarizationRequestsPerMinute {
		return false // Rate limit exceeded
	}

	// Add current timestamp and update map
	validTimestamps = append(validTimestamps, now)
	h.summarizationRateLimits.Store(userID, validTimestamps)

	return true // Within rate limit
}

// acquireSummarizationJobSlot attempts to acquire a slot for a new summarization job
// Returns true if successful, false if the user has reached their concurrent job limit
func (h *Handler) acquireSummarizationJobSlot(userID int) bool {
	// Get or create mutex for this user
	muInt, _ := h.summarizationActiveJobsMu.LoadOrStore(userID, &sync.Mutex{})
	mu := muInt.(*sync.Mutex)

	mu.Lock()
	defer mu.Unlock()

	// Load current job count
	countInt, _ := h.summarizationActiveJobs.LoadOrStore(userID, 0)
	count := countInt.(int)

	// Check if limit is reached
	if count >= MaxConcurrentSummarizationsPerUser {
		return false
	}

	// Increment count
	h.summarizationActiveJobs.Store(userID, count+1)
	return true
}

// releaseSummarizationJobSlot releases a slot when a summarization job completes
func (h *Handler) releaseSummarizationJobSlot(userID int) {
	// Get or create mutex for this user
	muInt, _ := h.summarizationActiveJobsMu.LoadOrStore(userID, &sync.Mutex{})
	mu := muInt.(*sync.Mutex)

	mu.Lock()
	defer mu.Unlock()

	// Load current job count
	countInt, _ := h.summarizationActiveJobs.LoadOrStore(userID, 0)
	count := countInt.(int)

	// Decrement count (but not below 0)
	if count > 0 {
		h.summarizationActiveJobs.Store(userID, count-1)
	}
}

// CreateRSSFeedRoute handles POST /api/rss/feeds
func (h *Handler) CreateRSSFeedRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var params models.CreateRSSFeedParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	feed, err := services.CreateRSSFeed(h.GetDB(), userID, params)
	if err != nil {
		log.Printf("Failed to create RSS feed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed)
}

// ListRSSFeedsRoute handles GET /api/rss/feeds
func (h *Handler) ListRSSFeedsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	feeds, err := services.ListRSSFeeds(h.GetDB(), userID)
	if err != nil {
		log.Printf("Failed to list RSS feeds: %v", err)
		http.Error(w, "Failed to list feeds", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feeds)
}

// GetRSSFeedRoute handles GET /api/rss/feeds/{id}
func (h *Handler) GetRSSFeedRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	feedID, _ := strconv.Atoi(mux.Vars(r)["id"])

	feed, err := services.GetRSSFeedByID(h.GetDB(), userID, feedID)
	if err != nil {
		log.Printf("Failed to get RSS feed: %v", err)
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed)
}

// UpdateRSSFeedRoute handles PUT /api/rss/feeds/{id}
func (h *Handler) UpdateRSSFeedRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	feedID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var params models.UpdateRSSFeedParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	feed, err := services.UpdateRSSFeed(h.GetDB(), userID, feedID, params)
	if err != nil {
		log.Printf("Failed to update RSS feed: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed)
}

// DeleteRSSFeedRoute handles DELETE /api/rss/feeds/{id}
func (h *Handler) DeleteRSSFeedRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	feedID, _ := strconv.Atoi(mux.Vars(r)["id"])

	if err := services.DeleteRSSFeed(h.GetDB(), userID, feedID); err != nil {
		log.Printf("Failed to delete RSS feed: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RefreshRSSFeedsRoute handles POST /api/rss/feeds/fetch
func (h *Handler) RefreshRSSFeedsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Get all enabled feeds for the user
	feeds, err := services.ListRSSFeeds(h.GetDB(), userID)
	if err != nil {
		log.Printf("Failed to list RSS feeds: %v", err)
		http.Error(w, "Failed to list feeds", http.StatusInternalServerError)
		return
	}

	// Fetch articles for each enabled feed
	count := 0
	for _, feed := range feeds {
		if feed.Enabled {
			if err := services.FetchRSSFeedArticles(h.GetDB(), feed.ID); err != nil {
				log.Printf("Failed to fetch feed %d: %v", feed.ID, err)
			} else {
				count++
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"fetched": count,
	})
}

// ListRSSArticlesRoute handles GET /api/rss/articles
func (h *Handler) ListRSSArticlesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse and validate query parameters using shared helper
	filters := parseRSSArticleFilters(r.URL.Query())

	// Get total count for pagination
	total, err := services.CountRSSArticles(h.GetDB(), userID, filters)
	if err != nil {
		log.Printf("Failed to count RSS articles: %v", err)
		http.Error(w, "Failed to count articles", http.StatusInternalServerError)
		return
	}

	articles, err := services.ListRSSArticles(h.GetDB(), userID, filters)
	if err != nil {
		log.Printf("Failed to list RSS articles: %v", err)
		http.Error(w, "Failed to list articles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(total))

	result := map[string]interface{}{
		"articles": articles,
		"total":    total,
	}

	json.NewEncoder(w).Encode(result)
}

// ListSmartRSSArticlesRoute handles GET /api/rss/articles/smart
func (h *Handler) ListSmartRSSArticlesRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse and validate query parameters using shared helper
	filters := parseRSSArticleFilters(r.URL.Query())

	articles, total, err := services.ListSmartRSSArticles(h.GetDB(), userID, filters)
	if err != nil {
		log.Printf("Failed to list smart RSS articles: %v", err)
		http.Error(w, "Failed to list smart feed articles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(total))

	result := map[string]interface{}{
		"articles": articles,
		"total":    total,
	}

	json.NewEncoder(w).Encode(result)
}

// GetRSSArticleRoute handles GET /api/rss/articles/{id}
func (h *Handler) GetRSSArticleRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	articleID, _ := strconv.Atoi(mux.Vars(r)["id"])

	article, err := services.GetRSSArticleByID(h.GetDB(), userID, articleID)
	if err != nil {
		log.Printf("Failed to get RSS article: %v", err)
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(article)
}

// MarkRSSArticleAsReadRoute handles POST /api/rss/articles/{id}/read
func (h *Handler) MarkRSSArticleAsReadRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	articleID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var params struct {
		Read bool `json:"read"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		params.Read = true // Default to marking as read
	}

	if err := services.MarkRSSArticleAsRead(h.GetDB(), userID, articleID, params.Read); err != nil {
		log.Printf("Failed to mark article as read: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MarkRSSFeedAsReadRoute handles POST /api/rss/feeds/{id}/read
func (h *Handler) MarkRSSFeedAsReadRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	feedID, _ := strconv.Atoi(mux.Vars(r)["id"])

	if err := services.MarkRSSFeedAsRead(h.GetDB(), userID, feedID); err != nil {
		log.Printf("Failed to mark feed as read: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MarkRSSFolderAsReadRoute handles POST /api/rss/folders/{id}/read
func (h *Handler) MarkRSSFolderAsReadRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	folderID, _ := strconv.Atoi(mux.Vars(r)["id"])

	// Get folder name
	folder, err := services.GetRSSFolderByID(h.GetDB(), userID, folderID)
	if err != nil {
		log.Printf("Failed to get folder: %v", err)
		http.Error(w, "Folder not found", http.StatusNotFound)
		return
	}

	if err := services.MarkRSSFolderAsRead(h.GetDB(), userID, folder.Name); err != nil {
		log.Printf("Failed to mark folder as read: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ConvertRSSArticleToCardRoute handles POST /api/rss/articles/{id}/convert
func (h *Handler) ConvertRSSArticleToCardRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	articleID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var params *models.ConvertArticleParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil && err.Error() != "EOF" {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	card, err := services.ConvertRSSArticleToCard(h.GetDB(), userID, articleID, params)
	if err != nil {
		log.Printf("Failed to convert article to card: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process the card after creation (memory generation + summarization for PRO users)
	h.ProcessCardAfterCreation(userID, *card, true)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

// ListRSSFoldersRoute handles GET /api/rss/folders
func (h *Handler) ListRSSFoldersRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	folders, err := services.ListRSSFolders(h.GetDB(), userID)
	if err != nil {
		log.Printf("Failed to list RSS folders: %v", err)
		http.Error(w, "Failed to list folders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)
}

// CreateRSSFolderRoute handles POST /api/rss/folders
func (h *Handler) CreateRSSFolderRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	var params models.CreateRSSFolderParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if params.Name == "" {
		http.Error(w, "Folder name is required", http.StatusBadRequest)
		return
	}

	folder, err := services.CreateRSSFolder(h.GetDB(), userID, params)
	if err != nil {
		log.Printf("Failed to create RSS folder: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(folder)
}

// UpdateRSSFolderRoute handles PUT /api/rss/folders/{id}
func (h *Handler) UpdateRSSFolderRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	folderID, _ := strconv.Atoi(mux.Vars(r)["id"])

	var params struct {
		Name       *string `json:"name"`
		OrderIndex *int    `json:"order_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	folder, err := services.UpdateRSSFolder(h.GetDB(), userID, folderID, params.Name, params.OrderIndex)
	if err != nil {
		log.Printf("Failed to update RSS folder: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folder)
}

// DeleteRSSFolderRoute handles DELETE /api/rss/folders/{id}
func (h *Handler) DeleteRSSFolderRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	folderID, _ := strconv.Atoi(mux.Vars(r)["id"])

	if err := services.DeleteRSSFolder(h.GetDB(), userID, folderID); err != nil {
		log.Printf("Failed to delete RSS folder: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetUnreadCountsRoute handles GET /api/rss/unread-counts
func (h *Handler) GetUnreadCountsRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	folderCounts, feedCounts, err := services.GetUnreadCounts(h.GetDB(), userID)
	if err != nil {
		log.Printf("Failed to get unread counts: %v", err)
		http.Error(w, "Failed to get unread counts", http.StatusInternalServerError)
		return
	}

	result := map[string]interface{}{
		"folders": folderCounts,
		"feeds":   feedCounts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ExportOPMLRoute handles GET /api/rss/opml/export
func (h *Handler) ExportOPMLRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	opml, err := services.ExportOPML(h.GetDB(), userID)
	if err != nil {
		log.Printf("Failed to export OPML: %v", err)
		http.Error(w, "Failed to export OPML", http.StatusInternalServerError)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="zettelgarden-feeds.opml"`)
	w.Write([]byte(opml))
}

// ImportOPMLRoute handles POST /api/rss/opml/import
func (h *Handler) ImportOPMLRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Printf("Failed to parse multipart form: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("Failed to get file from form: %v", err)
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file content
	content := make([]byte, header.Size)
	_, err = file.Read(content)
	if err != nil {
		log.Printf("Failed to read file content: %v", err)
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Import OPML
	result, err := services.ImportOPML(h.GetDB(), userID, content)
	if err != nil {
		log.Printf("Failed to import OPML: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// StarRSSArticleRoute handles starring an RSS article
func (h *Handler) StarRSSArticleRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	articleID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	// Verify article exists and belongs to user
	var exists bool
	err = h.GetDB().QueryRow(
		"SELECT EXISTS(SELECT 1 FROM rss_articles WHERE id = $1 AND user_id = $2)",
		articleID, userID,
	).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}

	// Star the article
	_, err = h.GetDB().Exec(
		"UPDATE rss_articles SET is_starred = TRUE WHERE id = $1 AND user_id = $2",
		articleID, userID,
	)
	if err != nil {
		log.Printf("Error starring article: %v", err)
		http.Error(w, "Failed to star article", http.StatusInternalServerError)
		return
	}

	// Notification was trigger-maintained (0124); now maintained in Go (Phase 5).
	if art, err := services.GetRSSArticleByID(h.GetDB(), userID, articleID); err == nil {
		services.SyncRSSArticleNotification(h.GetDB(), art)
	}

	w.WriteHeader(http.StatusNoContent)
}

// UnstarRSSArticleRoute handles unstarring an RSS article
func (h *Handler) UnstarRSSArticleRoute(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("current_user").(int)
	articleID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	// Unstar the article
	_, err = h.GetDB().Exec(
		"UPDATE rss_articles SET is_starred = FALSE WHERE id = $1 AND user_id = $2",
		articleID, userID,
	)
	if err != nil {
		log.Printf("Error unstarring article: %v", err)
		http.Error(w, "Failed to unstar article", http.StatusInternalServerError)
		return
	}

	// Notification was trigger-maintained (0124); now maintained in Go (Phase 5).
	if art, err := services.GetRSSArticleByID(h.GetDB(), userID, articleID); err == nil {
		services.SyncRSSArticleNotification(h.GetDB(), art)
	}

	w.WriteHeader(http.StatusNoContent)
}
