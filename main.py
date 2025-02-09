import time
import re
import requests

from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service  # For setting up ChromeDriver service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from webdriver_manager.chrome import ChromeDriverManager  # Automatically manages ChromeDriver

# --- Pushover API Credentials ---
PUSHOVER_USER_KEY = "ukayvywh7zjxg5s2jfrndmqix61prs"
PUSHOVER_API_TOKEN = "ayods1kowznws72y24ta547owk18br"

def send_push_notification(title, message):
    """
    Sends a push notification using the Pushover API.
    """
    data = {
        "token": PUSHOVER_API_TOKEN,
        "user": PUSHOVER_USER_KEY,
        "title": title,
        "message": message
    }
    try:
        response = requests.post("https://api.pushover.net/1/messages.json", data=data)
        response.raise_for_status()  # Raise an HTTPError for bad responses
        print("Notification sent successfully.")
    except Exception as e:
        print("Failed to send notification:", e)

def fetch_listing_count(url):
    """
    Uses Selenium to fetch the webpage and extract the numeric listing count from the element:
    
      <div class="-listing-count">
          <div class="-header">
              <span>7 Listings</span>
          </div>
      </div>
      
    Returns the listing count as an integer or None if not found.
    """
    options = Options()
    
    # For debugging, try running without headless mode:
    # options.add_argument("--headless")
    
    options.add_argument("--disable-gpu")
    options.add_argument("--window-size=1920,1080")
    # Set a user agent to mimic a regular browser
    options.add_argument("user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36")
    
    # Set up ChromeDriver with webdriver_manager so it uses the correct driver version
    service = Service(ChromeDriverManager().install())
    driver = webdriver.Chrome(service=service, options=options)
    
    try:
        driver.get(url)
        # Increase the wait time to ensure the element loads
        wait = WebDriverWait(driver, 20)
        # Wait for the <span> inside the listing count element to be present
        element = wait.until(EC.presence_of_element_located((By.CSS_SELECTOR, "div.-listing-count span")))
        
        # Get the text directly from the element
        text = element.text
        print(f"Found text: '{text}' on {url}")
        
        # Extract the first group of digits from the text (e.g., "7" from "7 Listings")
        match = re.search(r'(\d+)', text)
        if match:
            count = int(match.group(1))
            return count
        else:
            print(f"Warning: Could not extract a number from the text '{text}' at {url}.")
            return None
    except Exception as e:
        print(f"Error using Selenium to fetch {url}: {e}")
        return None
    finally:
        driver.quit()

def check_listing_count_update(url, last_count):
    """
    Checks the listing count for the given URL.
    
    Returns a tuple (current_count, update_detected). On the first run (when last_count is None),
    the current count is stored but no notification is sent.
    """
    try:
        current_count = fetch_listing_count(url)
        if current_count is None:
            # Could not retrieve a valid count; skip update.
            return last_count, False

        if last_count is None:
            # First run: initialize with the current count.
            return current_count, False
        
        if current_count != last_count:
            return current_count, True
        else:
            return current_count, False
    except Exception as e:
        print(f"Error fetching listing count from {url}: {e}")
        return last_count, False

def main():
    # --- List of webpages to monitor ---
    urls = [
        "https://www.grailed.com/shop/nxzCtqQtfg",
        # Add more URLs as needed.
    ]

    # Dictionary to store the last known listing count for each URL.
    last_counts = {url: None for url in urls}

    print("Starting the monitoring process. Press Ctrl+C to exit.")
    while True:
        for url in urls:
            new_count, updated = check_listing_count_update(url, last_counts[url])
            if updated:
                print(f"Listing count updated for {url}: {last_counts[url]} -> {new_count}")
                send_push_notification(
                    "Listing Count Update",
                    f"Listings changed from {last_counts[url]} to {new_count} at {url}"
                )
            # Update the stored count for this URL.
            last_counts[url] = new_count
        
        # Wait for 60 seconds before checking again.
        time.sleep(60)

if __name__ == "__main__":
    main()
