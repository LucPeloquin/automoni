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

// PushoverCredentials stores API credentials for Pushover
type PushoverCredentials struct {
	UserKey   string
	APIToken  string
}

// Constants for Pushover API
const (
	pushoverUserKey = "ukayvywh7zjxg5s2jfrndmqix61prs"
	pushoverAPIToken = "ayods1kowznws72y24ta547owk18br"
	pushoverAPI = "https://api.pushover.net/1/messages.json"
)

// sendPushNotification sends a push notification using the Pushover API
func sendPushNotification(title, message string) error {
	data := http.Client{}
	
	form := strings.NewReader(fmt.Sprintf(
		"token=%s&user=%s&title=%s&message=%s",
		pushoverAPIToken,
		pushoverUserKey,
		title,
		message,
	))

	req, err := http.NewRequest("POST", pushoverAPI, form)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := data.Do(req)
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

// fetchListingCount retrieves the listing count from the specified URL
func fetchListingCount(url string) (int, error) {
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

	var statsText string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(3*time.Second),
		chromedp.Text("div.ais-Panel.-stats", &statsText, chromedp.NodeVisible, chromedp.ByQuery),
	)

	if err != nil {
		return 0, fmt.Errorf("failed to fetch page: %v", err)
	}

	if statsText == "" {
		return 0, fmt.Errorf("empty stats text received")
	}

	log.Printf("Full text from the ais-Panel -stats element: %s", statsText)

	// Try to extract number using regex
	re := regexp.MustCompile(`(\d+(?:,\d+)?)\s+(\w+)`)
	matches := re.FindStringSubmatch(statsText)

	if len(matches) < 2 {
		return 0, fmt.Errorf("could not extract number from text: '%s'", statsText)
	}

	// Remove commas and convert to integer
	numberStr := strings.ReplaceAll(matches[1], ",", "")
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return 0, fmt.Errorf("error converting '%s' to integer: %v", numberStr, err)
	}

	log.Printf("Extracted number: %d", number)
	return number, nil
}

// checkListingCountUpdate checks if the listing count has been updated
func checkListingCountUpdate(url string, lastCount *int) (int, bool, error) {
	currentCount, err := fetchListingCount(url)
	if err != nil {
		return 0, false, err
	}

	if lastCount == nil {
		return currentCount, false, nil
	}

	return currentCount, currentCount > *lastCount, nil
}

func main() {
	// List of URLs to monitor
	urls := []string{
		"https://www.grailed.com/shop/nxzCtqQtfg",
		"https://www.grailed.com/shop/lRwSEkgxZw",
		"https://www.grailed.com/shop/PweX949iwA",
	}

	// Map to store the last known listing count for each URL
	lastCounts := make(map[string]*int)
	for _, url := range urls {
		lastCounts[url] = nil
	}

	log.Println("Starting the monitoring process. Press Ctrl+C to exit.")

	for {
		for _, url := range urls {
			newCount, updated, err := checkListingCountUpdate(url, lastCounts[url])
			if err != nil {
				log.Printf("Error checking listing count for %s: %v", url, err)
				continue
			}

			if updated {
				log.Printf("Listing count updated for %s: %d -> %d", url, *lastCounts[url], newCount)
				err := sendPushNotification(
					"Listing Count Update",
					fmt.Sprintf("Listings changed from %d to %d at %s", *lastCounts[url], newCount, url),
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
