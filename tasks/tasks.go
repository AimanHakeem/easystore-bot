package tasks

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
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

func fetchHTML(url string, client *http.Client) (string, *http.Response, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch URL: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return "", resp, fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", resp, fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), resp, nil
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

	orConditions := strings.Split(keywordQuery, ",")

	for _, product := range collection.Products {

		productName := strings.ToLower(product.Name)
		productMatches := false

		for _, orCondition := range orConditions {
			andConditions := strings.Split(orCondition, "&")
			andMatches := true

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

		randSrc := rand.NewSource(time.Now().UnixNano())
		r := rand.New(randSrc)

		randomIndex := r.Intn(len(availableVariants))
		selectedVariant := &availableVariants[randomIndex]
		if selectedVariant.Available {
			return selectedVariant, nil
		}
	}

	for _, variant := range product.Variants {
		if strings.EqualFold(variant.Title, size) && variant.Available {
			return &variant, nil
		}
	}
	return nil, fmt.Errorf("variant with size %s not found", size)
}

func handleDirectLink(task map[string]string, scriptContent string, idx int) (*Variant, error) {
	var product Product
	err := json.Unmarshal([]byte(scriptContent), &product)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON for site %s: %w", task["site"], err)
	}
	if !product.Available {
		fmt.Printf("[Task %d][OOS][%s] %s \n", idx+1, task["site"], product.Name)
		return nil, nil
	}
	fmt.Printf("[Task %d][Product Found][%s] %s \n", idx+1, task["site"], product.Name)
	variant, err := findVariant(product, task["size"])
	if err != nil {
		fmt.Printf("[Task %d][Variant OOS][%s] %s \n", idx+1, task["site"], product.Name)
		return nil, nil
	}
	fmt.Printf("[Task %d][Variant found][%s] %s \n", idx+1, task["site"], variant.Title)
	return variant, nil
}

func handleKeywordMatching(task map[string]string, scriptContent string, idx int) (*Variant, error) {
	var collection Collection
	err := json.Unmarshal([]byte(scriptContent), &collection)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON for site %s: %w", task["site"], err)
	}

	keywords := task["keyword"]
	matchedProduct := searchProducts(collection, keywords)
	if matchedProduct == nil {
		fmt.Printf("[Task %d][%s] No product matched / Product not loaded\n", idx+1, task["site"])
		return nil, nil
	}
	if !matchedProduct.Available {
		fmt.Printf("[Task %d][OOS][%s] %s \n", idx+1, task["site"], matchedProduct.Name)
		return nil, nil
	}

	fmt.Printf("[Task %d][Product Found][%s] %s \n", idx+1, task["site"], matchedProduct.Name)

	variant, err := findVariant(*matchedProduct, task["size"])
	if err != nil {
		fmt.Printf("[Task %d][Variant OOS][%s] %s \n", idx+1, task["site"], matchedProduct.Name)
		return nil, nil
	}
	fmt.Printf("[Task %d][Variant Found][%s] %s, Variant: %s \n", idx+1, task["site"], matchedProduct.Name, variant.Title)
	return variant, nil
}

func extractXsrfToken(resp *http.Response) (string, error) {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "XSRF-TOKEN" {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("XSRF-TOKEN not found in cookies")
}

func addToCart(link string, variantID int, quantity string, xsrfToken string, client *http.Client) error {
	url := fmt.Sprintf("%v/cart/add?retrieve=true", link)
	payload := map[string]interface{}{
		"id":       variantID,
		"quantity": quantity,
		"_token":   xsrfToken,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-XSRF-TOKEN", xsrfToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}

	fmt.Printf("Variant %d added to cart successfully.\n", variantID)
	return nil
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
	var resp *http.Response
	var retryAttempts int
	delay, err := strconv.Atoi(task["delay"])
	if err != nil {
		fmt.Printf("Delay Error: %v", err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Printf("Failed to create cookie jar: %v\n", err)
		return
	}
	client := &http.Client{
		Jar: jar,
	}

	for {
		if isDirectLink {
			htmlContent, resp, err = fetchHTML(task["keyword"], client)
		} else {
			htmlContent, resp, err = fetchHTML(productlink, client)
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

		xsrfToken, err := extractXsrfToken(resp)
		if err != nil {
			fmt.Printf("Failed to extract XSRF token for site %s: %v\n", task["site"], err)
			break
		}

		scriptContent, err := extractJavaScript(htmlContent, isDirectLink)
		if err != nil {
			fmt.Printf("Failed to find JavaScript object for site %s: %v\n", task["site"], err)
			break
		}

		var variant *Variant
		if isDirectLink {
			variant, err = handleDirectLink(task, scriptContent, idx)
		} else {
			variant, err = handleKeywordMatching(task, scriptContent, idx)
		}

		if err != nil {
			fmt.Println(err)
			break
		}

		if variant != nil {
			err = addToCart(link, variant.ID, task["quantity"], xsrfToken, client)
			if err != nil {
				fmt.Printf("Failed to add variant to cart for site %s: %v\n", task["site"], err)
			} else {
				break
			}
		}

		retryAttempts++
		time.Sleep(time.Duration(delay) * time.Millisecond)
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
