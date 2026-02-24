// Package monitor provides system monitoring functionality.
// This file implements Music Player Daemon (MPD) client integration.
package monitor

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MPDState represents the current playback state of MPD.
type MPDState string

const (
	// MPDStatePlaying indicates the player is currently playing.
	MPDStatePlaying MPDState = "playing"
	// MPDStatePaused indicates the player is paused.
	MPDStatePaused MPDState = "paused"
	// MPDStateStopped indicates the player is stopped.
	MPDStateStopped MPDState = "stopped"
	// MPDStateUnknown indicates an unknown state.
	MPDStateUnknown MPDState = "unknown"
)

// MPDStats contains current MPD playback information.
type MPDStats struct {
	// State is the current playback state (playing/paused/stopped).
	State MPDState
	// Artist is the current track's artist.
	Artist string
	// Album is the current track's album.
	Album string
	// Title is the current track's title.
	Title string
	// Track is the current track number within the album.
	Track string
	// Name is the stream/radio name (for streams).
	Name string
	// File is the filename/URI of the current track.
	File string
	// Genre is the current track's genre.
	Genre string
	// Date is the release date/year.
	Date string
	// Elapsed is the elapsed time in seconds.
	Elapsed float64
	// Length is the total track length in seconds.
	Length float64
	// Bitrate is the current bitrate in kbps.
	Bitrate int
	// Volume is the current volume (0-100).
	Volume int
	// Repeat indicates if repeat mode is enabled.
	Repeat bool
	// Random indicates if random/shuffle mode is enabled.
	Random bool
	// Single indicates if single mode is enabled.
	Single bool
	// Consume indicates if consume mode is enabled.
	Consume bool
	// Connected indicates if we successfully connected to MPD.
	Connected bool
}

// mpdReader reads MPD status via TCP protocol.
type mpdReader struct {
	host      string
	port      int
	password  string
	timeout   time.Duration
	cache     *MPDStats
	cacheTime time.Time
	cacheTTL  time.Duration
	mu        sync.RWMutex
}

// newMPDReader creates a new mpdReader with default configuration.
// Default connects to localhost:6600 with 5 second timeout.
func newMPDReader() *mpdReader {
	return &mpdReader{
		host:     "localhost",
		port:     6600,
		timeout:  5 * time.Second,
		cacheTTL: 1 * time.Second,
	}
}

// SetHost sets the MPD server host.
func (r *mpdReader) SetHost(host string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.host = host
}

// SetPort sets the MPD server port.
func (r *mpdReader) SetPort(port int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.port = port
}

// SetPassword sets the MPD server password.
func (r *mpdReader) SetPassword(password string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.password = password
}

// ReadStats reads current MPD status.
// Results are cached for the cache TTL to reduce connection overhead.
func (r *mpdReader) ReadStats() (MPDStats, error) {
	r.mu.RLock()
	if r.cache != nil && time.Since(r.cacheTime) < r.cacheTTL {
		stats := *r.cache
		r.mu.RUnlock()
		return stats, nil
	}
	r.mu.RUnlock()

	stats, err := r.fetchStats()

	r.mu.Lock()
	r.cache = &stats
	r.cacheTime = time.Now()
	r.mu.Unlock()

	return stats, err
}

// fetchStats performs the actual MPD connection and status fetch.
func (r *mpdReader) fetchStats() (MPDStats, error) {
	r.mu.RLock()
	host := r.host
	port := r.port
	password := r.password
	timeout := r.timeout
	r.mu.RUnlock()

	stats := MPDStats{
		State: MPDStateStopped,
	}

	// Connect with timeout using interface types
	addr := fmt.Sprintf("%s:%d", host, port)
	var conn net.Conn
	var err error
	conn, err = net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return stats, fmt.Errorf("connect to MPD: %w", err)
	}
	defer conn.Close()

	// Set read/write deadline using Conn interface
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return stats, fmt.Errorf("set deadline: %w", err)
	}

	reader := bufio.NewReader(conn)

	// Read greeting line
	greeting, err := reader.ReadString('\n')
	if err != nil {
		return stats, fmt.Errorf("read greeting: %w", err)
	}
	if !strings.HasPrefix(greeting, "OK MPD") {
		return stats, fmt.Errorf("unexpected greeting: %s", greeting)
	}

	// Authenticate if password is set
	if password != "" {
		if _, err := fmt.Fprintf(conn, "password %s\n", password); err != nil {
			return stats, fmt.Errorf("send password: %w", err)
		}
		resp, err := reader.ReadString('\n')
		if err != nil {
			return stats, fmt.Errorf("read password response: %w", err)
		}
		if !strings.HasPrefix(resp, "OK") {
			return stats, fmt.Errorf("authentication failed: %s", strings.TrimSpace(resp))
		}
	}

	stats.Connected = true

	// Get status
	if _, err := fmt.Fprintf(conn, "status\n"); err != nil {
		return stats, fmt.Errorf("send status: %w", err)
	}
	statusData, err := r.readResponse(reader)
	if err != nil {
		return stats, fmt.Errorf("read status: %w", err)
	}
	r.parseStatus(statusData, &stats)

	// Get current song info
	if _, err := fmt.Fprintf(conn, "currentsong\n"); err != nil {
		return stats, fmt.Errorf("send currentsong: %w", err)
	}
	songData, err := r.readResponse(reader)
	if err != nil {
		return stats, fmt.Errorf("read currentsong: %w", err)
	}
	r.parseSong(songData, &stats)

	return stats, nil
}

// readResponse reads an MPD response until OK or ACK line.
func (r *mpdReader) readResponse(reader *bufio.Reader) (map[string]string, error) {
	data := make(map[string]string)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return data, err
		}
		line = strings.TrimSpace(line)

		if line == "OK" {
			break
		}
		if strings.HasPrefix(line, "ACK") {
			return data, fmt.Errorf("MPD error: %s", line)
		}

		// Parse key: value
		if idx := strings.Index(line, ": "); idx > 0 {
			key := strings.ToLower(line[:idx])
			value := line[idx+2:]
			data[key] = value
		}
	}

	return data, nil
}

// parseStatus parses the status command response.
func (r *mpdReader) parseStatus(data map[string]string, stats *MPDStats) {
	if state, ok := data["state"]; ok {
		switch state {
		case "play":
			stats.State = MPDStatePlaying
		case "pause":
			stats.State = MPDStatePaused
		case "stop":
			stats.State = MPDStateStopped
		default:
			stats.State = MPDStateUnknown
		}
	}

	if volume, ok := data["volume"]; ok {
		if v, err := strconv.Atoi(volume); err == nil {
			stats.Volume = v
		}
	}

	if repeat, ok := data["repeat"]; ok {
		stats.Repeat = repeat == "1"
	}

	if random, ok := data["random"]; ok {
		stats.Random = random == "1"
	}

	if single, ok := data["single"]; ok {
		stats.Single = single == "1"
	}

	if consume, ok := data["consume"]; ok {
		stats.Consume = consume == "1"
	}

	// Parse elapsed time (format: seconds.milliseconds or "elapsed:total")
	if elapsed, ok := data["elapsed"]; ok {
		if e, err := strconv.ParseFloat(elapsed, 64); err == nil {
			stats.Elapsed = e
		}
	} else if timeStr, ok := data["time"]; ok {
		// Older format: elapsed:total
		parts := strings.Split(timeStr, ":")
		if len(parts) >= 1 {
			if e, err := strconv.ParseFloat(parts[0], 64); err == nil {
				stats.Elapsed = e
			}
		}
		if len(parts) >= 2 {
			if l, err := strconv.ParseFloat(parts[1], 64); err == nil {
				stats.Length = l
			}
		}
	}

	// Parse duration (newer MPD versions)
	if duration, ok := data["duration"]; ok {
		if d, err := strconv.ParseFloat(duration, 64); err == nil {
			stats.Length = d
		}
	}

	if bitrate, ok := data["bitrate"]; ok {
		if b, err := strconv.Atoi(bitrate); err == nil {
			stats.Bitrate = b
		}
	}
}

// parseSong parses the currentsong command response.
func (r *mpdReader) parseSong(data map[string]string, stats *MPDStats) {
	if artist, ok := data["artist"]; ok {
		stats.Artist = artist
	}
	if album, ok := data["album"]; ok {
		stats.Album = album
	}
	if title, ok := data["title"]; ok {
		stats.Title = title
	}
	if track, ok := data["track"]; ok {
		stats.Track = track
	}
	if name, ok := data["name"]; ok {
		stats.Name = name
	}
	if file, ok := data["file"]; ok {
		stats.File = file
	}
	if genre, ok := data["genre"]; ok {
		stats.Genre = genre
	}
	if date, ok := data["date"]; ok {
		stats.Date = date
	}
	// Also check for duration in song data (some MPD versions)
	if duration, ok := data["time"]; ok {
		if d, err := strconv.ParseFloat(duration, 64); err == nil && stats.Length == 0 {
			stats.Length = d
		}
	}
}

// IsPlaying returns true if MPD is currently playing.
func (s MPDStats) IsPlaying() bool {
	return s.State == MPDStatePlaying
}

// IsPaused returns true if MPD is currently paused.
func (s MPDStats) IsPaused() bool {
	return s.State == MPDStatePaused
}

// IsStopped returns true if MPD is currently stopped.
func (s MPDStats) IsStopped() bool {
	return s.State == MPDStateStopped
}

// Percent returns the progress as a percentage (0-100).
func (s MPDStats) Percent() float64 {
	if s.Length <= 0 {
		return 0
	}
	pct := (s.Elapsed / s.Length) * 100
	if pct > 100 {
		pct = 100
	}
	return pct
}

// ElapsedTime returns the elapsed time formatted as MM:SS.
func (s MPDStats) ElapsedTime() string {
	return formatDuration(s.Elapsed)
}

// LengthTime returns the total length formatted as MM:SS.
func (s MPDStats) LengthTime() string {
	return formatDuration(s.Length)
}

// Smart returns a smart representation of the current track.
// Returns "Artist - Title" if both are available, otherwise falls back
// to Name (for streams) or filename.
func (s MPDStats) Smart() string {
	if s.Artist != "" && s.Title != "" {
		return s.Artist + " - " + s.Title
	}
	if s.Title != "" {
		return s.Title
	}
	if s.Name != "" {
		return s.Name
	}
	if s.File != "" {
		// Extract filename from path
		if idx := strings.LastIndex(s.File, "/"); idx >= 0 {
			return s.File[idx+1:]
		}
		return s.File
	}
	return ""
}

// formatDuration formats seconds as MM:SS or HH:MM:SS.
func formatDuration(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int(seconds)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
