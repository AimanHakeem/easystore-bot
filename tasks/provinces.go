package tasks

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Province struct {
	ID         int    `json:"id"`
	RCountryID int    `json:"r_country_id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
}

type ProvinceResponse struct {
	Provinces []Province `json:"provinces"`
}

var provinces []Province

func LoadProvinces(link string) error {
	resp, err := http.Get(link)
	if err != nil {
		return fmt.Errorf("error making request to %s: %w", link, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
	}

	var provinceResponse ProvinceResponse
	if err := json.NewDecoder(resp.Body).Decode(&provinceResponse); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	provinces = provinceResponse.Provinces
	return nil
}

func GetProvinceCode(provinceName string) (string, error) {
	for _, province := range provinces {
		if province.Name == provinceName {
			return province.Code, nil
		}
	}
	return "", fmt.Errorf("province not found: %s", provinceName)
}
