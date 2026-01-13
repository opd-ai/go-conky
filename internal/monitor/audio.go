package monitor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Package-level compiled regex patterns for performance.
var (
	// cardLineRegex matches card entries in /proc/asound/cards.
	// Example: " 0 [PCH            ]: HDA-Intel - HDA Intel PCH"
	cardLineRegex = regexp.MustCompile(`^\s*(\d+)\s+\[([^\]]+)\]:\s+(\S+)\s+-\s+(.+)$`)

	// ampRegex matches amp values in codec dumps.
	// Example: "[0x57 0x57]"
	ampRegex = regexp.MustCompile(`\[0x([0-9a-fA-F]+)\s+0x([0-9a-fA-F]+)\]`)

	// cardRegex matches card number in symlink targets.
	// Example: "card0"
	cardRegex = regexp.MustCompile(`card(\d+)`)
)

// AudioCard represents an audio hardware card.
type AudioCard struct {
	// Index is the card number (e.g., 0, 1).
	Index int
	// ID is the card identifier (e.g., "PCH", "HDMI").
	ID string
	// Name is the card name (e.g., "HDA Intel PCH").
	Name string
	// Driver is the ALSA driver module name (e.g., "HDA-Intel").
	Driver string
	// Mixers contains mixer control information keyed by control name.
	Mixers map[string]MixerInfo
}

// MixerInfo represents a mixer control (volume/mute).
type MixerInfo struct {
	// Name is the control name (e.g., "Master", "PCM", "Headphone").
	Name string
	// VolumePercent is the volume level as a percentage (0-100).
	VolumePercent float64
	// VolumeLeft is the left channel volume percentage (0-100).
	VolumeLeft float64
	// VolumeRight is the right channel volume percentage (0-100).
	VolumeRight float64
	// Muted indicates if the control is muted.
	Muted bool
	// HasVolume indicates if this control has volume capability.
	HasVolume bool
	// HasSwitch indicates if this control has mute/switch capability.
	HasSwitch bool
}

// AudioStats contains audio system statistics.
type AudioStats struct {
	// Cards contains audio card information keyed by card index.
	Cards map[int]AudioCard
	// DefaultCard is the index of the default audio card.
	DefaultCard int
	// MasterVolume is the master volume percentage (0-100) of the default card.
	MasterVolume float64
	// MasterMuted indicates if the master volume is muted.
	MasterMuted bool
	// HasAudio indicates if any audio hardware was detected.
	HasAudio bool
}

// audioReader reads audio information from /proc/asound.
type audioReader struct {
	asoundPath string
}

// newAudioReader creates a new audioReader with default paths.
func newAudioReader() *audioReader {
	return &audioReader{
		asoundPath: "/proc/asound",
	}
}

// ReadStats reads current audio system statistics.
func (r *audioReader) ReadStats() (AudioStats, error) {
	stats := AudioStats{
		Cards:       make(map[int]AudioCard),
		DefaultCard: -1,
	}

	// Check if asound directory exists
	if _, err := os.Stat(r.asoundPath); os.IsNotExist(err) {
		return stats, nil // No ALSA support, return empty stats
	}

	// Read card information from /proc/asound/cards
	cards, err := r.readCards()
	if err != nil {
		return stats, fmt.Errorf("reading cards: %w", err)
	}

	for idx, card := range cards {
		// Read mixer information for each card
		card.Mixers = r.readMixers(idx)
		stats.Cards[idx] = card
	}

	stats.HasAudio = len(stats.Cards) > 0

	// Set default card (first available)
	if stats.HasAudio {
		// Try to find default from /proc/asound/default, otherwise use first card
		stats.DefaultCard = r.findDefaultCard(stats.Cards)

		// Set master volume from default card
		if defaultCard, ok := stats.Cards[stats.DefaultCard]; ok {
			stats.MasterVolume, stats.MasterMuted = r.getMasterVolumeInfo(defaultCard)
		}
	}

	return stats, nil
}

// readCards reads card information from /proc/asound/cards.
func (r *audioReader) readCards() (map[int]AudioCard, error) {
	cards := make(map[int]AudioCard)

	cardsFile := filepath.Join(r.asoundPath, "cards")
	file, err := os.Open(cardsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return cards, nil
		}
		return nil, fmt.Errorf("opening cards file: %w", err)
	}
	defer file.Close()

	// Parse format:
	//  0 [PCH            ]: HDA-Intel - HDA Intel PCH
	//                       HDA Intel PCH at 0xf7610000 irq 30
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		matches := cardLineRegex.FindStringSubmatch(line)
		if matches != nil {
			idx, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}

			cards[idx] = AudioCard{
				Index:  idx,
				ID:     strings.TrimSpace(matches[2]),
				Driver: matches[3],
				Name:   strings.TrimSpace(matches[4]),
				Mixers: make(map[string]MixerInfo),
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return cards, fmt.Errorf("scanning cards file: %w", err)
	}

	return cards, nil
}

// readMixers reads mixer information for a card from /proc/asound/cardN.
func (r *audioReader) readMixers(cardIndex int) map[string]MixerInfo {
	mixers := make(map[string]MixerInfo)

	// Try to read from codec info for more detailed mixer info
	codecPath := filepath.Join(r.asoundPath, fmt.Sprintf("card%d", cardIndex), "codec#0")
	if codecInfo, err := os.ReadFile(codecPath); err == nil {
		mixers = r.parseMixerFromCodec(string(codecInfo))
	}

	// If no mixers found from codec, try amixer-style reading from /proc
	if len(mixers) == 0 {
		// Fallback: check for common mixer controls in card directory
		cardPath := filepath.Join(r.asoundPath, fmt.Sprintf("card%d", cardIndex))
		mixers = r.readMixerFromCard(cardPath)
	}

	return mixers
}

// parseMixerFromCodec parses mixer information from codec dump.
func (r *audioReader) parseMixerFromCodec(content string) map[string]MixerInfo {
	mixers := make(map[string]MixerInfo)

	// Look for Pin controls and amp settings in codec dump
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Parse amp-out/amp-in settings
		// Example: "Amp-Out vals:  [0x57 0x57]" (volume values)
		if !strings.Contains(line, "Amp-Out vals:") && !strings.Contains(line, "Amp-In vals:") {
			continue
		}

		matches := ampRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		leftHex, err := strconv.ParseInt(matches[1], 16, 64)
		if err != nil {
			continue
		}
		rightHex, err := strconv.ParseInt(matches[2], 16, 64)
		if err != nil {
			continue
		}

		// Amp values are typically 0-127 (7-bit), normalize to percentage
		// Mask the actual volume bits (lower 7 bits)
		leftVol := float64(leftHex&0x7F) / 127.0 * 100
		rightVol := float64(rightHex&0x7F) / 127.0 * 100

		// Check mute bit (bit 7)
		leftMuted := (leftHex & 0x80) != 0
		rightMuted := (rightHex & 0x80) != 0

		mixerName := "Master"
		if strings.Contains(line, "Amp-In") {
			mixerName = "Capture"
		}

		mixer := MixerInfo{
			Name:          mixerName,
			VolumeLeft:    leftVol,
			VolumeRight:   rightVol,
			VolumePercent: (leftVol + rightVol) / 2,
			Muted:         leftMuted && rightMuted,
			HasVolume:     true,
			HasSwitch:     true,
		}
		mixers[mixerName] = mixer
	}

	return mixers
}

// readMixerFromCard attempts to read mixer info from card directory.
func (r *audioReader) readMixerFromCard(cardPath string) map[string]MixerInfo {
	mixers := make(map[string]MixerInfo)

	// Read pcm info to check for active streams
	pcmPath := filepath.Join(cardPath, "pcm0p", "info")
	if _, err := os.Stat(pcmPath); err == nil {
		// Card has playback capability
		mixer := MixerInfo{
			Name:          "PCM",
			VolumePercent: 100, // Default to 100% when we can't read actual volume
			HasVolume:     true,
			HasSwitch:     false,
		}
		mixers["PCM"] = mixer
	}

	// Check for capture device
	capturePath := filepath.Join(cardPath, "pcm0c", "info")
	if _, err := os.Stat(capturePath); err == nil {
		mixer := MixerInfo{
			Name:          "Capture",
			VolumePercent: 100,
			HasVolume:     true,
			HasSwitch:     false,
		}
		mixers["Capture"] = mixer
	}

	return mixers
}

// findDefaultCard determines the default audio card.
func (r *audioReader) findDefaultCard(cards map[int]AudioCard) int {
	// Check for default card symlink
	defaultPath := filepath.Join(r.asoundPath, "default")
	if target, err := os.Readlink(defaultPath); err == nil {
		// Extract card number from target like "card0"
		if matches := cardRegex.FindStringSubmatch(target); matches != nil {
			if idx, err := strconv.Atoi(matches[1]); err == nil {
				if _, exists := cards[idx]; exists {
					return idx
				}
			}
		}
	}

	// Fallback to lowest numbered card
	minIdx := -1
	for idx := range cards {
		if minIdx == -1 || idx < minIdx {
			minIdx = idx
		}
	}
	return minIdx
}

// getMasterVolumeInfo extracts master volume information from a card.
func (r *audioReader) getMasterVolumeInfo(card AudioCard) (volume float64, muted bool) {
	// Priority order for master volume: Master > PCM > Front > Headphone
	priorityControls := []string{"Master", "PCM", "Front", "Headphone", "Speaker"}

	for _, name := range priorityControls {
		if mixer, ok := card.Mixers[name]; ok && mixer.HasVolume {
			return mixer.VolumePercent, mixer.Muted
		}
	}

	// If no priority control found, return first available
	for _, mixer := range card.Mixers {
		if mixer.HasVolume {
			return mixer.VolumePercent, mixer.Muted
		}
	}

	return 0, false
}
