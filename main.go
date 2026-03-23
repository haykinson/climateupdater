package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

func main() {
	ds := NewDataStore()

	// Initial synchronous block to make sure we have data before serving
	log.Println("Fetching initial data...")
	for _, region := range regions {
		data, err := fetchRegionData(region.URL)
		if err != nil {
			log.Printf("Failed initial fetch for %s: %v", region.Name, err)
			continue
		}
		ds.Set(region.ID, data)
		time.Sleep(1 * time.Second) // Polite delay
	}
	log.Println("Initial fetch complete.")

	// Start background refresher
	StartDailyWorker(ds)

	// API Handlers (Go 1.22 structured routing)
	http.HandleFunc("GET /api/regions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Send standard regions array
		json.NewEncoder(w).Encode(regions)
	})

	http.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"last_updated": ds.GetLastUpdated(),
			"data_through": ds.GetDataThrough(),
		})
	})

	http.HandleFunc("GET /api/recent", func(w http.ResponseWriter, r *http.Request) {
		regionID := r.URL.Query().Get("region")
		if regionID == "" {
			http.Error(w, "Missing region parameter", http.StatusBadRequest)
			return
		}

		data, ok := ds.Get(regionID)
		if !ok {
			http.Error(w, "Region data not found or not yet loaded", http.StatusNotFound)
			return
		}

		// Extract the climatological mean (the first non-integer-named entry, e.g. "1979-2000").
		var climMean []float64
		for _, yd := range data {
			if _, err := strconv.Atoi(yd.Name); err != nil {
				climMean = yd.Data
				break
			}
		}

		results := CalculateRecentDays(data, climMean, 90)
		if results == nil {
			results = make([]RecentDayData, 0)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})

	http.HandleFunc("GET /api/records", func(w http.ResponseWriter, r *http.Request) {
		regionID := r.URL.Query().Get("region")
		if regionID == "" {
			http.Error(w, "Missing region parameter", http.StatusBadRequest)
			return
		}

		data, ok := ds.Get(regionID)
		if !ok {
			http.Error(w, "Region data not found or not yet loaded", http.StatusNotFound)
			return
		}

		results := CalculateRecords(data)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})

	// Static Files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("GET /", fs)

	port := "8081"
	log.Printf("Server listening on :%s", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
