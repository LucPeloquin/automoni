package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// Constants for ntfy
const (
	ntfyTopic  = "automonitor"     // Replace with your desired topic
	ntfyServer = "https://ntfy.sh" // You can use the public server or your self-hosted instance
)

// PushoverCredentials stores API credentials for Pushover
type PushoverCredentials struct {
	UserKey  string
	APIToken string
}

// Response represents the API response structure
type Response struct {
	Status    string   `json:"status"`
	Message   string   `json:"message"`
	Timestamp string   `json:"timestamp"`
	Updates   []Update `json:"updates,omitempty"`
}

// Update represents a change in listings
type Update struct {
	URL           string `json:"url"`
	SearchTerm    string `json:"searchTerm"`
	CurrentCount  int    `json:"currentCount"`
	PreviousCount int    `json:"previousCount,omitempty"`
	Changed       bool   `json:"changed"`
}

// sendPushNotification sends a push notification using ntfy
func sendPushNotification(title, message string) error {
	url := fmt.Sprintf("%s/%s", ntfyServer, ntfyTopic)

	req, err := http.NewRequest("POST", url, strings.NewReader(message))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Title", title)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("notification failed with status: %d", resp.StatusCode)
	}

	log.Println("Notification sent successfully")
	return nil
}

// fetchListingCount retrieves the listing count and search term from the specified URL
func fetchListingCount(url string) (int, string, error) {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Configure Chrome
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("start-maximized", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("remote-debugging-port", "9222"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// Create context
	ctx, cancel = chromedp.NewContext(allocCtx)
	defer cancel()

	var statsText, searchTerm, refinements string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(3*time.Second),
		chromedp.Text("div.ais-Panel.-stats", &statsText, chromedp.NodeVisible, chromedp.ByQuery),
		chromedp.Text(`h1[data-testid="Title"]`, &searchTerm, chromedp.NodeVisible, chromedp.ByQuery),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('ul.-current-refinements span.-refinement-label')).map(el => el.textContent).join(', ')`, &refinements),
	)

	if err != nil {
		return 0, "", fmt.Errorf("failed to fetch page: %v", err)
	}

	if statsText == "" {
		return 0, "", fmt.Errorf("empty stats text received")
	}

	// log.Printf("Full text from the ais-Panel -stats element: %s", statsText)
	// log.Printf("Extracted search term: %s, refinements: %s", searchTerm, refinements)

	// Try to extract number using regex
	re := regexp.MustCompile(`(\d+(?:,\d+)?)\s+(\w+)`)
	matches := re.FindStringSubmatch(statsText)

	if len(matches) < 2 {
		return 0, "", fmt.Errorf("could not extract number from text: '%s'", statsText)
	}

	// Remove commas and convert to integer
	numberStr := strings.ReplaceAll(matches[1], ",", "")
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return 0, "", fmt.Errorf("error converting '%s' to integer: %v", numberStr, err)
	}

	log.Printf("Listing cnt: %d, search term: %s, refinements: %s", number, searchTerm, refinements)
	fullSearchTerm := fmt.Sprintf("%s (%s)", searchTerm, refinements)
	return number, fullSearchTerm, nil
}

// checkListingCountUpdate checks if the listing count has been updated
func checkListingCountUpdate(url string, lastCount *int) (int, string, bool, error) {
	currentCount, fullSearchTerm, err := fetchListingCount(url)
	if err != nil {
		return *lastCount, "", false, err // Return existing count on error
	}

	if lastCount == nil {
		return currentCount, fullSearchTerm, false, nil
	}

	return currentCount, fullSearchTerm, currentCount > *lastCount, nil
}

// Handler handles HTTP requests for Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		sendJSONResponse(w, http.StatusMethodNotAllowed, Response{
			Status:    "error",
			Message:   "Method not allowed",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// Check for authorization header
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != os.Getenv("API_KEY") {
		sendJSONResponse(w, http.StatusUnauthorized, Response{
			Status:    "error",
			Message:   "Unauthorized",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	updates, err := checkAllListings()
	if err != nil {
		sendJSONResponse(w, http.StatusInternalServerError, Response{
			Status:    "error",
			Message:   err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	sendJSONResponse(w, http.StatusOK, Response{
		Status:    "success",
		Message:   "Listings checked successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Updates:   updates,
	})
}

// checkAllListings checks all URLs and returns updates
func checkAllListings() ([]Update, error) {
	urls := []string{
		"https://www.grailed.com/shop/nxzCtqQtfg",
		"https://www.grailed.com/shop/lRwSEkgxZw",
		"https://www.grailed.com/shop/PweX949iwA",
		"https://www.grailed.com/shop/z5RvSYTnZQ",
	}

	var updates []Update
	for _, url := range urls {
		currentCount, searchTerm, err := fetchListingCount(url)
		if err != nil {
			log.Printf("Error checking %s: %v", url, err)
			continue
		}

		// Get previous count from database (implementation needed)
		previousCount, err := getPreviousCount(url)
		if err != nil {
			log.Printf("Error getting previous count for %s: %v", url, err)
		}

		update := Update{
			URL:          url,
			SearchTerm:   searchTerm,
			CurrentCount: currentCount,
			Changed:      previousCount != nil && currentCount != *previousCount,
		}

		if previousCount != nil {
			update.PreviousCount = *previousCount
		}

		// Store new count in database (implementation needed)
		if err := storeCount(url, currentCount); err != nil {
			log.Printf("Error storing count for %s: %v", url, err)
		}

		// Send notification if count changed
		if update.Changed {
			err := sendPushNotification(
				"Listing Count Update",
				fmt.Sprintf("Listings changed from %d to %d at %s (%s)",
					update.PreviousCount, update.CurrentCount, url, searchTerm),
			)
			if err != nil {
				log.Printf("Failed to send notification: %v", err)
			}
		}

		updates = append(updates, update)
	}

	return updates, nil
}

// sendJSONResponse sends a JSON response with the given status code
func sendJSONResponse(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// Database interface functions (to be implemented)
func getPreviousCount(url string) (*int, error) {
	// TODO: Implement database retrieval
	return nil, nil
}

func storeCount(url string, count int) error {
	// TODO: Implement database storage
	return nil
}

func main() {
	// Set up HTTP handler
	http.HandleFunc("/api/check", Handler)

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start the server
	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
