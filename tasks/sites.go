package tasks

import (
	"encoding/json"
	"fmt"
	"os"
)

type Site struct {
	Site string `json:"site"`
	Link string `json:"link"`
}

var sites []Site

func LoadSites() error {
	bytes, err := os.ReadFile("data/sites.json")
	if err != nil {
		return fmt.Errorf("error opening sites.json file: %w", err)
	}

	if err := json.Unmarshal(bytes, &sites); err != nil {
		return fmt.Errorf("error unmarshalling sites.json: %w", err)
	}

	return nil
}

func GetSiteLink(siteName string) (string, error) {
	for _, site := range sites {
		if site.Site == siteName {
			return site.Link, nil
		}
	}
	return "", fmt.Errorf("site not found: %s", siteName)
}
