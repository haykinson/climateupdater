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
- **Baseline Period**: The first 9 years of data (typically 1940-1948) are used strictly as a baseline to establish historical maximums. They are excluded from the calculated records to prevent artificially inflating record counts during early years.
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
  - A dynamic, side-by-side masonry grid layout displaying all 6 regions concurrently.
  - For each region, a primary display emphasizing the **Current Year YTD Records**.
  - For each region, a data table and top 5 list comparing the current year's YTD to past years' YTD and Full Year totals.
  - Focus will be on usability and clear comparison of the data to visually highlight climate trends globally.

## 6. Trend Visualization

No backend changes are required for this section. All data needed for trend visualization is already present in the `/api/records` response; calculations and rendering are performed entirely in the frontend.

### 6.1 YTD Trend Sparkline

Each region card will include a full-width SVG bar chart placed between the YTD hero number and the Top 5 list.

- **Data**: One vertical bar per year (post-baseline), representing that year's `ytd_records`.
- **Coloring**: Bars are heat-colored on a per-region scale — the minimum YTD value in the dataset maps to a cool blue (`#3b82f6`) and the maximum maps to a hot red (`#ef4444`), with smooth interpolation in between. This makes high-record years visually pop without needing to read the numbers.
- **Current year highlight**: The current year's bar is always rendered with a bright white/accent top edge and a subtle glow, so it stands out even if it is not the tallest bar.
- **Hover tooltip**: Hovering a bar shows a small floating label with the year and YTD count. Implemented with SVG `<title>` elements or a positioned `<div>` updated via `mousemove`.
- **Dimensions**: Full card width, ~64px tall. Rendered as inline SVG — no external charting library.
- **Axis**: A faint baseline and a single horizontal reference line at the dataset mean help orient the viewer without cluttering the chart.

### 6.2 Trend Badge

A compact inline badge is displayed directly beneath the large YTD number in each card's hero section.

- **Metric**: Compares the current year's YTD count against the **5-year rolling average** of YTD records for the same YTD cutoff (i.e., the average of the preceding 5 completed years' YTD values).
- **Display format**: `↑ +42% vs 5yr avg` or `↓ −12% vs 5yr avg`.
- **Color**: Green (`text-emerald-400`) when below average, red (`text-red-400`) when above. Because more record-temperature days indicates warming, being above average is the alarming signal and should be red.
- **Fallback**: If fewer than 5 prior years of data exist for a region, the badge is omitted.

### 6.3 Inline Bar in the Historical Table

Each row in the scrollable historical table will include a subtle proportional bar behind the YTD value cell.

- Rendered as a `<div>` with a fixed-width container and a colored inner `<div>` whose width is set as a percentage of the column-maximum YTD value for that region.
- The bar is a translucent red (`bg-red-500/20`) so the number remains readable in front of it.
- The current year's bar uses a slightly brighter tint (`bg-red-500/40`) to stay consistent with the existing accent highlighting on that row.
- This makes the relative magnitude of each year scannable at a glance without leaving the table.
