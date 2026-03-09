package main

import (
	"testing"
)

func TestCalculateRecords(t *testing.T) {
	// Let's create a mock dataset of 3 years
	// Year 1 (1940) -> everything is a record. Length 5 days.
	// Year 2 (1941) -> we'll test exceeding year 1. Length 5 days.
	// Year 3 (1942) -> current year, only 3 days of data.

	// Year 1: 10, 11, 12, 13, 14 (-999 padding)
	var y1 = YearData{
		Name: "1940",
		Data: []float64{10, 11, 12, 13, 14, -999, -999},
	}
	// Year 2: 9 (0), 12 (1), 11 (0), 14 (1), 13 (0). Total records: 2
	var y2 = YearData{
		Name: "1941",
		Data: []float64{9, 12, 11, 14, 13, -999, -999},
	}
	// Year 3 (Current): 15 (1), 11 (0), 13 (1), -999, -999.
	// Last valid index is 2 (3rd day).
	// YTD window is indices 0, 1, 2.
	// Records YTD: 2. Total Records expected: 2 (Full is null)
	var y3 = YearData{
		Name: "1942",
		Data: []float64{15, 11, 13, -999, -999, -999, -999},
	}

	data := []YearData{y1, y2, y3}

	results := CalculateRecords(data)

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Result 2 (1941)
	y2Full := 2
	if results[1].FullYearRecords == nil {
		t.Errorf("Year 2 Full expected %d, got nil", y2Full)
	} else if *results[1].FullYearRecords != y2Full {
		t.Errorf("Year 2 Full expected %d, got %d", y2Full, *results[1].FullYearRecords)
	}
	// YTD records: The YTD window is based on year 3, which is index 2.
	// Year 2 records within index <= 2: day 1 (index 1) which was 12. So 1 record.
	if results[1].YTDRecords != 1 {
		t.Errorf("Year 2 YTD expected 1, got %d", results[1].YTDRecords)
	}

	// Result 3 (1942)
	if results[2].FullYearRecords != nil {
		t.Errorf("Year 3 Full expected nil")
	}
	if results[2].YTDRecords != 2 {
		t.Errorf("Year 3 YTD expected 2, got %d", results[2].YTDRecords)
	}
}

func TestCalculateRecordsEmpty(t *testing.T) {
	if res := CalculateRecords(nil); res != nil {
		t.Errorf("Expected nil on empty data")
	}
}
