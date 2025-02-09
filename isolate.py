import re
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

# Hard-code the path to your chromedriver.exe (adjust the path if needed)
chromedriver_path = "chromedriver.exe"

# Setup Chrome options
options = Options()
options.add_argument("--disable-gpu")
options.add_argument("--window-size=1920,1080")

# Initialize the Service using the hard-coded chromedriver.exe path
service = Service(executable_path=chromedriver_path)

# Initialize the Chrome driver with the service and options
driver = webdriver.Chrome(service=service, options=options)

try:
    # Replace with the URL that contains the element with the classes "ais-Panel" and "-stats"
    url = "https://www.grailed.com/shop/nxzCtqQtfg"
    driver.get(url)
    

    # Wait for the element with both classes "ais-Panel" and "-stats" to be present.
    # Using a CSS selector that targets a <div> with both classes.
    wait = WebDriverWait(driver, 20)
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
        print("Extracted label:", label)
    else:
        print("Could not parse the text using the regex.")
        
except Exception as e:
    print("Error using Selenium:", e)
    
finally:
    driver.quit()
