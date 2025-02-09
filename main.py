import time
import re
import requests

from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service  # For setting up ChromeDriver service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

# --- Pushover API Credentials ---
PUSHOVER_USER_KEY = "ukayvywh7zjxg5s2jfrndmqix61prs"
PUSHOVER_API_TOKEN = "ayods1kowznws72y24ta547owk18br"

def send_push_notification(title, message):

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

    options = Options()
    # Run headless (uncomment the next line to run headless)
    # options.add_argument("--headless")
    options.add_argument("--disable-gpu")
    options.add_argument("--window-size=1920,1080")
    
    # Hard-code the path to your chromedriver.exe (adjust if needed)
    chromedriver_path = "chromedriver.exe"
    
    # Initialize the Service using the hard-coded chromedriver.exe path
    service = Service(executable_path=chromedriver_path)
    
    # Initialize the Chrome driver with the service and options
    driver = webdriver.Chrome(service=service, options=options)
    
    try:
        driver.get(url)
        # Wait up to 20 seconds for the element to be present
        wait = WebDriverWait(driver, 20)
        time.sleep(3)
        stats_element = wait.until(
            EC.presence_of_element_located((By.CSS_SELECTOR, "div.ais-Panel.-stats"))
        )
        
        # Get the full text from the element (e.g., "123 results found")
        full_text = stats_element.text
        print("Full text from the ais-Panel -stats element:", full_text)
        
        # Extract the number and label using regex.
        # For example, if the text is "123 results", this will extract 123 and "results".
        match = re.match(r"(\d+)\s+(\w+)", full_text)
        if match:
            number = int(match.group(1))
            label = match.group(2)
            print("Extracted number:", number)
            # print("Extracted label:", label)
            return number
        else:
        # Attempt to match a number with commas (e.g., "1,652,234 listings")
            match = re.match(r"([\d,]+)\s+(\w+)", full_text)
            number_str = match.group(1).replace(',', '')
            number = int(number_str)
            label = match.group(2)
            print("Extracted number (with commas removed):", number)
            # print("Extracted label:", label)
            return number

    except Exception as e:
        print("Error using Selenium:", e)
            
    finally:
        driver.quit()

def check_listing_count_update(url, last_count):

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
        "https://www.grailed.com/shop/lRwSEkgxZw",
        "https://www.grailed.com/shop/PweX949iwA",
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
        time.sleep(600)

if __name__ == "__main__":
    main()
