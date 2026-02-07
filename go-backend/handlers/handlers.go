package handlers

import (
	"database/sql"
	"encoding/json"
	"go-backend/models"
	"go-backend/server"
	"go-backend/services"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
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

	// Job queue rate limiting
	JobRateLimiter *services.JobRateLimiter

	// LLM worker pool for job processing
	LLMWorkerPool *services.WorkerPool

	// Encryption service for sensitive data
	EncryptionService *services.EncryptionService
}

// GetDB returns the appropriate database connection for database operations.
// During testing (when Testing=true and Tx is set), it returns the test transaction.
// Otherwise, it returns the standard database connection.
//
// This allows the same code to work in both test and production environments:
// - Tests use transactions that are rolled back after each test for isolation
// - Production uses the actual database connection
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

	// Parse query parameters
	filters := make(map[string]interface{})
	if folder := r.URL.Query().Get("folder"); folder != "" {
		filters["folder"] = folder
	}
	if unread := r.URL.Query().Get("unread"); unread == "true" {
		filters["unread"] = true
	}
	if feedID := r.URL.Query().Get("feed_id"); feedID != "" {
		if id, err := strconv.Atoi(feedID); err == nil {
			filters["feed_id"] = id
		}
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters["limit"] = l
		}
	}

	articles, err := services.ListRSSArticles(h.GetDB(), userID, filters)
	if err != nil {
		log.Printf("Failed to list RSS articles: %v", err)
		http.Error(w, "Failed to list articles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(articles)
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
