package main

import (
	"strconv"
	"testing"
)

func TestCalculateRecords(t *testing.T) {
	// Let's create a mock dataset of 10 years to pass the 9-year baseline
	// Years 1-9 (1940-1948) -> Baseline.
	// Year 10 (1949) -> we'll test exceeding previous baseline. Length 5 days.
	// Year 11 (1950) -> current year, only 3 days of data.

	var data []YearData

	// Baseline Years 1-9
	for i := 0; i < 9; i++ {
		data = append(data, YearData{
			Name: strconv.Itoa(1940 + i),
			Data: []float64{10, 11, 12, 13, 14, -999, -999},
		})
	}

	// Year 10 (1949): 9 (0), 12 (1), 11 (0), 14 (1), 13 (0). Total records: 2
	data = append(data, YearData{
		Name: "1949",
		Data: []float64{9, 12, 11, 14, 13, -999, -999},
	})

	// Year 11 (1950, Current): 15 (1), 11 (0), 13 (1), -999, -999.
	// Last valid index is 2 (3rd day).
	// YTD window is indices 0, 1, 2.
	// Records YTD: 2. Total Records expected: 2 (Full is null)
	data = append(data, YearData{
		Name: "1950",
		Data: []float64{15, 11, 13, -999, -999, -999, -999},
	})

	results := CalculateRecords(data)

	if len(results) != 2 {
		t.Fatalf("Expected 2 results (after 9 year baseline), got %d", len(results))
	}

	// Result 2 (Year 10 - 1949)
	y10Full := 2
	if results[0].FullYearRecords == nil {
		t.Errorf("Year 10 Full expected %d, got nil", y10Full)
	} else if *results[0].FullYearRecords != y10Full {
		t.Errorf("Year 10 Full expected %d, got %d", y10Full, *results[0].FullYearRecords)
	}
	if results[0].YTDRecords != 1 {
		t.Errorf("Year 10 YTD expected 1, got %d", results[0].YTDRecords)
	}

	// Result 3 (Year 11 - 1950)
	if results[1].FullYearRecords != nil {
		t.Errorf("Year 11 Full expected nil")
	}
	if results[1].YTDRecords != 2 {
		t.Errorf("Year 11 YTD expected 2, got %d", results[1].YTDRecords)
	}
}

func TestCalculateRecordsEmpty(t *testing.T) {
	if res := CalculateRecords(nil); res != nil {
		t.Errorf("Expected nil on empty data")
	}
}
