package geo

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

type Locator func(lat, lon float64, address string) (Location, error)

type Location struct {
	City    string
	ZipCode string
}

func (l Location) IsInGironde() bool {
	return l.ZipCode[:2] == "33"
}

type Communes []Commune

type Commune struct {
	Code string `json:"code"`
	Nom  string `json:"nom"`
}

func FindLocation(lat, lon float64, address string) (Location, error) {
	getURL, err := url.Parse("https://geo.api.gouv.fr/communes")
	if err != nil {
		return Location{}, err
	}
	values := getURL.Query()
	values.Add("lat", fmt.Sprintf("%f", lat))
	values.Add("lon", fmt.Sprintf("%f", lon))
	values.Add("fields", "codesPostaux")
	values.Add("format", "json")
	values.Add("geometry", "centre")
	getURL.RawQuery = values.Encode()
	resp, err := http.Get(getURL.String())
	if err != nil {
		return Location{}, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("no location found for coordinates %f,%f", lat, lon)
		return Location{
			City:    fmt.Sprintf("🗺️ %s", address),
			ZipCode: "00000",
		}, nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Location{}, err
	}
	var communes Communes
	if err := json.Unmarshal(body, &communes); err != nil {
		return Location{}, err
	}
	if len(communes) == 0 {
		return Location{
			City:    fmt.Sprintf("🗺️ %s", address),
			ZipCode: "00000",
		}, nil
	}
	return Location{
		City:    communes[0].Nom,
		ZipCode: communes[0].Code,
	}, nil
}

func FakeLocate(lat, lon float64, address string) (Location, error) {
	if address == "Lormont, France" {
		return Location{City: address, ZipCode: "33310"}, nil
	}
	if lat != 0 && lon != 0 {
		return Location{City: address, ZipCode: "12345"}, nil
	}
	return Location{City: address, ZipCode: "00000"}, nil
}
