package handlers

import (
	"encoding/json"
	"fmt"
	htmlstd "html"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// DiscoverFeedRequest represents the request payload for feed discovery
type DiscoverFeedRequest struct {
	URL string `json:"url"`
}

// DiscoverFeedResponse represents the response payload for feed discovery
type DiscoverFeedResponse struct {
	FeedURL string `json:"feed_url"`
	Title   string `json:"title"`
}

const (
	// DiscoveryTimeout is the HTTP timeout for feed discovery requests
	DiscoveryTimeout = 10 * time.Second
	// MaxRedirects is the maximum number of redirects to follow
	MaxRedirects = 3
)

// DiscoverFeedRoute handles POST /api/rss/discover
// It fetches the HTML from the given URL and attempts to discover RSS/Atom feed links
func (h *Handler) DiscoverFeedRoute(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req DiscoverFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate URL format
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.Parse(req.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		log.Printf("Invalid URL format: %s", req.URL)
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	// Ensure scheme is http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	// Discover feed
	response, err := discoverFeed(req.URL)
	if err != nil {
		log.Printf("Failed to discover feed: %v", err)

		// Check for timeout errors
		if err, ok := err.(interface{ Timeout() bool }); ok && err.Timeout() {
			http.Error(w, "Request timed out. Please try again.", http.StatusGatewayTimeout)
			return
		}

		// Check for no feed found (case-insensitive matching)
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "rss/atom feed found") || strings.Contains(errMsg, "feed found") || strings.Contains(errMsg, "non-html response") || strings.Contains(errMsg, "not a webpage") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// discoverFeed attempts to discover an RSS/Atom feed from the given URL
func discoverFeed(targetURL string) (*DiscoverFeedResponse, error) {
	client := &http.Client{
		Timeout: DiscoveryTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Follow up to MaxRedirects
			if len(via) >= MaxRedirects {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// First, check if the URL itself is a feed
	resp, err := client.Get(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// If we get an XML response, it might be a feed already
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/xml") ||
		strings.Contains(contentType, "application/rss+xml") ||
		strings.Contains(contentType, "application/atom+xml") ||
		strings.Contains(contentType, "text/xml") {
		// Try to parse as feed
		parsedURL, _ := url.Parse(targetURL)
		return &DiscoverFeedResponse{
			FeedURL: targetURL,
			Title:   parsedURL.Host,
		}, nil
	}

	// Check if we got HTML
	if !strings.Contains(contentType, "text/html") {
		return nil, fmt.Errorf("no RSS/Atom feed found. The URL may not be a webpage")
	}

	// Parse HTML to find feed links
	feedURL, title, err := findFeedInHTML(resp.Body, targetURL)
	if err != nil {
		return nil, err
	}

	return &DiscoverFeedResponse{
		FeedURL: feedURL,
		Title:   title,
	}, nil
}

// findFeedInHTML parses HTML content to find RSS/Atom feed links
func findFeedInHTML(body io.Reader, baseURL string) (string, string, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Look for feed links in the <head>
	var feedLinks []struct {
		href    string
		feedURL string
		title   string
		isRSS   bool
	}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Look for <link> tags with RSS/Atom types
			if n.Data == "link" {
				var rel, href, linkType, title string
				for _, attr := range n.Attr {
					switch attr.Key {
					case "rel":
						rel = attr.Val
					case "href":
						href = attr.Val
					case "type":
						linkType = attr.Val
					case "title":
						title = attr.Val
					}
				}

				// Check if this is an alternate link with RSS or Atom type
				if (rel == "alternate" || rel == "") && href != "" {
					isRSS := strings.Contains(linkType, "rss") || strings.Contains(linkType, "application/rss+xml")
					isAtom := strings.Contains(linkType, "atom") || strings.Contains(linkType, "application/atom+xml")

					if isRSS || isAtom {
						feedLinks = append(feedLinks, struct {
							href    string
							feedURL string
							title   string
							isRSS   bool
						}{
							href:    href,
							title:   title,
							isRSS:   isRSS,
							feedURL: "", // Will be resolved below
						})
					}
				}
			}

			// Extract page title from <title> tag
			if n.Data == "title" && n.FirstChild != nil {
				if n.Parent != nil && n.Parent.Data == "head" {
					// This is the page title
					parsedBaseURL.Fragment = ""
					parsedBaseURL.RawQuery = ""
				}
			}
		}

		// Recursively traverse child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	// Extract page title separately
	pageTitle := extractPageTitle(doc, parsedBaseURL.Host)

	// Prefer RSS feeds, fallback to Atom
	var chosenFeed *struct {
		href    string
		feedURL string
		title   string
		isRSS   bool
	}
	for _, feed := range feedLinks {
		if feed.isRSS {
			chosenFeed = &feed
			break
		}
	}
	if chosenFeed == nil && len(feedLinks) > 0 {
		chosenFeed = &feedLinks[0]
	}

	if chosenFeed != nil {
		// Resolve relative URL to absolute URL
		feedURL, err := resolveURL(parsedBaseURL, chosenFeed.href)
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve feed URL: %w", err)
		}

		// Use feed title if available, otherwise use page title
		title := chosenFeed.title
		if title == "" {
			title = pageTitle
		}

		return feedURL, htmlstd.UnescapeString(title), nil
	}

	// No feed found in headers, try common paths
	commonPaths := []string{"/feed", "/rss", "/atom.xml", "/feed.xml", "/rss.xml"}
	for _, path := range commonPaths {
		feedURL := *parsedBaseURL
		feedURL.Path = path
		feedURL.Fragment = ""
		feedURL.RawQuery = ""

		// Try to fetch the potential feed
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(feedURL.String())
		if err == nil {
			defer resp.Body.Close()
			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(contentType, "application/xml") ||
				strings.Contains(contentType, "application/rss+xml") ||
				strings.Contains(contentType, "application/atom+xml") ||
				strings.Contains(contentType, "text/xml") {
				return feedURL.String(), pageTitle, nil
			}
		}
	}

	return "", "", fmt.Errorf("No RSS/Atom feed found. Try checking /feed, /rss, or /atom")
}

// extractPageTitle extracts the page title from an HTML document
func extractPageTitle(doc *html.Node, fallback string) string {
	var findTitle func(*html.Node) string
	findTitle = func(n *html.Node) string {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				return htmlstd.UnescapeString(strings.TrimSpace(n.FirstChild.Data))
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if title := findTitle(c); title != "" {
				return title
			}
		}
		return ""
	}

	title := findTitle(doc)
	if title != "" {
		return title
	}
	return fallback
}

// resolveURL resolves a potentially relative URL against a base URL
func resolveURL(base *url.URL, href string) (string, error) {
	if href == "" {
		return "", fmt.Errorf("empty href")
	}

	// If already absolute, return as-is
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href, nil
	}

	// Resolve relative URL
	parsedHref, err := url.Parse(href)
	if err != nil {
		return "", fmt.Errorf("failed to parse href: %w", err)
	}

	resolved := base.ResolveReference(parsedHref)
	return resolved.String(), nil
}
