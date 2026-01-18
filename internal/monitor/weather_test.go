package monitor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWeatherReader_ParseMETAR(t *testing.T) {
	r := newWeatherReader()

	tests := []struct {
		name           string
		stationID      string
		rawMETAR       string
		wantTemp       float64
		wantDewPoint   float64
		wantWindSpeed  float64
		wantWindDir    int
		wantWindGust   float64
		wantVisibility float64
		wantCloud      string
		wantCondition  string
	}{
		{
			name:           "basic METAR",
			stationID:      "KJFK",
			rawMETAR:       "KJFK 151756Z 31012KT 10SM FEW045 SCT250 22/06 A3012",
			wantTemp:       22,
			wantDewPoint:   6,
			wantWindSpeed:  12,
			wantWindDir:    310,
			wantVisibility: 10,
			wantCloud:      "few clouds",
		},
		{
			name:           "METAR with gusts",
			stationID:      "KORD",
			rawMETAR:       "KORD 151751Z 27018G28KT 10SM SCT050 BKN250 18/08 A2995",
			wantTemp:       18,
			wantDewPoint:   8,
			wantWindSpeed:  18,
			wantWindDir:    270,
			wantWindGust:   28,
			wantVisibility: 10,
			wantCloud:      "scattered clouds",
		},
		{
			name:          "METAR with weather phenomena",
			stationID:     "KSFO",
			rawMETAR:      "KSFO 151756Z 29008KT 8SM -RA FEW015 BKN025 OVC040 14/11 A3002",
			wantTemp:      14,
			wantDewPoint:  11,
			wantWindSpeed: 8,
			wantWindDir:   290,
			wantCondition: "light rain",
		},
		{
			name:          "METAR with negative temps",
			stationID:     "CYYZ",
			rawMETAR:      "CYYZ 151800Z 35015KT 15SM SKC M05/M12 A3032",
			wantTemp:      -5,
			wantDewPoint:  -12,
			wantWindSpeed: 15,
			wantWindDir:   350,
			wantCloud:     "clear",
		},
		{
			name:           "METAR with QNH pressure",
			stationID:      "EGLL",
			rawMETAR:       "EGLL 151750Z 24010KT 9999 FEW040 15/08 Q1023",
			wantTemp:       15,
			wantDewPoint:   8,
			wantWindSpeed:  10,
			wantWindDir:    240,
			wantVisibility: 0, // 9999 is METAR unlimited visibility notation
		},
		{
			name:          "METAR with variable wind",
			stationID:     "KLAX",
			rawMETAR:      "KLAX 151753Z VRB03KT 10SM CLR 18/12 A2998",
			wantTemp:      18,
			wantDewPoint:  12,
			wantWindSpeed: 3,
			wantWindDir:   -1, // Variable
			wantCloud:     "clear",
		},
		{
			name:          "METAR with thunderstorm",
			stationID:     "KDFW",
			rawMETAR:      "KDFW 151753Z 18012KT 3SM +TSRA BKN020 OVC040 24/22 A2980",
			wantCondition: "heavy thunderstorm",
		},
		{
			name:          "METAR with fog",
			stationID:     "KSFO",
			rawMETAR:      "KSFO 151756Z 00000KT 1/4SM FG VV002 12/12 A3010",
			wantCondition: "fog",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := r.parseMETAR(tt.stationID, tt.rawMETAR)

			if stats.StationID != tt.stationID {
				t.Errorf("StationID = %q, want %q", stats.StationID, tt.stationID)
			}

			if tt.wantTemp != 0 && stats.Temperature != tt.wantTemp {
				t.Errorf("Temperature = %v, want %v", stats.Temperature, tt.wantTemp)
			}

			if tt.wantDewPoint != 0 && stats.DewPoint != tt.wantDewPoint {
				t.Errorf("DewPoint = %v, want %v", stats.DewPoint, tt.wantDewPoint)
			}

			if tt.wantWindSpeed != 0 && stats.WindSpeed != tt.wantWindSpeed {
				t.Errorf("WindSpeed = %v, want %v", stats.WindSpeed, tt.wantWindSpeed)
			}

			if tt.wantWindDir != 0 && stats.WindDirection != tt.wantWindDir {
				t.Errorf("WindDirection = %v, want %v", stats.WindDirection, tt.wantWindDir)
			}

			if tt.wantWindGust != 0 && stats.WindGust != tt.wantWindGust {
				t.Errorf("WindGust = %v, want %v", stats.WindGust, tt.wantWindGust)
			}

			if tt.wantVisibility != 0 && stats.Visibility != tt.wantVisibility {
				t.Errorf("Visibility = %v, want %v", stats.Visibility, tt.wantVisibility)
			}

			if tt.wantCloud != "" && stats.Cloud != tt.wantCloud {
				t.Errorf("Cloud = %q, want %q", stats.Cloud, tt.wantCloud)
			}

			if tt.wantCondition != "" && !strings.Contains(stats.Condition, tt.wantCondition) {
				t.Errorf("Condition = %q, want to contain %q", stats.Condition, tt.wantCondition)
			}
		})
	}
}

func TestWeatherStats_GetField(t *testing.T) {
	stats := WeatherStats{
		StationID:     "KJFK",
		Temperature:   22,
		DewPoint:      10,
		Humidity:      45,
		Pressure:      1013.25,
		WindSpeed:     15,
		WindDirection: 270,
		WindGust:      25,
		Visibility:    10,
		Condition:     "clear",
		Cloud:         "few clouds",
		RawMETAR:      "KJFK 151756Z 27015G25KT 10SM FEW045 22/10 A2992",
	}

	tests := []struct {
		field string
		want  string
	}{
		{"temp", "22"},
		{"temperature", "22"},
		{"temp_f", "72"}, // 22°C = 71.6°F ≈ 72°F
		{"dewpoint", "10"},
		{"humidity", "45"},
		{"pressure", "1013"},
		{"pressure_inhg", "29.92"},
		{"wind", "15"},
		{"wind_speed", "15"},
		{"wind_dir", "270"},
		{"wind_direction", "270"},
		{"wind_dir_compass", "W"},
		{"wind_gust", "25"},
		{"visibility", "10.0"},
		{"condition", "clear"},
		{"weather", "clear"},
		{"cloud", "few clouds"},
		{"raw", "KJFK 151756Z 27015G25KT 10SM FEW045 22/10 A2992"},
		{"unknown_field", ""},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got := stats.GetField(tt.field)
			if got != tt.want {
				t.Errorf("GetField(%q) = %q, want %q", tt.field, got, tt.want)
			}
		})
	}
}

func TestDegreesToCompass(t *testing.T) {
	tests := []struct {
		degrees int
		want    string
	}{
		{0, "N"},
		{45, "NE"},
		{90, "E"},
		{135, "SE"},
		{180, "S"},
		{225, "SW"},
		{270, "W"},
		{315, "NW"},
		{360, "N"},
		{-1, "VRB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := degreesToCompass(tt.degrees)
			if got != tt.want {
				t.Errorf("degreesToCompass(%d) = %q, want %q", tt.degrees, got, tt.want)
			}
		})
	}
}

func TestWeatherReader_ReadWeather_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		_, _ = w.Write([]byte("KJFK 151756Z 31012KT 10SM FEW045 22/06 A3012"))
	}))
	defer server.Close()

	r := newWeatherReader()
	r.metarURL = server.URL + "?ids=%s"
	r.minInterval = 100 * time.Millisecond

	// First call should fetch
	stats1, err := r.ReadWeather("KJFK")
	if err != nil {
		t.Fatalf("First ReadWeather failed: %v", err)
	}
	if stats1.Temperature != 22 {
		t.Errorf("Temperature = %v, want 22", stats1.Temperature)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}

	// Second call should use cache
	stats2, err := r.ReadWeather("KJFK")
	if err != nil {
		t.Fatalf("Second ReadWeather failed: %v", err)
	}
	if stats2.Temperature != 22 {
		t.Errorf("Temperature = %v, want 22", stats2.Temperature)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d after cache hit, want 1", callCount)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call should fetch again
	_, err = r.ReadWeather("KJFK")
	if err != nil {
		t.Fatalf("Third ReadWeather failed: %v", err)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d after cache expiry, want 2", callCount)
	}
}

func TestWeatherReader_ReadWeather_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	r := newWeatherReader()
	r.metarURL = server.URL + "?ids=%s"

	_, err := r.ReadWeather("INVALID")
	if err == nil {
		t.Error("Expected error for HTTP 404")
	}
}

func TestWeatherReader_ReadWeather_EmptyStation(t *testing.T) {
	r := newWeatherReader()
	_, err := r.ReadWeather("")
	if err == nil {
		t.Error("Expected error for empty station ID")
	}
}

func TestWeatherReader_ClearCache(t *testing.T) {
	r := newWeatherReader()
	r.cache["KJFK"] = &weatherCacheEntry{
		stats:     WeatherStats{StationID: "KJFK"},
		fetchTime: time.Now(),
	}

	r.ClearCache()

	r.mu.RLock()
	_, ok := r.cache["KJFK"]
	r.mu.RUnlock()

	if ok {
		t.Error("Cache entry should be cleared")
	}
}

func TestCalculateHumidity(t *testing.T) {
	tests := []struct {
		temp     float64
		dewPoint float64
		wantMin  float64
		wantMax  float64
	}{
		{20, 20, 95, 100}, // Same temp and dew point = ~100% humidity
		{30, 15, 35, 45},  // Typical warm day
		{10, -5, 25, 40},  // Cold day with low dew point
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := calculateHumidity(tt.temp, tt.dewPoint)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("calculateHumidity(%v, %v) = %v, want between %v and %v",
					tt.temp, tt.dewPoint, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestParseWeatherCondition(t *testing.T) {
	tests := []struct {
		part     string
		existing string
		want     string
	}{
		{"RA", "", "rain"},
		{"+RA", "", "heavy rain"},
		{"-RA", "", "light rain"},
		{"TSRA", "", "thunderstorm with rain"},
		{"SN", "", "snow"},
		{"FG", "", "fog"},
		{"BR", "", "mist"},
		{"HZ", "", "haze"},
		{"RA", "fog", "fog, rain"},
		{"UNKNOWN", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.part, func(t *testing.T) {
			got := parseWeatherCondition(tt.part, tt.existing)
			if got != tt.want {
				t.Errorf("parseWeatherCondition(%q, %q) = %q, want %q",
					tt.part, tt.existing, got, tt.want)
			}
		})
	}
}
