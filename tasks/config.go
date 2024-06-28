package tasks

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	DiscordWebhook string `json:"DiscordWebhook"`
}

var config Config

func LoadConfig() error {
	bytes, err := os.ReadFile("config.json")
	if err != nil {
		return fmt.Errorf("error opening config.json file: %w", err)
	}

	if err := json.Unmarshal(bytes, &config); err != nil {
		return fmt.Errorf("error unmarshalling config.json: %w", err)
	}

	return nil
}

func GetDiscordWebhook() string {
	return config.DiscordWebhook
}
