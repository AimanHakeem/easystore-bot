package tasks

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"
)

func fetchHTML(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

func extractJavaScript(htmlContent string) (string, error) {
	re := regexp.MustCompile(`const collection = ({.*})`)
	match := re.FindStringSubmatch(htmlContent)

	if len(match) < 2 {
		return "", fmt.Errorf("failed to find product endpoint")
	}

	return match[1], nil
}
func RunTasks() {
	err := LoadSites()
	if err != nil {
		fmt.Println("Error loading sites:", err)
		return
	}

	file, err := os.Open("Tasks.csv")
	if err != nil {
		fmt.Println("Error opening CSV file:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Error reading CSV file:", err)
		return
	}

	headers := records[0]
	nullableFields := map[string]bool{
		"cardno":     true,
		"expirydate": true,
		"cvv":        true,
	}

	for idx, record := range records[1:] {
		startTime := time.Now()
		task := make(map[string]string)
		for i, header := range headers {
			task[header] = record[i]
		}

		validationFailed := false
		for key, value := range task {
			if !nullableFields[key] && value == "" {
				fmt.Printf("Validation error: %s cannot be null in task: %v\n", key, task)
				validationFailed = true
				break
			}
		}
		if validationFailed {
			continue
		}

		link, err := GetSiteLink(task["site"])
		if err != nil {
			fmt.Println(err)
			continue
		}

		provincesLink := fmt.Sprintf("%s/sf/countries/MY/provinces", link)
		err = LoadProvinces(provincesLink)
		if err != nil {
			fmt.Println("Error loading provinces:", err)
			continue
		}

		provinceCode, err := GetProvinceCode(task["state"])
		if err != nil {
			fmt.Println(err)
			continue
		}

		task["state"] = provinceCode

		htmlContent, err := fetchHTML(link)
		if err != nil {
			fmt.Printf("Failed to fetch HTML content for site %s: %v\n", task["site"], err)
			continue
		}

		scriptContent, err := extractJavaScript(htmlContent)
		if err != nil {
			fmt.Printf("Failed to extract JavaScript for site %s: %v\n", task["site"], err)
			continue
		}
		fmt.Printf("Extracted Collection Data: %+v\n", scriptContent)
		duration := time.Since(startTime)
		fmt.Printf("[Task %d]Execution time: %s \n", idx+1, duration)
	}
}
