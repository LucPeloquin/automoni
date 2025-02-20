package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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
	// Create context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Add timeout to context
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
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

func main() {
	// List of URLs to monitor
	urls := []string{
		"https://www.grailed.com/shop/nxzCtqQtfg",
		"https://www.grailed.com/shop/lRwSEkgxZw",
		"https://www.grailed.com/shop/PweX949iwA",
		"https://www.grailed.com/shop/z5RvSYTnZQ",
	}

	// Map to store the last known listing count for each URL
	lastCounts := make(map[string]*int)
	for _, url := range urls {
		lastCounts[url] = nil
	}

	log.Println("Starting the monitoring process. Press Ctrl+C to exit.")

	for {
		for _, url := range urls {
			newCount, fullSearchTerm, updated, err := checkListingCountUpdate(url, lastCounts[url])
			if err != nil {
				log.Printf("Error checking listing count for %s: %v", url, err)
				continue
			}

			if updated {
				log.Printf("Listing count updated for %s (%s): %d -> %d", url, fullSearchTerm, *lastCounts[url], newCount)
				err := sendPushNotification(
					"Listing Count Update",
					fmt.Sprintf("Listings changed from %d to %d at %s (%s)", *lastCounts[url], newCount, url, fullSearchTerm),
				)
				if err != nil {
					log.Printf("Failed to send notification: %v", err)
				}
			}

			// Update the stored count
			count := newCount // Create a new variable to store the address
			lastCounts[url] = &count
		}

		// Wait before next check
		time.Sleep(600 * time.Second)
	}
}
