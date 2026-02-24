package monitor

import (
	"testing"
	"time"
)

func TestMPDStatsHelpers(t *testing.T) {
	tests := []struct {
		name        string
		stats       MPDStats
		wantPlaying bool
		wantPaused  bool
		wantStopped bool
		wantPercent float64
		wantSmart   string
	}{
		{
			name: "playing state",
			stats: MPDStats{
				State:   MPDStatePlaying,
				Elapsed: 60,
				Length:  180,
				Artist:  "Test Artist",
				Title:   "Test Title",
			},
			wantPlaying: true,
			wantPaused:  false,
			wantStopped: false,
			wantPercent: 33.33,
			wantSmart:   "Test Artist - Test Title",
		},
		{
			name: "paused state",
			stats: MPDStats{
				State:   MPDStatePaused,
				Elapsed: 30,
				Length:  120,
				Title:   "Only Title",
			},
			wantPlaying: false,
			wantPaused:  true,
			wantStopped: false,
			wantPercent: 25,
			wantSmart:   "Only Title",
		},
		{
			name: "stopped state",
			stats: MPDStats{
				State: MPDStateStopped,
			},
			wantPlaying: false,
			wantPaused:  false,
			wantStopped: true,
			wantPercent: 0,
			wantSmart:   "",
		},
		{
			name: "stream with name",
			stats: MPDStats{
				State: MPDStatePlaying,
				Name:  "Radio Station",
			},
			wantPlaying: true,
			wantSmart:   "Radio Station",
		},
		{
			name: "file fallback",
			stats: MPDStats{
				State: MPDStatePlaying,
				File:  "/music/artist/album/song.mp3",
			},
			wantPlaying: true,
			wantSmart:   "song.mp3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.stats.IsPlaying(); got != tt.wantPlaying {
				t.Errorf("IsPlaying() = %v, want %v", got, tt.wantPlaying)
			}
			if got := tt.stats.IsPaused(); got != tt.wantPaused {
				t.Errorf("IsPaused() = %v, want %v", got, tt.wantPaused)
			}
			if got := tt.stats.IsStopped(); got != tt.wantStopped {
				t.Errorf("IsStopped() = %v, want %v", got, tt.wantStopped)
			}
			if tt.wantPercent > 0 {
				got := tt.stats.Percent()
				// Allow 0.5% tolerance for floating point
				if got < tt.wantPercent-0.5 || got > tt.wantPercent+0.5 {
					t.Errorf("Percent() = %v, want %v (±0.5)", got, tt.wantPercent)
				}
			}
			if tt.wantSmart != "" {
				if got := tt.stats.Smart(); got != tt.wantSmart {
					t.Errorf("Smart() = %v, want %v", got, tt.wantSmart)
				}
			}
		})
	}
}

func TestMPDStatsTimeFormat(t *testing.T) {
	tests := []struct {
		name     string
		elapsed  float64
		length   float64
		wantElap string
		wantLen  string
	}{
		{
			name:     "short track",
			elapsed:  65,
			length:   180,
			wantElap: "1:05",
			wantLen:  "3:00",
		},
		{
			name:     "long track",
			elapsed:  3665, // 1:01:05
			length:   7200, // 2:00:00
			wantElap: "1:01:05",
			wantLen:  "2:00:00",
		},
		{
			name:     "zero",
			elapsed:  0,
			length:   0,
			wantElap: "0:00",
			wantLen:  "0:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := MPDStats{
				Elapsed: tt.elapsed,
				Length:  tt.length,
			}
			if got := stats.ElapsedTime(); got != tt.wantElap {
				t.Errorf("ElapsedTime() = %v, want %v", got, tt.wantElap)
			}
			if got := stats.LengthTime(); got != tt.wantLen {
				t.Errorf("LengthTime() = %v, want %v", got, tt.wantLen)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds float64
		want    string
	}{
		{0, "0:00"},
		{59, "0:59"},
		{60, "1:00"},
		{65, "1:05"},
		{599, "9:59"},
		{600, "10:00"},
		{3599, "59:59"},
		{3600, "1:00:00"},
		{3665, "1:01:05"},
		{-5, "0:00"}, // Negative should be treated as 0
	}

	for _, tt := range tests {
		if got := formatDuration(tt.seconds); got != tt.want {
			t.Errorf("formatDuration(%v) = %v, want %v", tt.seconds, got, tt.want)
		}
	}
}

func TestMPDReaderCache(t *testing.T) {
	reader := newMPDReader()
	reader.cacheTTL = 100 * time.Millisecond

	// Initially no cache
	reader.mu.RLock()
	hasCache := reader.cache != nil
	reader.mu.RUnlock()
	if hasCache {
		t.Error("Expected no initial cache")
	}

	// Set a cache entry manually
	reader.mu.Lock()
	reader.cache = &MPDStats{
		State:  MPDStatePlaying,
		Artist: "Cached Artist",
	}
	reader.cacheTime = time.Now()
	reader.mu.Unlock()

	// Read should return cached data
	stats, _ := reader.ReadStats()
	if stats.Artist != "Cached Artist" {
		t.Errorf("Expected cached artist, got %v", stats.Artist)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Cache should be expired now - but ReadStats will fail to connect
	// and return the error. We just test that it tries to refresh.
	reader.mu.RLock()
	oldCacheTime := reader.cacheTime
	reader.mu.RUnlock()

	_, _ = reader.ReadStats() // Will fail to connect but should update cacheTime

	reader.mu.RLock()
	newCacheTime := reader.cacheTime
	reader.mu.RUnlock()

	if !newCacheTime.After(oldCacheTime) {
		t.Error("Expected cache time to be updated after TTL expiry")
	}
}

func TestMPDReaderParseStatus(t *testing.T) {
	reader := newMPDReader()

	tests := []struct {
		name       string
		statusData map[string]string
		wantState  MPDState
		wantVol    int
		wantRepeat bool
	}{
		{
			name: "playing",
			statusData: map[string]string{
				"state":  "play",
				"volume": "75",
				"repeat": "1",
			},
			wantState:  MPDStatePlaying,
			wantVol:    75,
			wantRepeat: true,
		},
		{
			name: "paused",
			statusData: map[string]string{
				"state":  "pause",
				"volume": "50",
				"repeat": "0",
			},
			wantState:  MPDStatePaused,
			wantVol:    50,
			wantRepeat: false,
		},
		{
			name: "stopped",
			statusData: map[string]string{
				"state": "stop",
			},
			wantState: MPDStateStopped,
			wantVol:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := MPDStats{}
			reader.parseStatus(tt.statusData, &stats)

			if stats.State != tt.wantState {
				t.Errorf("State = %v, want %v", stats.State, tt.wantState)
			}
			if stats.Volume != tt.wantVol {
				t.Errorf("Volume = %v, want %v", stats.Volume, tt.wantVol)
			}
			if stats.Repeat != tt.wantRepeat {
				t.Errorf("Repeat = %v, want %v", stats.Repeat, tt.wantRepeat)
			}
		})
	}
}

func TestMPDReaderParseSong(t *testing.T) {
	reader := newMPDReader()

	songData := map[string]string{
		"artist": "Test Artist",
		"album":  "Test Album",
		"title":  "Test Title",
		"track":  "5",
		"file":   "/music/test.mp3",
		"genre":  "Rock",
		"date":   "2024",
	}

	stats := MPDStats{}
	reader.parseSong(songData, &stats)

	if stats.Artist != "Test Artist" {
		t.Errorf("Artist = %v, want Test Artist", stats.Artist)
	}
	if stats.Album != "Test Album" {
		t.Errorf("Album = %v, want Test Album", stats.Album)
	}
	if stats.Title != "Test Title" {
		t.Errorf("Title = %v, want Test Title", stats.Title)
	}
	if stats.Track != "5" {
		t.Errorf("Track = %v, want 5", stats.Track)
	}
	if stats.File != "/music/test.mp3" {
		t.Errorf("File = %v, want /music/test.mp3", stats.File)
	}
	if stats.Genre != "Rock" {
		t.Errorf("Genre = %v, want Rock", stats.Genre)
	}
	if stats.Date != "2024" {
		t.Errorf("Date = %v, want 2024", stats.Date)
	}
}

func TestMPDReaderSetters(t *testing.T) {
	reader := newMPDReader()

	// Test SetHost
	reader.SetHost("192.168.1.100")
	reader.mu.RLock()
	if reader.host != "192.168.1.100" {
		t.Errorf("host = %v, want 192.168.1.100", reader.host)
	}
	reader.mu.RUnlock()

	// Test SetPort
	reader.SetPort(6601)
	reader.mu.RLock()
	if reader.port != 6601 {
		t.Errorf("port = %v, want 6601", reader.port)
	}
	reader.mu.RUnlock()

	// Test SetPassword
	reader.SetPassword("secret")
	reader.mu.RLock()
	if reader.password != "secret" {
		t.Errorf("password = %v, want secret", reader.password)
	}
	reader.mu.RUnlock()
}
