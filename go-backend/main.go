package main

import (
	//	"bytes"
	"context"
	//"encoding/json"
	"fmt"
	"go-backend/bootstrap"
	"go-backend/handlers"
	"go-backend/mail"
	"go-backend/models"
	"go-backend/server"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	openai "github.com/sashabaranov/go-openai"
	"github.com/stripe/stripe-go/v82"
)

var s *server.Server
var h *handlers.Handler

func admin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("current_user").(int)
		user, err := h.QueryUser(userID)
		if err != nil {
			http.Error(w, "User not found", http.StatusBadRequest)
			return
		}
		if !user.IsAdmin {
			http.Error(w, "Access denied", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func jwtMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.Header.Get("Authorization")

		if tokenStr == "" {
			http.Error(w, "Authorization header is missing", http.StatusUnauthorized)
			return
		}

		tokenStr = tokenStr[len("Bearer "):]

		claims := &models.Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return s.JwtSecretKey, nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				http.Error(w, "Invalid token signature", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add the claims to the request context
		ctx := context.WithValue(r.Context(), "current_user", claims.Sub)

		// Update last_seen asynchronously
		go func() {
			_, err := s.DB.Exec("UPDATE users SET last_seen = NOW() WHERE id = $1", claims.Sub)
			if err != nil {
				log.Printf("Error updating last_seen: %v", err)
			}
		}()

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func addProtectedRoute(r *mux.Router, path string, handler http.HandlerFunc, method string) *mux.Route {
	return r.HandleFunc(path, jwtMiddleware(handlers.LogRoute(handler))).Methods(method)

}

func addRoute(r *mux.Router, path string, handler http.HandlerFunc, method string) *mux.Route {
	return r.HandleFunc(path, handlers.LogRoute(handler)).Methods(method)
}

func main() {
	// Set up logging based on environment
	if os.Getenv("ZETTEL_DEV") != "true" {
		file, err := handlers.OpenLogFile(os.Getenv("ZETTEL_BACKEND_LOG_LOCATION"))
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(file)
	}

	// Initialize shared server using bootstrap package
	s = bootstrap.InitServer()

	h = &handlers.Handler{
		Server: s,
		DB:     s.DB,
	}

	// Initialize Stripe
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	s.S3 = h.CreateS3Client()

	s.Mail = &mail.MailClient{
		Host:     os.Getenv("MAIL_HOST"),
		Password: os.Getenv("MAIL_PASSWORD"),
		Queue:    mail.NewEmailQueue(),
		DB:       s.DB,
	}

	typesenseClient, err := bootstrap.InitTypesense()
	if err == nil {
		s.TypesenseClient = typesenseClient
		go func() {
			log.Printf("updating typesense")
			h.InitSearchCollection()
		}()
	}
	log.Printf("email server: %v", s.Mail)
	s.JwtSecretKey = []byte(os.Getenv("SECRET_KEY"))
	config := openai.DefaultConfig(os.Getenv("ZETTEL_LLM_KEY"))
	config.BaseURL = os.Getenv("ZETTEL_LLM_ENDPOINT")

	if os.Getenv("ZETTEL_RUN_CHUNKING_EMBEDDING") == "true" {
		go func() {
			start := time.Now()
			//			migrations.RunEmbeddings(h)
			elapsed := time.Since(start)
			fmt.Printf("Operation took %v\n", elapsed)
		}()

	}

	r := mux.NewRouter()
	addProtectedRoute(r, "/api/auth", h.CheckTokenRoute, "GET")
	addRoute(r, "/api/auth/github", h.StartGitHubOAuthRoute, "GET")
	addRoute(r, "/api/auth/github/callback", h.GitHubCallbackRoute, "GET")
	addRoute(r, "/api/login", h.LoginRoute, "POST")
	addRoute(r, "/api/reset-password", h.ResetPasswordRoute, "POST")
	addRoute(r, "/api/email-validate", h.ValidateEmailRoute, "POST")
	addRoute(r, "/api/request-reset", h.RequestPasswordResetRoute, "POST")

	addProtectedRoute(r, "/api/files", h.GetAllFilesRoute, "GET")
	addProtectedRoute(r, "/api/files/upload", h.UploadFileRoute, "POST")
	addProtectedRoute(r, "/api/files/{id}", h.GetFileMetadataRoute, "GET")
	addProtectedRoute(r, "/api/files/{id}", h.EditFileMetadataRoute, "PATCH")
	addProtectedRoute(r, "/api/files/{id}", h.DeleteFileRoute, "DELETE")
	addProtectedRoute(r, "/api/files/download/{id}", h.DownloadFileRoute, "GET")

	addProtectedRoute(r, "/api/cards", h.CreateCardRoute, "POST")
	addProtectedRoute(r, "/api/cards/next-root-id", h.GetNextRootCardIDRoute, "GET")
	addProtectedRoute(r, "/api/cards/suggest-title", h.SuggestCardTitleRoute, "POST")
	addProtectedRoute(r, "/api/cards/starred", h.GetStarredCardsRoute, "GET")
	addProtectedRoute(r, "/api/cards/unsorted", h.GetUnsortedCardsRoute, "GET")
	addProtectedRoute(r, "/api/cards/{id}", h.GetCardRoute, "GET")
	addProtectedRoute(r, "/api/cards/{id}", h.UpdateCardRoute, "PUT")
	addProtectedRoute(r, "/api/cards/{id}", h.DeleteCardRoute, "DELETE")
	addProtectedRoute(r, "/api/cards/{id}/audit", h.GetCardAuditEventsRoute, "GET")
	addProtectedRoute(r, "/api/cards/{id}/star", h.StarCardRoute, "POST")
	addProtectedRoute(r, "/api/cards/{id}/star", h.UnstarCardRoute, "DELETE")
	addProtectedRoute(r, "/api/cards/{id}/facts", h.GetCardFacts, "GET")
	addProtectedRoute(r, "/api/cards/{id}/references", h.GetCardReferencesRoute, "GET")
	addProtectedRoute(r, "/api/cards/{id}/children", h.GetCardChildrenRoute, "GET")
	addProtectedRoute(r, "/api/cards/{id}/files", h.GetCardFilesRoute, "GET")
	addProtectedRoute(r, "/api/cards/{id}/tags", h.GetCardTagsRoute, "GET")
	addProtectedRoute(r, "/api/cards/{id}/tasks", h.GetCardTasksRoute, "GET")
	addProtectedRoute(r, "/api/cards/{id}/entities", h.GetCardEntitiesRoute, "GET")
	addProtectedRoute(r, "/api/cards/{card_pk:[0-9]+}/linked-entities", h.GetEntityByLinkedCardPKRoute, "GET")
	addProtectedRoute(r, "/api/cards/{card_pk:[0-9]+}/analysis", h.GetCardAnalysisRoute, "GET")

	addProtectedRoute(r, "/api/templates", h.GetTemplatesRoute, "GET")
	addProtectedRoute(r, "/api/templates", h.CreateTemplateRoute, "POST")
	addProtectedRoute(r, "/api/templates/{id}", h.GetTemplateRoute, "GET")
	addProtectedRoute(r, "/api/templates/{id}", h.UpdateTemplateRoute, "PUT")
	addProtectedRoute(r, "/api/templates/{id}", h.DeleteTemplateRoute, "DELETE")

	addProtectedRoute(r, "/api/search", h.SearchRoute, "POST")

	addProtectedRoute(r, "/api/users/{id}", h.GetUserRoute, "GET")
	addProtectedRoute(r, "/api/users/{id}", h.UpdateUserRoute, "PUT")
	addProtectedRoute(r, "/api/users", h.GetUsersRoute, "GET")
	addRoute(r, "/api/users", h.CreateUserRoute, "POST")
	addProtectedRoute(r, "/api/users/{id}/subscription", h.GetUserSubscriptionRoute, "GET")
	addProtectedRoute(r, "/api/billing/subscribe", h.CreateSubscriptionRoute, "POST")
	addProtectedRoute(r, "/api/billing/portal", h.BillingPortalRoute, "GET")
	addProtectedRoute(r, "/api/billing/public-key", h.StripePublicKeyRoute, "GET")
	addRoute(r, "/api/stripe/webhook", h.StripeWebhookRoute, "POST")

	addProtectedRoute(r, "/api/user/memory", h.GetUserMemoryRoute, "GET")
	addProtectedRoute(r, "/api/user/memory", h.UpdateUserMemoryRoute, "PUT")
	addProtectedRoute(r, "/api/current", h.GetCurrentUserRoute, "GET")
	addProtectedRoute(r, "/api/admin", h.GetUserAdminRoute, "GET")

	addProtectedRoute(r, "/api/tasks/{id}", h.GetTaskRoute, "GET")
	addProtectedRoute(r, "/api/tasks", h.GetTasksRoute, "GET")
	addProtectedRoute(r, "/api/tasks", h.CreateTaskRoute, "POST")
	addProtectedRoute(r, "/api/tasks/{id}", h.UpdateTaskRoute, "PUT")
	addProtectedRoute(r, "/api/tasks/{id}", h.DeleteTaskRoute, "DELETE")
	addProtectedRoute(r, "/api/tasks/{id}/audit", h.GetTaskAuditEventsRoute, "GET")

	addProtectedRoute(r, "/api/tags", h.GetTagsRoute, "GET")
	addProtectedRoute(r, "/api/tags", h.CreateTagRoute, "POST")
	addProtectedRoute(r, "/api/tags/id/{id}", h.DeleteTagRoute, "DELETE")

	addProtectedRoute(r, "/api/url/parse", h.ParseURLRoute, "POST")

	addRoute(r, "/api/mailing-list", h.AddToMailingListRoute, "POST")
	addProtectedRoute(r, "/api/mailing-list", h.GetMailingListSubscribersRoute, "GET")
	addProtectedRoute(r, "/api/mailing-list/messages", h.GetMailingListMessagesRoute, "GET")
	addProtectedRoute(r, "/api/mailing-list/messages/send", h.SendMailingListMessageRoute, "POST")
	addProtectedRoute(r, "/api/mailing-list/messages/recipients", h.GetMessageRecipientsRoute, "GET")
	addProtectedRoute(r, "/api/mailing-list/unsubscribe", h.UnsubscribeMailingListRoute, "POST")

	addProtectedRoute(r, "/api/entities", h.GetEntitiesRoute, "GET")
	addProtectedRoute(r, "/api/entities/id/{id}", h.GetEntityByIDRoute, "GET")
	addProtectedRoute(r, "/api/entities/name/{name}", h.GetEntityByNameRoute, "GET")
	addProtectedRoute(r, "/api/entities/merge", h.MergeEntitiesRoute, "POST")
	addProtectedRoute(r, "/api/entities/id/{id}", h.DeleteEntityRoute, "DELETE")
	addProtectedRoute(r, "/api/entities/id/{id}", h.UpdateEntityRoute, "PUT")
	addProtectedRoute(r, "/api/entities/{entityId}/cards/{cardId}", h.AddEntityToCardRoute, "POST")
	addProtectedRoute(r, "/api/entities/{entityId}/cards/{cardId}", h.RemoveEntityFromCardRoute, "DELETE")
	addProtectedRoute(r, "/api/entities/{id}/facts", h.GetEntityFacts, "GET")
	addProtectedRoute(r, "/api/entities/{id}/similar", h.GetSimilarEntitiesRoute, "GET")
	addProtectedRoute(r, "/api/facts/{id}/entities", h.GetFactEntities, "GET")
	addProtectedRoute(r, "/api/facts", h.GetAllFacts, "GET")
	addProtectedRoute(r, "/api/facts/{id}", h.GetFact, "GET")
	addProtectedRoute(r, "/api/facts/{id}", h.UpdateFact, "PUT")
	addProtectedRoute(r, "/api/facts/{factID}/cards/{cardID}", h.LinkFactToCardHandler, "POST")
	addProtectedRoute(r, "/api/facts/merge", h.MergeFactsRoute, "POST")
	addProtectedRoute(r, "/api/facts/{id}/cards", h.GetFactCards, "GET")
	addProtectedRoute(r, "/api/facts/{id}/similar", h.GetSimilarFacts, "GET")
	addProtectedRoute(r, "/api/facts/{id}", h.DeleteFactRoute, "DELETE")

	// Summarize routes
	addProtectedRoute(r, "/api/summarize", h.CreateSummarizationRoute, "POST")
	addProtectedRoute(r, "/api/summarize/{id}", h.GetSummarizationRoute, "GET")
	addProtectedRoute(r, "/api/summarizations", h.ListSummarizationsRoute, "GET")
	addProtectedRoute(r, "/api/cards/{card_pk:[0-9]+}/summaries", h.GetSummariesByCardRoute, "GET")

	// Pinned searches routes
	addProtectedRoute(r, "/api/searches/star", h.StarSearchRoute, "POST")
	addProtectedRoute(r, "/api/searches/star/{id}", h.UnstarSearchRoute, "DELETE")
	addProtectedRoute(r, "/api/searches/starred", h.GetStarredSearchesRoute, "GET")
	// Chat routes
	addProtectedRoute(r, "/api/chat/conversations", h.CreateConversationRoute, "POST")
	addProtectedRoute(r, "/api/chat/conversations", h.GetConversationsRoute, "GET")
	addProtectedRoute(r, "/api/chat/conversations/{id}", h.GetConversationRoute, "GET")
	addProtectedRoute(r, "/api/chat/conversations/{id}/messages", h.SendMessageRoute, "POST")
	addProtectedRoute(r, "/api/chat/conversations/{id}/messages/stream", h.StreamMessageRoute, "POST")
	addProtectedRoute(r, "/api/chat/conversations/{id}/messages/{messageId}/regenerate", h.RegenerateMessageRoute, "POST")
	addProtectedRoute(r, "/api/chat/conversations/{id}/status", h.GetConversationStatusRoute, "GET")
	addProtectedRoute(r, "/api/chat/conversations/{id}", h.DeleteConversationRoute, "DELETE")
	addProtectedRoute(r, "/api/chat/conversations/{id}/star", h.StarConversationRoute, "POST")
	addProtectedRoute(r, "/api/chat/conversations/{id}/title", h.UpdateConversationTitleRoute, "PUT")
	addProtectedRoute(r, "/api/chat/usage", h.GetUsageQuotaRoute, "GET")
	addProtectedRoute(r, "/api/chat/instructions", h.GetInstructionsRoute, "GET")
	addProtectedRoute(r, "/api/chat/instructions", h.UpdateInstructionsRoute, "PUT")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{os.Getenv("ZETTEL_URL")},
		AllowCredentials: true,
		AllowedHeaders:   []string{"authorization", "content-type"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		// Enable Debugging for testing, consider disabling in production
		//Debug: true,
	})

	handler := c.Handler(r)

	port := os.Getenv("ZETTEL_PORT")
	if port == "" {
		port = "8080"
	}
	http.ListenAndServe(":"+port, handler)
}
