package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Currency struct {
	Code              string `json:"code"`
	Rate              string `json:"rate"`
	FormatPrefix      string `json:"format_prefix"`
	FormatSuffix      string `json:"format_suffix"`
	FormatDecimals    int    `json:"format_decimals"`
	ThousandSeparator string `json:"thousand_separator"`
	Name              string `json:"name"`
}

type Item struct {
	ID                           int      `json:"id"`
	ProductID                    int      `json:"product_id"`
	VariantID                    int      `json:"variant_id"`
	ProductName                  string   `json:"product_name"`
	VariantName                  string   `json:"variant_name"`
	SKU                          string   `json:"sku"`
	Barcode                      string   `json:"barcode"`
	CostPrice                    string   `json:"cost_price"`
	TotalDiscount                string   `json:"total_discount"`
	Grams                        string   `json:"grams"`
	Width                        string   `json:"width"`
	Height                       string   `json:"height"`
	Length                       string   `json:"length"`
	Quantity                     int      `json:"quantity"`
	Taxable                      bool     `json:"taxable"`
	ShippingRequired             bool     `json:"shipping_required"`
	TotalTax                     string   `json:"total_tax"`
	IsCustomPrice                bool     `json:"is_custom_price"`
	InventoryManagement          string   `json:"inventory_management"`
	BarcodeImage                 string   `json:"barcode_image"`
	Properties                   []string `json:"properties"`
	Brands                       string   `json:"brands"`
	Tags                         string   `json:"tags"`
	Handle                       string   `json:"handle"`
	OriginalLinePrice            string   `json:"original_line_price"`
	FinalLinePrice               string   `json:"final_line_price"`
	ImageURL                     string   `json:"image_url"`
	LineLevelDiscountAllocations []string `json:"line_level_discount_allocations"`
	LineLevelTotalDiscount       int      `json:"line_level_total_discount"`
	Name                         string   `json:"name"`
	URL                          string   `json:"url"`
	OriginalPrice                string   `json:"original_price"`
	Collections                  []string `json:"collections"`
	Vendor                       string   `json:"vendor"`
	IsQuantityEditable           bool     `json:"is_quantity_editable"`
}

type CartResponse struct {
	ID                 int      `json:"id"`
	Token              string   `json:"token"`
	OrderToken         string   `json:"order_token"`
	CustomerID         *int     `json:"customer_id"`
	ItemCount          int      `json:"item_count"`
	Items              []Item   `json:"items"`
	ItemsSubtotalPrice string   `json:"items_subtotal_price"`
	OriginalTotalPrice string   `json:"original_total_price"`
	TaxesIncluded      bool     `json:"taxes_included"`
	TotalDiscount      string   `json:"total_discount"`
	TotalPrice         string   `json:"total_price"`
	TotalWeight        string   `json:"total_weight"`
	Currency           Currency `json:"currency"`
}

type ShippingRateResponse struct {
	Checkout struct {
		SelectedShippingMethod struct {
			ID           int    `json:"id"`
			Handle       string `json:"handle"`
			Title        string `json:"title"`
			Price        string `json:"price"`
			TotalPrice   string `json:"total_price"`
			ImageURL     string `json:"image_url"`
			FreeMinPrice string `json:"free_min_price"`
			CodMinAmount string `json:"cod_min_amount"`
			CodMaxAmount string `json:"cod_max_amount"`
			CodType      int    `json:"cod_type"`
		} `json:"selected_shipping_method"`
	} `json:"checkout"`
}

func addToCart(link string, variantID int, quantity string, xsrfToken string, client *http.Client, idx int) (string, error) {
	url := fmt.Sprintf("%v/cart/add?retrieve=true", link)
	payload := map[string]interface{}{
		"id":       variantID,
		"quantity": quantity,
		"_token":   xsrfToken,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-XSRF-TOKEN", xsrfToken)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("[Task %d]failed to send POST request: %w", idx+1, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return "", fmt.Errorf("[Task %d] received non-200 response: %d, response body: %s", idx+1, resp.StatusCode, bodyString)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("[Task %d]failed to read response body: %w", idx+1, err)
	}

	var cartResponse CartResponse
	if err := json.Unmarshal(bodyBytes, &cartResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	for idx, item := range cartResponse.Items {
		fmt.Printf("[Task %d][ATC] Carted: %s | Quantity: %d\n", idx+1, item.ProductName, item.Quantity)
	}

	return cartResponse.Token, nil
}

func getShippingRate(idx int, link string, client *http.Client, cartToken string, addressLine1 string, postcode string, city string, provinceCode string, xsrfToken string) (string, error) {
	entrypoint := fmt.Sprintf("%v/sf/checkout/%v/shipping_address", link, cartToken)

	form := url.Values{}
	form.Add("_token", xsrfToken)
	form.Add("_testing", strconv.FormatBool(false))
	form.Add("checkout[detail][first_name]", "")
	form.Add("checkout[detail][last_name]", "")
	form.Add("checkout[detail][email]", "")
	form.Add("checkout[detail][phone]", "")
	form.Add("base_delivery_method", "shipping")
	form.Add("checkout[delivery_datetime]", "")
	form.Add("checkout[pickup_address][is_self_collect]", strconv.FormatBool(true))
	form.Add("checkout[pickup_address][receiver][first_name]", "")
	form.Add("checkout[pickup_address][receiver][last_name]", "")
	form.Add("checkout[pickup_address][receiver][email]", "")
	form.Add("checkout[pickup_address][receiver][phone]", "")
	form.Add("checkout[delivery_method]", "shipping-standard")
	form.Add("checkout[shipping_address][first_name]", "")
	form.Add("checkout[shipping_address][last_name]", "")
	form.Add("checkout[shipping_address][email]", "")
	form.Add("checkout[shipping_address][phone]", "")
	form.Add("checkout[shipping_address][company]", "")
	form.Add("checkout[shipping_address][address1]", addressLine1)
	form.Add("checkout[shipping_address][province_code]", provinceCode)
	form.Add("checkout[shipping_address][address2]", "")
	form.Add("checkout[shipping_address][country_code]", "MY")
	form.Add("checkout[shipping_address][city]", city)
	form.Add("checkout[shipping_address][zip]", postcode)
	form.Add("shipping_handle", "")
	form.Add("checkout[remark]", "")
	form.Add("checkout[billing_same_as_shipping]", "")
	form.Add("checkout[billing_address][company]", "")
	form.Add("checkout[billing_address][address1]", "")
	form.Add("checkout[billing_address][address2]", "")
	form.Add("checkout[billing_address][city]", "")
	form.Add("checkout[billing_address][zip]", "")
	form.Add("payment_category", "")
	form.Add("checkout[gateway_handle]", "")

	req, err := http.NewRequest("PUT", entrypoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create PUT request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send PUT request %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch shipping rate, status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response from shipping rate: %w", err)
	}

	var shippingRateResp ShippingRateResponse
	if err := json.Unmarshal(bodyBytes, &shippingRateResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	shippingRateHandle := shippingRateResp.Checkout.SelectedShippingMethod.Handle

	fmt.Printf("[Task %d][Shipping Rate] %v \n", idx+1, shippingRateHandle)

	return shippingRateHandle, nil

}
