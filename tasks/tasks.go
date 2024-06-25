package tasks

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Variant struct {
	ID                int    `json:"id"`
	Title             string `json:"title"`
	SKU               string `json:"sku"`
	Available         bool   `json:"available"`
	InventoryQuantity int    `json:"inventory_quantity"`
	IsEnabled         bool   `json:"is_enabled"`
}

type Product struct {
	ID                              int       `json:"id"`
	Handle                          string    `json:"handle"`
	Name                            string    `json:"name"`
	Title                           string    `json:"title"`
	URL                             string    `json:"url"`
	Price                           float64   `json:"price"`
	Available                       bool      `json:"available"`
	SoleVariantID                   int       `json:"sole_variant_id"`
	Variants                        []Variant `json:"variants"`
	SelectedVariant                 Variant   `json:"selected_variant"`
	FirstAvailableVariant           Variant   `json:"first_available_variant"`
	SelectedOrFirstAvailableVariant Variant   `json:"selected_or_first_available_variant"`
	ImgURL                          string    `json:"img_url"`
	PublishedAt                     string    `json:"published_at"`
	CreatedAt                       string    `json:"created_at"`
}

type Collection struct {
	Products []Product `json:"products"`
}

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

func extractJavaScript(htmlContent string, isDirectLink bool) (string, error) {
	var re *regexp.Regexp
	if isDirectLink {
		re = regexp.MustCompile(`const product = ({.*})`)
	} else {
		re = regexp.MustCompile(`const collection = ({.*})`)
	}

	match := re.FindStringSubmatch(htmlContent)
	if len(match) < 2 {
		return "", fmt.Errorf("failed to find product endpoint")
	}

	return match[1], nil
}

func searchProducts(collection Collection, keywordQuery string) *Product {
	var matchedProducts []Product

	// Comma untuk split keywords ("OR")
	orConditions := strings.Split(keywordQuery, ",")

	for _, product := range collection.Products {
		//Skip soldout products
		if !product.Available {
			continue
		}

		productName := strings.ToLower(product.Name)
		productMatches := false

		// Kalau ada split keyword means dia OR
		for _, orCondition := range orConditions {
			andConditions := strings.Split(orCondition, "&")
			andMatches := true

			// Check AND condition dalam OR (,)
			for _, andCondition := range andConditions {
				keyword := strings.TrimSpace(andCondition)
				if !strings.Contains(productName, keyword) {
					andMatches = false
					break
				}
			}

			if andMatches {
				productMatches = true
				break
			}
		}

		if productMatches {
			matchedProducts = append(matchedProducts, product)
		}
	}
	if len(matchedProducts) > 0 {
		highestIDProduct := matchedProducts[0]
		for _, product := range matchedProducts {
			if product.ID > highestIDProduct.ID {
				highestIDProduct = product
			}
		}
		return &highestIDProduct
	}

	return nil
}

func findVariant(product Product, size string) (*Variant, error) {
	// Pick random avilable size.
	if size == "RA" {
		var availableVariants []Variant
		for _, variant := range product.Variants {
			if variant.Available {
				availableVariants = append(availableVariants, variant)
			}
		}
		if len(availableVariants) == 0 {
			return nil, fmt.Errorf("no available variants found for product %s", product.Name)
		}

		// Create a new random source and random generator
		randSrc := rand.NewSource(time.Now().UnixNano())
		r := rand.New(randSrc)

		// Pick a random variant
		randomIndex := r.Intn(len(availableVariants))
		return &availableVariants[randomIndex], nil
	}

	// Pick specific variant
	for _, variant := range product.Variants {
		if strings.EqualFold(variant.Title, size) {
			return &variant, nil
		}
	}
	return nil, fmt.Errorf("variant with size %s not found", size)
}

func processTask(idx int, task map[string]string, wg *sync.WaitGroup) {
	defer wg.Done()
	startTime := time.Now()

	nullableFields := map[string]bool{
		"cardno":     true,
		"expirydate": true,
		"cvv":        true,
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
		return
	}

	link, err := GetSiteLink(task["site"])
	if err != nil {
		fmt.Println(err)
		return
	}

	productlink, err := GetProductLink(task["site"])
	if err != nil {
		fmt.Println(err)
		return
	}

	provincesLink := fmt.Sprintf("%s/sf/countries/MY/provinces", link)
	err = LoadProvinces(provincesLink)
	if err != nil {
		fmt.Println("Error loading provinces:", err)
		return
	}

	provinceCode, err := GetProvinceCode(task["state"])
	if err != nil {
		fmt.Println(err)
		return
	}

	task["state"] = provinceCode

	isDirectLink := strings.HasPrefix(task["keyword"], "https")
	var htmlContent string
	var retryAttempts int
	delay, err := strconv.Atoi(task["delay"])
	if err != nil {
		fmt.Printf("Delay Error: %v", err)
	}

	for {
		if isDirectLink {
			htmlContent, err = fetchHTML(task["keyword"])
		} else {
			htmlContent, err = fetchHTML(productlink)
		}

		if err != nil {
			if strings.Contains(err.Error(), "received non-200 response") {
				fmt.Printf("[Task %d][%s]Product not loaded yet.\n", idx+1, task["site"])
				time.Sleep(time.Duration(delay) * time.Millisecond)
				continue
			} else {
				fmt.Printf("Failed to fetch HTML content for site %s: %v\n", task["site"], err)
				break
			}
		}

		scriptContent, err := extractJavaScript(htmlContent, isDirectLink)
		if err != nil {
			fmt.Printf("Failed to find JavaScript object for site %s: %v\n", task["site"], err)
			break
		}

		if isDirectLink {
			//using directlink from keyword field
			var product Product
			err = json.Unmarshal([]byte(scriptContent), &product)
			if err != nil {
				fmt.Printf("Failed to unmarshal JSON for site %s: %v\n", task["site"], err)
				break
			}
			if !product.Available {
				fmt.Printf("[Task %d][OOS][%s] %s \n", idx+1, task["site"], product.Name)
				retryAttempts++
				time.Sleep(time.Duration(delay) * time.Millisecond)
				continue
			}
			fmt.Printf("[Task %d][Product Found][%s] %s \n", idx+1, task["site"], product.Name)
			variant, err := findVariant(product, task["size"])
			if err != nil {
				fmt.Printf("[Task %d][Variant OOS][%s] %s \n", idx+1, task["site"], product.Name)
				retryAttempts++
				time.Sleep(time.Duration(delay) * time.Millisecond)
				continue
			}
			fmt.Printf("[Task %d][Variant found][%s] %s \n", idx+1, task["site"], variant.Title)
		} else {
			//using keyword matching
			var collection Collection
			err = json.Unmarshal([]byte(scriptContent), &collection)
			if err != nil {
				fmt.Printf("Failed to unmarshal JSON for site %s: %v\n", task["site"], err)
				break
			}
			keywords := task["keyword"]
			matchedProduct := searchProducts(collection, keywords)
			if !matchedProduct.Available {
				fmt.Printf("[Task %d][OOS][%s] %s \n", idx+1, task["site"], matchedProduct.Name)
				retryAttempts++
				time.Sleep(time.Duration(delay) * time.Millisecond)
				continue
			}

			if matchedProduct == nil {
				fmt.Printf("[Task %d][Product not found][%s] keywords: %v\n", idx+1, task["site"], keywords)
				retryAttempts++
				time.Sleep(time.Duration(delay) * time.Millisecond)
				continue
			} else {
				fmt.Printf("[Task %d][Product Found][%s] %s \n", idx+1, task["site"], matchedProduct.Name)

				variant, err := findVariant(*matchedProduct, task["size"])
				if err != nil {
					fmt.Printf("[Task %d][Variant OOS][%s] %s \n", idx+1, task["site"], matchedProduct.Name)
					retryAttempts++
					time.Sleep(time.Duration(delay) * time.Millisecond)
					continue
				}
				fmt.Printf("[Task %d][Variant Found][%s] %s, Variant: %s \n", idx+1, task["site"], matchedProduct.Name, variant.Title)
			}
		}

		// If successfully processed, break the retry loop
		break
	}

	duration := time.Since(startTime)
	fmt.Printf("[Task %d]Execution time: %s, Site: %s\n", idx+1, duration, task["site"])

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

	var wg sync.WaitGroup

	for idx, record := range records[1:] {
		task := make(map[string]string)
		for i, header := range headers {
			task[header] = record[i]
		}
		wg.Add(1)
		go processTask(idx, task, &wg)
	}
	wg.Wait()
}
