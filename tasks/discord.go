package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}
type Thumbnail struct {
	Url string `json:"url"`
}
type Footer struct {
	Text     string `json:"text"`
	Icon_url string `json:"icon_url"`
}
type Embed struct {
	Title       string    `json:"title"`
	Url         string    `json:"url"`
	Description string    `json:"description"`
	Color       int       `json:"color"`
	Thumbnail   Thumbnail `json:"thumbnail"`
	Footer      Footer    `json:"footer"`
	Fields      []Field   `json:"fields"`
	Timestamp   time.Time `json:"timestamp"`
	Author      Author    `json:"author"`
}

type Author struct {
	Name     string `json:"name"`
	Icon_URL string `json:"icon_url"`
	Url      string `json:"url"`
}

type Attachment struct {
	Id          string `json:"id"`
	Description string `json:"description"`
	Filename    string `json:"filename"`
}
type Hook struct {
	Username    string       `json:"username"`
	Avatar_url  string       `json:"avatar_url"`
	Content     string       `json:"content"`
	Embeds      []Embed      `json:"embeds"`
	Attachments []Attachment `json:"attachments"`
}

func postToDiscord(idx int, productName string, variant string, price float64, imageUrl string, checkoutLink string, discordWebhook string) error {
	now := time.Now()
	timestamp := fmt.Sprintf("%02d:%02d:%02d.%03d", now.Hour(), now.Minute(), now.Second(), now.Nanosecond()/1e6)
	fields := []Field{
		{
			Name:   "Product Name",
			Value:  productName,
			Inline: false,
		},
		{
			Name:   "Variant",
			Value:  variant,
			Inline: false,
		},
		{
			Name:   "Price",
			Value:  fmt.Sprintf("%.2f", price),
			Inline: false,
		},
		{
			Name:   "Task No",
			Value:  fmt.Sprintf("%d", idx+1),
			Inline: false,
		},
	}

	embedTitle := productName
	embedColor := 0x00FF00 // Green color

	if checkoutLink == "" {
		embedTitle = "Checkout Failed!"
		embedColor = 0xFF0000 // Red color
	} else {
		fields = append(fields, Field{
			Name:   "Checkout Link",
			Value:  fmt.Sprintf("||%s||", checkoutLink),
			Inline: false,
		})
	}

	embed := Embed{
		Title:     embedTitle,
		Url:       "",
		Fields:    fields,
		Color:     embedColor,
		Timestamp: now,
		Thumbnail: Thumbnail{
			Url: imageUrl,
		},
		Footer: Footer{
			Text: fmt.Sprintf("v2 | Easystore Bot - %s", timestamp),
		},
	}

	hook := Hook{
		Username: "Easystore Bot",
		Embeds:   []Embed{embed},
	}

	payload, err := json.Marshal(hook)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", discordWebhook, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("received non-204 response status: %d", resp.StatusCode)
	}

	return nil
}
