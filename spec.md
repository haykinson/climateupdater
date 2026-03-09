# Climate Tracker Specifications

## 1. Overview
The goal of the Climate Tracker is to ingest historical and current daily temperature data from Climate Reanalyzer, calculate the number of "record temperature days" for requested years, and present this data through a simple web interface. 

The tool will support 6 regions based on the provided endpoints:
- Global
- Northern Hemisphere
- Southern Hemisphere
- Arctic
- Antarctic
- Tropics

## 2. Definitions and Metrics
- **Record Temperature Day**: For a given year $Y$ and a specific day of the year $d$ (e.g., Jan 5th), the temperature is considered a "record" if it is strictly greater than the temperature on day $d$ for all recorded years prior to $Y$.
- **Full Year Records**: The total number of record temperature days that occurred over the entire 365/366 days of year $Y$.
- **Year-To-Date (YTD) Records**: The number of record temperature days that occurred in year $Y$ *up to the current day of the current year*. This allows for an apples-to-apples comparison of the current year (which is incomplete) against past years at the exact same point in time.

## 3. Data Ingestion & Processing (Backend)
- **Language**: Go (restricted to version 1.18 features only).
- **Fetching Strategy**: The backend will fetch the JSON data from the 6 URLs once daily and cache it in memory.
- **Parsing**: 
  - The JSON array contains objects with a `name` parameter. The backend will filter out non-year entries (like `"1979-2000"` means) by ensuring `name` parses as an integer.
  - Leap years will be accounted for by aligning day indices (0 to 365). Missing data handles (`null` or out-of-bounds) will be ignored.
- **Calculation Logic**:
  - The system will iterate chronologically from the earliest year (1940) to the most recent.
  - It will maintain an array of 366 "historical maximums".
  - For each year, it will check how many daily temperatures exceed the running historical maximum.
  - It will store both the Total Records and YTD Records for each year. YTD is bounded by the last populated data index of the *current* year.

## 4. API Endpoints
- `GET /api/regions`: Returns the list of supported regions.
- `GET /api/records?region={region_id}`: Returns a JSON array of record counts per year.
  - Response format:
    ```json
    [
      { "year": 2024, "full_year_records": 15, "ytd_records": 5 },
      { "year": 2025, "full_year_records": null, "ytd_records": ... }
    ]
    ```

## 5. Client (Frontend)
- **Stack**: HTML, TailwindCSS (via CDN for simplicity, or built if preferred), and Vanilla JavaScript.
- **UI Design**:
  - A clean, modern interface using a dark mode theme and responsive design.
  - A dropdown/tab selector to switch between the 6 regions.
  - A primary display emphasizing the **Current Year YTD Records**.
  - A data table or bar chart comparing the current year's YTD to past years' YTD and Full Year totals.
  - Focus will be on usability and clear comparison of the data to visually highlight climate trends.
