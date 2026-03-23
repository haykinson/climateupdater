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
	mu          sync.RWMutex
	data        map[string][]YearData
	lastUpdated time.Time
	dataThrough time.Time
}

// NewDataStore creates a new DataStore.
func NewDataStore() *DataStore {
	return &DataStore{
		data: make(map[string][]YearData),
	}
}

// Set stores the data for a given region and updates the dataThrough date.
func (ds *DataStore) Set(regionID string, data []YearData) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.data[regionID] = data
	ds.lastUpdated = time.Now()
	if t, ok := latestDataDate(data); ok && (ds.dataThrough.IsZero() || t.After(ds.dataThrough)) {
		ds.dataThrough = t
	}
}

// Get returns the data for a given region.
func (ds *DataStore) Get(regionID string) ([]YearData, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	data, ok := ds.data[regionID]
	return data, ok
}

// GetLastUpdated returns the time the data was last updated.
func (ds *DataStore) GetLastUpdated() time.Time {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.lastUpdated
}

// GetDataThrough returns the date of the most recent valid data point across all regions.
func (ds *DataStore) GetDataThrough() time.Time {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.dataThrough
}

// latestDataDate finds the calendar date of the last valid temperature reading in a dataset.
func latestDataDate(data []YearData) (time.Time, bool) {
	for i := len(data) - 1; i >= 0; i-- {
		year, err := strconv.Atoi(data[i].Name)
		if err != nil {
			continue // skip clim mean entries
		}
		for j := len(data[i].Data) - 1; j >= 0; j-- {
			if data[i].Data[j] > -900.0 {
				t := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, j)
				return t, true
			}
		}
	}
	return time.Time{}, false
}

// fetchRegionData fetches and parses the JSON from the given URL.
// All entries are preserved, including climatological means like "1979-2000".
// Callers that need only year entries filter by strconv.Atoi on the Name field.
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

	var parsedEntries []YearData
	for _, entry := range rawData {
		parsedData := make([]float64, len(entry.Data))
		for i, v := range entry.Data {
			if v == nil {
				parsedData[i] = -999.0 // Sentinel for bad/null data
			} else {
				if f, ok := v.(float64); ok {
					parsedData[i] = f
				}
			}
		}
		parsedEntries = append(parsedEntries, YearData{
			Name: entry.Name,
			Data: parsedData,
		})
	}

	return parsedEntries, nil
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
