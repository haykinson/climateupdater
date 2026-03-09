package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Region represents the available regions to fetch data for.
type Region struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// YearData represents the raw JSON data for a given year.
type YearData struct {
	Name string    `json:"name"`
	Data []float64 `json:"data"`
}

var regions = []Region{
	{ID: "world", Name: "Global", URL: "https://climatereanalyzer.org/clim/t2_daily/json/era5_world_t2_day.json"},
	{ID: "nh", Name: "Northern Hemisphere", URL: "https://climatereanalyzer.org/clim/t2_daily/json/era5_nh_t2_day.json"},
	{ID: "sh", Name: "Southern Hemisphere", URL: "https://climatereanalyzer.org/clim/t2_daily/json/era5_sh_t2_day.json"},
	{ID: "arctic", Name: "Arctic", URL: "https://climatereanalyzer.org/clim/t2_daily/json/era5_arctic_t2_day.json"},
	{ID: "antarctic", Name: "Antarctic", URL: "https://climatereanalyzer.org/clim/t2_daily/json/era5_antarctic_t2_day.json"},
	{ID: "tropics", Name: "Tropics", URL: "https://climatereanalyzer.org/clim/t2_daily/json/era5_tropics_t2_day.json"},
}

// DataStore holds the latest fetched and parsed data for each region securely.
type DataStore struct {
	mu   sync.RWMutex
	data map[string][]YearData
}

// NewDataStore creates a new DataStore.
func NewDataStore() *DataStore {
	return &DataStore{
		data: make(map[string][]YearData),
	}
}

// Set stores the data for a given region.
func (ds *DataStore) Set(regionID string, data []YearData) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.data[regionID] = data
}

// Get returns the data for a given region.
func (ds *DataStore) Get(regionID string) ([]YearData, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	data, ok := ds.data[regionID]
	return data, ok
}

// fetchRegionData fetches and parses the JSON from the given URL.
// It filters out any invalid entries where the 'name' field is not a year (e.g. '1979-2000').
func fetchRegionData(url string) ([]YearData, error) {
	// Custom HTTP client to add headers if required
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 Tracker/1.0 (haykinson/climateupdater)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawData []struct {
		Name string        `json:"name"`
		Data []interface{} `json:"data"` // some data points might be null at the end of the year
	}

	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, err
	}

	var filteredData []YearData
	for _, entry := range rawData {
		// Only include entry if 'name' is a valid year (integer)
		if _, err := strconv.Atoi(entry.Name); err != nil {
			continue // Skip averages like '1979-2000'
		}

		// Parse float64 data, converting nulls to a sentinel value (e.g., -999.0) or simply truncating up to first null
		parsedData := make([]float64, len(entry.Data))
		for i, v := range entry.Data {
			if v == nil {
				// We can stop here, or fill with NaN. Go json.Unmarshal parses numbers as float64
				parsedData[i] = -999.0 // Sentinel for bad/null data
			} else {
				if f, ok := v.(float64); ok {
					parsedData[i] = f
				}
			}
		}
		filteredData = append(filteredData, YearData{
			Name: entry.Name,
			Data: parsedData,
		})
	}

	return filteredData, nil
}

// StartDailyWorker starts a background goroutine that fetches data daily.
func StartDailyWorker(ds *DataStore) {
	go func() {
		for {
			log.Println("Starting daily fetch of climate data...")
			for _, region := range regions {
				data, err := fetchRegionData(region.URL)
				if err != nil {
					log.Printf("Error fetching data for %s: %v", region.Name, err)
					continue
				}
				ds.Set(region.ID, data)
				log.Printf("Successfully fetched %d years for %s", len(data), region.Name)
				// Add small delay between fetches to be polite to the server
				time.Sleep(2 * time.Second)
			}
			log.Println("Fetch complete. Sleeping for 24 hours.")
			time.Sleep(24 * time.Hour)
		}
	}()
}
