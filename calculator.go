package main

import (
	"math"
	"strconv"
)

// RecordResult represents the YTD and Full Year record counts for a given year.
type RecordResult struct {
	Year            int  `json:"year"`
	FullYearRecords *int `json:"full_year_records"` // Pointer to int, allows null if year incomplete
	YTDRecords      int  `json:"ytd_records"`
}

// RecentDayData holds per-day temperature info for the recent activity strip.
type RecentDayData struct {
	DayIndex int     `json:"day_index"`
	Temp     float64 `json:"temp"`
	ClimAvg  float64 `json:"clim_avg"` // -999.0 if no climatological mean available
	IsRecord bool    `json:"is_record"`
}

// CalculateRecords iterates through chronological year data and counts "Record Temperature Days".
func CalculateRecords(data []YearData) []RecordResult {
	if len(data) == 0 {
		return nil
	}

	// Find the most recent integer-named year to determine the current day index.
	var latestYear *YearData
	for i := len(data) - 1; i >= 0; i-- {
		if _, err := strconv.Atoi(data[i].Name); err == nil {
			latestYear = &data[i]
			break
		}
	}
	if latestYear == nil {
		return nil
	}

	currentDayIndex := -1
	for i := len(latestYear.Data) - 1; i >= 0; i-- {
		if latestYear.Data[i] > -900.0 { // Valid data
			currentDayIndex = i
			break
		}
	}

	// This stores the highest temperature seen for any given day of the year (0-365)
	// prior to the *current* year being evaluated.
	historicalMaximums := make([]float64, 366)
	for i := range 366 {
		historicalMaximums[i] = -math.MaxFloat64 // Start impossibly low
	}

	var results []RecordResult

	// The first 9 years (typically 1940-1948) are considered a baseline period.
	// We want to register their maximums, but NOT include them in the final output
	// so the UI isn't spammed with early artificially high record years.
	// yearCount tracks only actual integer-named year entries, so non-year entries
	// (e.g. climatological means like "1979-2000") do not shift the baseline cutoff.
	yearCount := 0
	for _, yd := range data {
		year, err := strconv.Atoi(yd.Name)
		if err != nil {
			continue // Skip climatological mean entries
		}
		yearCount++

		ytdCount := 0
		fullCount := 0
		validDaysThisYear := 0

		for d, temp := range yd.Data {
			if temp < -900.0 { // Sentinel for missing data
				continue
			}

			validDaysThisYear++

			if temp > historicalMaximums[d] {
				if d <= currentDayIndex {
					ytdCount++
				}
				fullCount++

				// Update historical maximum for next year's comparison
				historicalMaximums[d] = temp
			}
		}

		res := RecordResult{
			Year:       year,
			YTDRecords: ytdCount,
		}

		if float64(validDaysThisYear)/float64(len(yd.Data)) > 0.6 || validDaysThisYear >= 360 {
			countCopy := fullCount
			res.FullYearRecords = &countCopy
		}

		// Only append to results if we have passed the initial 9 year baseline period
		if yearCount > 9 {
			results = append(results, res)
		}
	}

	return results
}

// CalculateRecentDays returns per-day temperature data for the last n days of the current year,
// including deviation from the climatological mean and whether each day set a new record.
func CalculateRecentDays(data []YearData, climMean []float64, n int) []RecentDayData {
	// Find the index of the most recent integer-named year.
	currentYearIdx := -1
	for i := range data {
		if _, err := strconv.Atoi(data[i].Name); err == nil {
			currentYearIdx = i
		}
	}
	if currentYearIdx < 0 {
		return nil
	}

	// Build historical maximums from all years strictly before the current year.
	historicalMaximums := make([]float64, 366)
	for i := range 366 {
		historicalMaximums[i] = -math.MaxFloat64
	}
	for i := 0; i < currentYearIdx; i++ {
		if _, err := strconv.Atoi(data[i].Name); err != nil {
			continue // skip non-year entries
		}
		for d, temp := range data[i].Data {
			if temp > -900.0 && temp > historicalMaximums[d] {
				historicalMaximums[d] = temp
			}
		}
	}

	currentYear := data[currentYearIdx]

	// Find the last valid day index in the current year.
	lastValidIdx := -1
	for i := len(currentYear.Data) - 1; i >= 0; i-- {
		if currentYear.Data[i] > -900.0 {
			lastValidIdx = i
			break
		}
	}
	if lastValidIdx < 0 {
		return nil
	}

	startIdx := lastValidIdx - n + 1
	if startIdx < 0 {
		startIdx = 0
	}

	var result []RecentDayData
	for d := startIdx; d <= lastValidIdx; d++ {
		temp := currentYear.Data[d]
		if temp < -900.0 {
			continue
		}

		avg := -999.0 // sentinel: no climatological mean available
		if climMean != nil && d < len(climMean) && climMean[d] > -900.0 {
			avg = math.Round(climMean[d]*100) / 100
		}

		result = append(result, RecentDayData{
			DayIndex: d,
			Temp:     math.Round(temp*100) / 100,
			ClimAvg:  avg,
			IsRecord: temp > historicalMaximums[d],
		})
	}

	return result
}
