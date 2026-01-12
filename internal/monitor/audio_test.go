package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAudioReader(t *testing.T) {
	reader := newAudioReader()
	if reader == nil {
		t.Fatal("newAudioReader() returned nil")
	}
	if reader.asoundPath != "/proc/asound" {
		t.Errorf("asoundPath = %q, want %q", reader.asoundPath, "/proc/asound")
	}
}

func TestAudioReaderMissingDirectory(t *testing.T) {
	reader := &audioReader{
		asoundPath: "/nonexistent/asound",
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Errorf("ReadStats() error = %v, want nil for missing directory", err)
	}
	if len(stats.Cards) != 0 {
		t.Errorf("Cards count = %d, want 0 for missing directory", len(stats.Cards))
	}
	if stats.HasAudio {
		t.Error("HasAudio = true, want false for missing directory")
	}
}

func TestAudioReaderWithMockCards(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock /proc/asound/cards file
	cardsContent := ` 0 [PCH            ]: HDA-Intel - HDA Intel PCH
                      HDA Intel PCH at 0xf7610000 irq 30
 1 [HDMI           ]: HDA-Intel - HDA NVidia
                      HDA NVidia at 0xf7080000 irq 31
`
	if err := os.WriteFile(filepath.Join(tmpDir, "cards"), []byte(cardsContent), 0o644); err != nil {
		t.Fatalf("failed to write cards file: %v", err)
	}

	// Create card directories
	card0 := filepath.Join(tmpDir, "card0")
	card1 := filepath.Join(tmpDir, "card1")
	if err := os.MkdirAll(card0, 0o755); err != nil {
		t.Fatalf("failed to create card0 directory: %v", err)
	}
	if err := os.MkdirAll(card1, 0o755); err != nil {
		t.Fatalf("failed to create card1 directory: %v", err)
	}

	reader := &audioReader{
		asoundPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Cards) != 2 {
		t.Errorf("Cards count = %d, want 2", len(stats.Cards))
	}

	// Verify card 0
	card0Info, ok := stats.Cards[0]
	if !ok {
		t.Fatal("Card 0 not found")
	}
	if card0Info.ID != "PCH" {
		t.Errorf("Card 0 ID = %q, want %q", card0Info.ID, "PCH")
	}
	if card0Info.Name != "HDA Intel PCH" {
		t.Errorf("Card 0 Name = %q, want %q", card0Info.Name, "HDA Intel PCH")
	}
	if card0Info.Driver != "HDA-Intel" {
		t.Errorf("Card 0 Driver = %q, want %q", card0Info.Driver, "HDA-Intel")
	}

	// Verify card 1
	card1Info, ok := stats.Cards[1]
	if !ok {
		t.Fatal("Card 1 not found")
	}
	if card1Info.ID != "HDMI" {
		t.Errorf("Card 1 ID = %q, want %q", card1Info.ID, "HDMI")
	}
	if card1Info.Name != "HDA NVidia" {
		t.Errorf("Card 1 Name = %q, want %q", card1Info.Name, "HDA NVidia")
	}

	if !stats.HasAudio {
		t.Error("HasAudio = false, want true")
	}

	// Default card should be 0 (lowest index)
	if stats.DefaultCard != 0 {
		t.Errorf("DefaultCard = %d, want 0", stats.DefaultCard)
	}
}

func TestAudioReaderEmptyCardsFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty cards file
	if err := os.WriteFile(filepath.Join(tmpDir, "cards"), []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write cards file: %v", err)
	}

	reader := &audioReader{
		asoundPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Cards) != 0 {
		t.Errorf("Cards count = %d, want 0", len(stats.Cards))
	}
	if stats.HasAudio {
		t.Error("HasAudio = true, want false")
	}
	if stats.DefaultCard != -1 {
		t.Errorf("DefaultCard = %d, want -1", stats.DefaultCard)
	}
}

func TestAudioReaderWithPCMDevice(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock cards file
	cardsContent := ` 0 [PCH            ]: HDA-Intel - HDA Intel PCH
                      HDA Intel PCH at 0xf7610000 irq 30
`
	if err := os.WriteFile(filepath.Join(tmpDir, "cards"), []byte(cardsContent), 0o644); err != nil {
		t.Fatalf("failed to write cards file: %v", err)
	}

	// Create card0 directory with PCM device
	card0 := filepath.Join(tmpDir, "card0")
	pcm0p := filepath.Join(card0, "pcm0p")
	if err := os.MkdirAll(pcm0p, 0o755); err != nil {
		t.Fatalf("failed to create pcm0p directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pcm0p, "info"), []byte("card: 0\ndevice: 0\n"), 0o644); err != nil {
		t.Fatalf("failed to write pcm info: %v", err)
	}

	reader := &audioReader{
		asoundPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	card := stats.Cards[0]
	if _, ok := card.Mixers["PCM"]; !ok {
		t.Error("PCM mixer not found")
	}
}

func TestAudioReaderWithCaptureDevice(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock cards file
	cardsContent := ` 0 [PCH            ]: HDA-Intel - HDA Intel PCH
                      HDA Intel PCH at 0xf7610000 irq 30
`
	if err := os.WriteFile(filepath.Join(tmpDir, "cards"), []byte(cardsContent), 0o644); err != nil {
		t.Fatalf("failed to write cards file: %v", err)
	}

	// Create card0 directory with capture device
	card0 := filepath.Join(tmpDir, "card0")
	pcm0c := filepath.Join(card0, "pcm0c")
	if err := os.MkdirAll(pcm0c, 0o755); err != nil {
		t.Fatalf("failed to create pcm0c directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pcm0c, "info"), []byte("card: 0\ndevice: 0\n"), 0o644); err != nil {
		t.Fatalf("failed to write pcm info: %v", err)
	}

	reader := &audioReader{
		asoundPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	card := stats.Cards[0]
	if _, ok := card.Mixers["Capture"]; !ok {
		t.Error("Capture mixer not found")
	}
}

func TestAudioReaderWithCodecInfo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock cards file
	cardsContent := ` 0 [PCH            ]: HDA-Intel - HDA Intel PCH
                      HDA Intel PCH at 0xf7610000 irq 30
`
	if err := os.WriteFile(filepath.Join(tmpDir, "cards"), []byte(cardsContent), 0o644); err != nil {
		t.Fatalf("failed to write cards file: %v", err)
	}

	// Create card0 directory with codec info
	card0 := filepath.Join(tmpDir, "card0")
	if err := os.MkdirAll(card0, 0o755); err != nil {
		t.Fatalf("failed to create card0 directory: %v", err)
	}

	// Mock codec dump with amp values
	// 0x57 = 87 decimal, which is about 68.5% volume (87/127 * 100)
	// Lower 7 bits for volume, bit 7 for mute
	codecContent := `Codec: Realtek ALC892
Address: 0
AFG Function Id: 0x1 (unsol 1)
Vendor Id: 0x10ec0892
Subsystem Id: 0x10438436
Revision Id: 0x100302
  Amp-Out vals:  [0x57 0x57]
  Amp-In vals:   [0x80 0x80]
`
	if err := os.WriteFile(filepath.Join(card0, "codec#0"), []byte(codecContent), 0o644); err != nil {
		t.Fatalf("failed to write codec info: %v", err)
	}

	reader := &audioReader{
		asoundPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	card := stats.Cards[0]

	// Check Master mixer (from Amp-Out)
	master, ok := card.Mixers["Master"]
	if !ok {
		t.Fatal("Master mixer not found")
	}
	if !master.HasVolume {
		t.Error("Master HasVolume = false, want true")
	}
	// Volume should be around 68.5% (87/127 * 100)
	expectedVol := float64(0x57&0x7F) / 127.0 * 100
	if master.VolumePercent < expectedVol-1 || master.VolumePercent > expectedVol+1 {
		t.Errorf("Master VolumePercent = %f, want approximately %f", master.VolumePercent, expectedVol)
	}
	if master.Muted {
		t.Error("Master Muted = true, want false (bit 7 not set)")
	}

	// Check Capture mixer (from Amp-In with mute bit set)
	capture, ok := card.Mixers["Capture"]
	if !ok {
		t.Fatal("Capture mixer not found")
	}
	if !capture.Muted {
		t.Error("Capture Muted = false, want true (bit 7 set in 0x80)")
	}
}

func TestAudioReaderDefaultCard(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock cards file
	cardsContent := ` 1 [HDMI           ]: HDA-Intel - HDA NVidia
                      HDA NVidia at 0xf7080000 irq 31
 2 [PCH            ]: HDA-Intel - HDA Intel PCH
                      HDA Intel PCH at 0xf7610000 irq 30
`
	if err := os.WriteFile(filepath.Join(tmpDir, "cards"), []byte(cardsContent), 0o644); err != nil {
		t.Fatalf("failed to write cards file: %v", err)
	}

	// Create card directories
	if err := os.MkdirAll(filepath.Join(tmpDir, "card1"), 0o755); err != nil {
		t.Fatalf("failed to create card1 directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "card2"), 0o755); err != nil {
		t.Fatalf("failed to create card2 directory: %v", err)
	}

	reader := &audioReader{
		asoundPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// Default should be lowest index (1, not 2)
	if stats.DefaultCard != 1 {
		t.Errorf("DefaultCard = %d, want 1", stats.DefaultCard)
	}
}

func TestAudioReaderWithDefaultSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock cards file
	cardsContent := ` 0 [HDMI           ]: HDA-Intel - HDA NVidia
                      HDA NVidia at 0xf7080000 irq 31
 1 [PCH            ]: HDA-Intel - HDA Intel PCH
                      HDA Intel PCH at 0xf7610000 irq 30
`
	if err := os.WriteFile(filepath.Join(tmpDir, "cards"), []byte(cardsContent), 0o644); err != nil {
		t.Fatalf("failed to write cards file: %v", err)
	}

	// Create card directories
	if err := os.MkdirAll(filepath.Join(tmpDir, "card0"), 0o755); err != nil {
		t.Fatalf("failed to create card0 directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "card1"), 0o755); err != nil {
		t.Fatalf("failed to create card1 directory: %v", err)
	}

	// Create default symlink to card1
	if err := os.Symlink("card1", filepath.Join(tmpDir, "default")); err != nil {
		t.Fatalf("failed to create default symlink: %v", err)
	}

	reader := &audioReader{
		asoundPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// Default should follow symlink to card1
	if stats.DefaultCard != 1 {
		t.Errorf("DefaultCard = %d, want 1 (from symlink)", stats.DefaultCard)
	}
}

func TestAudioReaderMasterVolumeExtraction(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock cards file
	cardsContent := ` 0 [PCH            ]: HDA-Intel - HDA Intel PCH
                      HDA Intel PCH at 0xf7610000 irq 30
`
	if err := os.WriteFile(filepath.Join(tmpDir, "cards"), []byte(cardsContent), 0o644); err != nil {
		t.Fatalf("failed to write cards file: %v", err)
	}

	// Create card0 with codec showing master volume
	card0 := filepath.Join(tmpDir, "card0")
	if err := os.MkdirAll(card0, 0o755); err != nil {
		t.Fatalf("failed to create card0 directory: %v", err)
	}

	// 0x3F = 63 decimal = ~49.6% volume
	codecContent := `Codec: Test
  Amp-Out vals:  [0x3f 0x3f]
`
	if err := os.WriteFile(filepath.Join(card0, "codec#0"), []byte(codecContent), 0o644); err != nil {
		t.Fatalf("failed to write codec info: %v", err)
	}

	reader := &audioReader{
		asoundPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	expectedVol := float64(0x3F) / 127.0 * 100
	if stats.MasterVolume < expectedVol-1 || stats.MasterVolume > expectedVol+1 {
		t.Errorf("MasterVolume = %f, want approximately %f", stats.MasterVolume, expectedVol)
	}
	if stats.MasterMuted {
		t.Error("MasterMuted = true, want false")
	}
}

func TestAudioCardStruct(t *testing.T) {
	card := AudioCard{
		Index:  0,
		ID:     "PCH",
		Name:   "HDA Intel PCH",
		Driver: "HDA-Intel",
		Mixers: make(map[string]MixerInfo),
	}

	card.Mixers["Master"] = MixerInfo{
		Name:          "Master",
		VolumePercent: 75,
		HasVolume:     true,
	}

	if card.Index != 0 {
		t.Errorf("Index = %d, want 0", card.Index)
	}
	if card.ID != "PCH" {
		t.Errorf("ID = %q, want %q", card.ID, "PCH")
	}
	if len(card.Mixers) != 1 {
		t.Errorf("Mixers count = %d, want 1", len(card.Mixers))
	}
}

func TestMixerInfoStruct(t *testing.T) {
	mixer := MixerInfo{
		Name:          "Master",
		VolumePercent: 75,
		VolumeLeft:    75,
		VolumeRight:   75,
		Muted:         false,
		HasVolume:     true,
		HasSwitch:     true,
	}

	if mixer.Name != "Master" {
		t.Errorf("Name = %q, want %q", mixer.Name, "Master")
	}
	if mixer.VolumePercent != 75 {
		t.Errorf("VolumePercent = %f, want 75", mixer.VolumePercent)
	}
	if mixer.Muted {
		t.Error("Muted = true, want false")
	}
	if !mixer.HasVolume {
		t.Error("HasVolume = false, want true")
	}
}

func TestAudioStatsStruct(t *testing.T) {
	stats := AudioStats{
		Cards:        make(map[int]AudioCard),
		DefaultCard:  0,
		MasterVolume: 80,
		MasterMuted:  false,
		HasAudio:     true,
	}

	stats.Cards[0] = AudioCard{
		Index: 0,
		ID:    "PCH",
		Name:  "HDA Intel PCH",
	}

	if len(stats.Cards) != 1 {
		t.Errorf("Cards count = %d, want 1", len(stats.Cards))
	}
	if stats.DefaultCard != 0 {
		t.Errorf("DefaultCard = %d, want 0", stats.DefaultCard)
	}
	if stats.MasterVolume != 80 {
		t.Errorf("MasterVolume = %f, want 80", stats.MasterVolume)
	}
	if !stats.HasAudio {
		t.Error("HasAudio = false, want true")
	}
}

func TestSystemDataAudioGetSet(t *testing.T) {
	sd := NewSystemData()

	audio := AudioStats{
		Cards:        make(map[int]AudioCard),
		DefaultCard:  0,
		MasterVolume: 75,
		MasterMuted:  false,
		HasAudio:     true,
	}
	audio.Cards[0] = AudioCard{
		Index:  0,
		ID:     "PCH",
		Name:   "HDA Intel PCH",
		Mixers: make(map[string]MixerInfo),
	}
	audio.Cards[0].Mixers["Master"] = MixerInfo{Name: "Master", VolumePercent: 75}

	sd.setAudio(audio)

	got := sd.GetAudio()
	if got.MasterVolume != 75 {
		t.Errorf("MasterVolume = %f, want 75", got.MasterVolume)
	}
	if !got.HasAudio {
		t.Error("HasAudio = false, want true")
	}
	if len(got.Cards) != 1 {
		t.Errorf("Cards count = %d, want 1", len(got.Cards))
	}

	// Verify deep copy - modifying returned value shouldn't affect stored value
	got.Cards[1] = AudioCard{Index: 1}
	stored := sd.GetAudio()
	if len(stored.Cards) != 1 {
		t.Errorf("Stored Cards count = %d after modification, want 1", len(stored.Cards))
	}
}

func TestAudioReaderMalformedCardsFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create malformed cards file
	cardsContent := `This is not a valid cards format
Some random text
123 abc def
`
	if err := os.WriteFile(filepath.Join(tmpDir, "cards"), []byte(cardsContent), 0o644); err != nil {
		t.Fatalf("failed to write cards file: %v", err)
	}

	reader := &audioReader{
		asoundPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// Malformed lines should be skipped
	if len(stats.Cards) != 0 {
		t.Errorf("Cards count = %d, want 0 for malformed file", len(stats.Cards))
	}
}

func TestAudioReaderEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	reader := &audioReader{
		asoundPath: tmpDir,
	}

	// Directory exists but no cards file
	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Cards) != 0 {
		t.Errorf("Cards count = %d, want 0", len(stats.Cards))
	}
	if stats.HasAudio {
		t.Error("HasAudio = true, want false")
	}
}
