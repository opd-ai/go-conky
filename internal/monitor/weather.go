// Package monitor provides weather monitoring via METAR data.
package monitor

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// WeatherStats contains weather data from METAR reports.
type WeatherStats struct {
	// StationID is the ICAO airport code (e.g., "KJFK").
	StationID string
	// Temperature is the temperature in Celsius.
	Temperature float64
	// DewPoint is the dew point temperature in Celsius.
	DewPoint float64
	// Humidity is the relative humidity percentage (0-100).
	Humidity float64
	// Pressure is the barometric pressure in hPa (millibars).
	Pressure float64
	// WindSpeed is the wind speed in knots.
	WindSpeed float64
	// WindDirection is the wind direction in degrees (0-360).
	WindDirection int
	// WindGust is the wind gust speed in knots (0 if no gusts).
	WindGust float64
	// Visibility is the visibility in statute miles.
	Visibility float64
	// Condition is the weather condition description.
	Condition string
	// Cloud is the cloud coverage description.
	Cloud string
	// RawMETAR is the raw METAR string.
	RawMETAR string
	// LastUpdate is when the data was fetched.
	LastUpdate time.Time
	// Error contains any error message.
	Error string
}

// weatherReader reads weather data from METAR sources.
type weatherReader struct {
	mu          sync.RWMutex
	cache       map[string]*weatherCacheEntry
	httpClient  *http.Client
	metarURL    string
	minInterval time.Duration
}

// weatherCacheEntry stores cached weather data for a station.
type weatherCacheEntry struct {
	stats     WeatherStats
	fetchTime time.Time
}

// newWeatherReader creates a new weather reader.
func newWeatherReader() *weatherReader {
	return &weatherReader{
		cache: make(map[string]*weatherCacheEntry),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		// Uses aviationweather.gov which provides free METAR data
		metarURL:    "https://aviationweather.gov/api/data/metar?ids=%s&format=raw",
		minInterval: 10 * time.Minute, // METAR data updates roughly every hour
	}
}

// ReadWeather returns weather data for the given station ID.
func (r *weatherReader) ReadWeather(stationID string) (WeatherStats, error) {
	stationID = strings.ToUpper(strings.TrimSpace(stationID))
	if stationID == "" {
		return WeatherStats{}, fmt.Errorf("station ID is required")
	}

	r.mu.RLock()
	entry, ok := r.cache[stationID]
	r.mu.RUnlock()

	// Return cached data if fresh
	if ok && time.Since(entry.fetchTime) < r.minInterval {
		return entry.stats, nil
	}

	// Fetch new data
	stats, err := r.fetchMETAR(stationID)
	if err != nil {
		// Return cached data with error if available
		if ok {
			stats = entry.stats
			stats.Error = err.Error()
			return stats, nil
		}
		return WeatherStats{StationID: stationID, Error: err.Error()}, err
	}

	// Cache the result
	r.mu.Lock()
	r.cache[stationID] = &weatherCacheEntry{
		stats:     stats,
		fetchTime: time.Now(),
	}
	r.mu.Unlock()

	return stats, nil
}

// fetchMETAR fetches and parses METAR data from aviationweather.gov.
func (r *weatherReader) fetchMETAR(stationID string) (WeatherStats, error) {
	url := fmt.Sprintf(r.metarURL, stationID)

	resp, err := r.httpClient.Get(url)
	if err != nil {
		return WeatherStats{}, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WeatherStats{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherStats{}, fmt.Errorf("read failed: %w", err)
	}

	raw := strings.TrimSpace(string(body))
	if raw == "" || strings.Contains(raw, "No data found") {
		return WeatherStats{}, fmt.Errorf("no data for station %s", stationID)
	}

	return r.parseMETAR(stationID, raw), nil
}

// parseMETAR parses a raw METAR string into WeatherStats.
func (r *weatherReader) parseMETAR(stationID, raw string) WeatherStats {
	stats := WeatherStats{
		StationID:  stationID,
		RawMETAR:   raw,
		LastUpdate: time.Now(),
	}

	parts := strings.Fields(raw)
	if len(parts) < 3 {
		return stats
	}

	for _, part := range parts {
		// Wind: dddssKT or dddssGggKT (direction, speed, optional gust)
		if windMatch := regexp.MustCompile(`^(\d{3}|VRB)(\d{2,3})(?:G(\d{2,3}))?KT$`).FindStringSubmatch(part); windMatch != nil {
			if windMatch[1] == "VRB" {
				stats.WindDirection = -1 // Variable
			} else {
				stats.WindDirection, _ = strconv.Atoi(windMatch[1])
			}
			speed, _ := strconv.ParseFloat(windMatch[2], 64)
			stats.WindSpeed = speed
			if windMatch[3] != "" {
				gust, _ := strconv.ParseFloat(windMatch[3], 64)
				stats.WindGust = gust
			}
			continue
		}

		// Temperature/Dewpoint: TT/DD or MTT/MDD (M prefix for negative)
		if tempMatch := regexp.MustCompile(`^(M?\d{2})/(M?\d{2})$`).FindStringSubmatch(part); tempMatch != nil {
			stats.Temperature = parseMETARTemp(tempMatch[1])
			stats.DewPoint = parseMETARTemp(tempMatch[2])
			// Calculate relative humidity from temperature and dew point
			stats.Humidity = calculateHumidity(stats.Temperature, stats.DewPoint)
			continue
		}

		// Altimeter setting: Annnn (inches of mercury * 100)
		if altMatch := regexp.MustCompile(`^A(\d{4})$`).FindStringSubmatch(part); altMatch != nil {
			inHg, _ := strconv.ParseFloat(altMatch[1], 64)
			inHg /= 100.0
			stats.Pressure = inHg * 33.8639 // Convert to hPa
			continue
		}

		// QNH pressure: Qnnnn (hPa)
		if qnhMatch := regexp.MustCompile(`^Q(\d{4})$`).FindStringSubmatch(part); qnhMatch != nil {
			stats.Pressure, _ = strconv.ParseFloat(qnhMatch[1], 64)
			continue
		}

		// Visibility: nnSM (statute miles)
		if visMatch := regexp.MustCompile(`^(\d+)SM$`).FindStringSubmatch(part); visMatch != nil {
			stats.Visibility, _ = strconv.ParseFloat(visMatch[1], 64)
			continue
		}

		// Visibility: M1/4SM (less than 1/4 mile)
		if part == "M1/4SM" {
			stats.Visibility = 0.25
			continue
		}

		// Fractional visibility: n/nSM
		if visMatch := regexp.MustCompile(`^(\d+)/(\d+)SM$`).FindStringSubmatch(part); visMatch != nil {
			num, _ := strconv.ParseFloat(visMatch[1], 64)
			denom, _ := strconv.ParseFloat(visMatch[2], 64)
			if denom > 0 {
				stats.Visibility = num / denom
			}
			continue
		}

		// Cloud coverage - only use first cloud layer for display
		if stats.Cloud == "" {
			if cloudMatch := regexp.MustCompile(`^(SKC|CLR|FEW|SCT|BKN|OVC|VV)(\d{3})?`).FindStringSubmatch(part); cloudMatch != nil {
				coverage := cloudMatch[1]
				switch coverage {
				case "SKC", "CLR":
					stats.Cloud = "clear"
				case "FEW":
					stats.Cloud = "few clouds"
				case "SCT":
					stats.Cloud = "scattered clouds"
				case "BKN":
					stats.Cloud = "broken clouds"
				case "OVC":
					stats.Cloud = "overcast"
				case "VV":
					stats.Cloud = "vertical visibility"
				}
				continue
			}
		}

		// Weather phenomena (simplified)
		stats.Condition = parseWeatherCondition(part, stats.Condition)
	}

	// Default condition based on clouds if no specific weather
	if stats.Condition == "" && stats.Cloud != "" {
		stats.Condition = stats.Cloud
	} else if stats.Condition == "" {
		stats.Condition = "unknown"
	}

	return stats
}

// parseMETARTemp parses a METAR temperature string (e.g., "M02" = -2).
func parseMETARTemp(s string) float64 {
	negative := strings.HasPrefix(s, "M")
	if negative {
		s = s[1:]
	}
	temp, _ := strconv.ParseFloat(s, 64)
	if negative {
		temp = -temp
	}
	return temp
}

// calculateHumidity calculates relative humidity from temperature and dew point.
// Uses the Magnus-Tetens approximation.
func calculateHumidity(tempC, dewPointC float64) float64 {
	a := 17.27
	b := 237.7

	alpha := (a * dewPointC) / (b + dewPointC)
	beta := (a * tempC) / (b + tempC)

	rh := 100.0 * (exp(alpha) / exp(beta))
	if rh > 100 {
		rh = 100
	}
	if rh < 0 {
		rh = 0
	}
	return rh
}

// exp is a simple exponential function approximation.
func exp(x float64) float64 {
	// Use standard library via import; here's a simple Taylor series for small x
	// For production, use math.Exp
	result := 1.0
	term := 1.0
	for i := 1; i <= 20; i++ {
		term *= x / float64(i)
		result += term
	}
	return result
}

// parseWeatherCondition parses METAR weather codes into descriptions.
func parseWeatherCondition(part, existing string) string {
	// Check for intensity prefix
	intensity := ""
	if strings.HasPrefix(part, "+") {
		intensity = "heavy "
		part = part[1:]
	} else if strings.HasPrefix(part, "-") {
		intensity = "light "
		part = part[1:]
	}

	// Check conditions in priority order (longer codes first to avoid partial matches)
	conditionsOrdered := []struct {
		code string
		desc string
	}{
		{"VCSH", "showers nearby"},
		{"VCTS", "thunderstorm nearby"},
		{"TSRA", "thunderstorm with rain"},
		{"TS", "thunderstorm"},
		{"SH", "showers"},
		{"RA", "rain"},
		{"SN", "snow"},
		{"DZ", "drizzle"},
		{"FG", "fog"},
		{"BR", "mist"},
		{"HZ", "haze"},
		{"GR", "hail"},
		{"IC", "ice crystals"},
		{"PL", "ice pellets"},
		{"FZ", "freezing"},
	}

	for _, c := range conditionsOrdered {
		if strings.Contains(part, c.code) {
			newCondition := intensity + c.desc
			if existing != "" && existing != newCondition {
				return existing + ", " + newCondition
			}
			return newCondition
		}
	}

	return existing
}

// GetField returns a specific weather field as a string.
func (w *WeatherStats) GetField(field string) string {
	field = strings.ToLower(strings.TrimSpace(field))
	switch field {
	case "temp", "temperature":
		return fmt.Sprintf("%.0f", w.Temperature)
	case "temp_f", "temperature_f":
		return fmt.Sprintf("%.0f", w.Temperature*9/5+32)
	case "dewpoint", "dew_point":
		return fmt.Sprintf("%.0f", w.DewPoint)
	case "humidity":
		return fmt.Sprintf("%.0f", w.Humidity)
	case "pressure", "pressure_mb":
		return fmt.Sprintf("%.0f", w.Pressure)
	case "pressure_inhg":
		return fmt.Sprintf("%.2f", w.Pressure/33.8639)
	case "wind", "wind_speed":
		return fmt.Sprintf("%.0f", w.WindSpeed)
	case "wind_dir", "wind_direction":
		if w.WindDirection == -1 {
			return "VRB"
		}
		return fmt.Sprintf("%d", w.WindDirection)
	case "wind_dir_compass":
		return degreesToCompass(w.WindDirection)
	case "wind_gust":
		return fmt.Sprintf("%.0f", w.WindGust)
	case "visibility":
		return fmt.Sprintf("%.1f", w.Visibility)
	case "condition", "weather":
		return w.Condition
	case "cloud", "clouds":
		return w.Cloud
	case "raw", "metar":
		return w.RawMETAR
	default:
		return ""
	}
}

// degreesToCompass converts wind direction in degrees to compass direction.
func degreesToCompass(degrees int) string {
	if degrees < 0 {
		return "VRB"
	}
	directions := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE",
		"S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	index := ((degrees + 11) / 22) % 16
	return directions[index]
}

// ClearCache clears the weather cache.
func (r *weatherReader) ClearCache() {
	r.mu.Lock()
	r.cache = make(map[string]*weatherCacheEntry)
	r.mu.Unlock()
}
