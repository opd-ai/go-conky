//go:build !noebiten

package render

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFontStyleString(t *testing.T) {
	tests := []struct {
		style FontStyle
		want  string
	}{
		{FontStyleRegular, "regular"},
		{FontStyleBold, "bold"},
		{FontStyleItalic, "italic"},
		{FontStyleBoldItalic, "bold-italic"},
		{FontStyle(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.style.String(); got != tt.want {
				t.Errorf("FontStyle.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFontStyle(t *testing.T) {
	tests := []struct {
		input   string
		want    FontStyle
		wantErr bool
	}{
		{"regular", FontStyleRegular, false},
		{"normal", FontStyleRegular, false},
		{"", FontStyleRegular, false},
		{"bold", FontStyleBold, false},
		{"italic", FontStyleItalic, false},
		{"bold-italic", FontStyleBoldItalic, false},
		{"bolditalic", FontStyleBoldItalic, false},
		{"bold_italic", FontStyleBoldItalic, false},
		{"invalid", FontStyleRegular, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFontStyle(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFontStyle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFontStyle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFontFamily(t *testing.T) {
	ff := NewFontFamily("TestFamily")
	if ff == nil {
		t.Fatal("NewFontFamily() returned nil")
	}
	if ff.Name() != "TestFamily" {
		t.Errorf("Name() = %v, want %v", ff.Name(), "TestFamily")
	}
}

func TestFontFamilyAddAndGetFont(t *testing.T) {
	fm := NewFontManager()
	ff := fm.GetFamily("GoMono")
	if ff == nil {
		t.Fatal("GoMono family not found")
	}

	// Test getting fonts for each style
	regular := ff.GetFont(FontStyleRegular)
	if regular == nil {
		t.Error("Regular font should be available")
	}

	bold := ff.GetFont(FontStyleBold)
	if bold == nil {
		t.Error("Bold font should be available")
	}

	italic := ff.GetFont(FontStyleItalic)
	if italic == nil {
		t.Error("Italic font should be available")
	}

	boldItalic := ff.GetFont(FontStyleBoldItalic)
	if boldItalic == nil {
		t.Error("BoldItalic font should be available")
	}
}

func TestFontFamilyGetFontWithFallback(t *testing.T) {
	ff := NewFontFamily("TestFamily")
	fm := NewFontManager()
	goMono := fm.GetFamily("GoMono")

	// Add only regular font
	ff.AddFont(FontStyleRegular, goMono.GetFont(FontStyleRegular))

	// Request bold should fallback to regular
	font := ff.GetFontWithFallback(FontStyleBold)
	if font == nil {
		t.Error("Should fallback to regular when bold not available")
	}

	// Request italic should fallback to regular
	font = ff.GetFontWithFallback(FontStyleItalic)
	if font == nil {
		t.Error("Should fallback to regular when italic not available")
	}

	// Request bold-italic should fallback to regular
	font = ff.GetFontWithFallback(FontStyleBoldItalic)
	if font == nil {
		t.Error("Should fallback to regular when bold-italic not available")
	}
}

func TestFontFamilyHasStyle(t *testing.T) {
	fm := NewFontManager()
	ff := fm.GetFamily("GoMono")
	if ff == nil {
		t.Fatal("GoMono family not found")
	}

	if !ff.HasStyle(FontStyleRegular) {
		t.Error("Should have regular style")
	}
	if !ff.HasStyle(FontStyleBold) {
		t.Error("Should have bold style")
	}
	if !ff.HasStyle(FontStyleItalic) {
		t.Error("Should have italic style")
	}
	if !ff.HasStyle(FontStyleBoldItalic) {
		t.Error("Should have bold-italic style")
	}
}

func TestFontFamilyAvailableStyles(t *testing.T) {
	fm := NewFontManager()
	ff := fm.GetFamily("GoMono")
	if ff == nil {
		t.Fatal("GoMono family not found")
	}

	styles := ff.AvailableStyles()
	if len(styles) != 4 {
		t.Errorf("Expected 4 styles, got %d", len(styles))
	}
}

func TestNewFontManager(t *testing.T) {
	fm := NewFontManager()
	if fm == nil {
		t.Fatal("NewFontManager() returned nil")
	}

	// Check embedded fonts are loaded
	if fm.GetFamily("GoMono") == nil {
		t.Error("GoMono family should be available")
	}
	if fm.GetFamily("GoSans") == nil {
		t.Error("GoSans family should be available")
	}

	// Check aliases work
	if fm.GetFamily("gomono") == nil {
		t.Error("gomono alias should work")
	}
	if fm.GetFamily("gosans") == nil {
		t.Error("gosans alias should work")
	}
}

func TestFontManagerGetFont(t *testing.T) {
	fm := NewFontManager()

	// Get existing font
	font := fm.GetFont("GoMono", FontStyleRegular)
	if font == nil {
		t.Error("Should get GoMono regular font")
	}

	// Get non-existent family
	font = fm.GetFont("NonExistent", FontStyleRegular)
	if font != nil {
		t.Error("Should return nil for non-existent family")
	}
}

func TestFontManagerGetFontWithFallback(t *testing.T) {
	fm := NewFontManager()

	// Test with existing family
	font := fm.GetFontWithFallback("GoMono", FontStyleBold)
	if font == nil {
		t.Error("Should get font with fallback for existing family")
	}

	// Test with non-existent family should fall back to chain
	font = fm.GetFontWithFallback("NonExistent", FontStyleRegular)
	if font == nil {
		t.Error("Should get font from fallback chain")
	}
}

func TestFontManagerSetFallbackChain(t *testing.T) {
	fm := NewFontManager()

	newChain := []string{"GoSans", "GoMono"}
	fm.SetFallbackChain(newChain)

	chain := fm.FallbackChain()
	if len(chain) != 2 {
		t.Errorf("Expected 2 families in chain, got %d", len(chain))
	}
	if chain[0] != "GoSans" {
		t.Errorf("First family should be GoSans, got %s", chain[0])
	}
}

func TestFontManagerDefaultFamily(t *testing.T) {
	fm := NewFontManager()

	if fm.DefaultFamily() != "GoMono" {
		t.Errorf("Default family should be GoMono, got %s", fm.DefaultFamily())
	}

	fm.SetDefaultFamily("GoSans")
	if fm.DefaultFamily() != "GoSans" {
		t.Errorf("Default family should be GoSans, got %s", fm.DefaultFamily())
	}
}

func TestFontManagerListFamilies(t *testing.T) {
	fm := NewFontManager()
	families := fm.ListFamilies()

	// Should have exactly GoMono and GoSans (canonical names only, no aliases)
	if len(families) != 2 {
		t.Errorf("Expected 2 families, got %d: %v", len(families), families)
	}

	// Verify the result is sorted
	if len(families) >= 2 && families[0] > families[1] {
		t.Errorf("Expected sorted families, got %v", families)
	}

	// Verify canonical names are returned
	expectedFamilies := map[string]bool{"GoMono": true, "GoSans": true}
	for _, name := range families {
		if !expectedFamilies[name] {
			t.Errorf("Unexpected family name: %s (expected canonical names GoMono or GoSans)", name)
		}
	}
}

func TestFontManagerRegisterAlias(t *testing.T) {
	fm := NewFontManager()

	err := fm.RegisterAlias("Mono", "GoMono")
	if err != nil {
		t.Errorf("RegisterAlias failed: %v", err)
	}

	// Alias should work
	if fm.GetFamily("Mono") == nil {
		t.Error("Alias 'Mono' should work")
	}

	// Non-existent family should fail
	err = fm.RegisterAlias("Test", "NonExistent")
	if err == nil {
		t.Error("RegisterAlias should fail for non-existent family")
	}
}

func TestFontManagerLoadFontFromData(t *testing.T) {
	fm := NewFontManager()

	// Invalid font data should fail
	err := fm.LoadFontFromData("TestFamily", FontStyleRegular, []byte("invalid"))
	if err == nil {
		t.Error("LoadFontFromData should fail with invalid data")
	}
}

func TestFontManagerLoadFontFromFile(t *testing.T) {
	fm := NewFontManager()

	// Non-existent file should fail
	err := fm.LoadFontFromFile("TestFamily", FontStyleRegular, "/nonexistent/font.ttf")
	if err == nil {
		t.Error("LoadFontFromFile should fail for non-existent file")
	}

	// Create a temp file with invalid content
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.ttf")
	writeErr := os.WriteFile(tmpFile, []byte("not a font"), 0o644)
	if writeErr != nil {
		t.Fatalf("Failed to create temp file: %v", writeErr)
	}

	err = fm.LoadFontFromFile("TestFamily", FontStyleRegular, tmpFile)
	if err == nil {
		t.Error("LoadFontFromFile should fail with invalid font file")
	}
}

func TestFontManagerConcurrentAccess(t *testing.T) {
	fm := NewFontManager()
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			_ = fm.GetFont("GoMono", FontStyleRegular)
			_ = fm.GetFontWithFallback("GoMono", FontStyleBold)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = fm.ListFamilies()
			_ = fm.FallbackChain()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			fm.SetDefaultFamily("GoMono")
			fm.SetFallbackChain([]string{"GoMono", "GoSans"})
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}

func TestFontFamilyConcurrentAccess(t *testing.T) {
	fm := NewFontManager()
	ff := fm.GetFamily("GoMono")
	if ff == nil {
		t.Fatal("GoMono family not found")
	}

	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			_ = ff.GetFont(FontStyleRegular)
			_ = ff.GetFontWithFallback(FontStyleBold)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = ff.HasStyle(FontStyleBold)
			_ = ff.AvailableStyles()
		}
		done <- true
	}()

	<-done
	<-done
}

func TestFontFamilyEmptyFallback(t *testing.T) {
	ff := NewFontFamily("Empty")

	// Empty family should return nil
	if ff.GetFontWithFallback(FontStyleRegular) != nil {
		t.Error("Empty family should return nil")
	}
}
