package render

import (
	"image/color"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Width != 400 {
		t.Errorf("Width = %d, want 400", config.Width)
	}
	if config.Height != 300 {
		t.Errorf("Height = %d, want 300", config.Height)
	}
	if config.Title != "conky-go" {
		t.Errorf("Title = %q, want %q", config.Title, "conky-go")
	}
	if config.UpdateInterval != time.Second {
		t.Errorf("UpdateInterval = %v, want %v", config.UpdateInterval, time.Second)
	}
	if config.BackgroundColor.A != 200 {
		t.Errorf("BackgroundColor.A = %d, want 200", config.BackgroundColor.A)
	}
}

func TestTextLineStruct(t *testing.T) {
	line := TextLine{
		Text:  "Hello World",
		X:     10.5,
		Y:     20.5,
		Color: color.RGBA{R: 255, G: 128, B: 64, A: 255},
	}

	if line.Text != "Hello World" {
		t.Errorf("Text = %q, want %q", line.Text, "Hello World")
	}
	if line.X != 10.5 {
		t.Errorf("X = %v, want 10.5", line.X)
	}
	if line.Y != 20.5 {
		t.Errorf("Y = %v, want 20.5", line.Y)
	}
	if line.Color.R != 255 {
		t.Errorf("Color.R = %d, want 255", line.Color.R)
	}
}

func TestConfigCustomValues(t *testing.T) {
	config := Config{
		Width:           1920,
		Height:          1080,
		Title:           "Custom Title",
		UpdateInterval:  500 * time.Millisecond,
		BackgroundColor: color.RGBA{R: 50, G: 50, B: 50, A: 255},
	}

	if config.Width != 1920 {
		t.Errorf("Width = %d, want 1920", config.Width)
	}
	if config.Height != 1080 {
		t.Errorf("Height = %d, want 1080", config.Height)
	}
	if config.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", config.Title, "Custom Title")
	}
	if config.UpdateInterval != 500*time.Millisecond {
		t.Errorf("UpdateInterval = %v, want 500ms", config.UpdateInterval)
	}
}

func TestConfigZeroValues(t *testing.T) {
	config := Config{}

	// Verify zero values don't cause issues
	if config.Width != 0 {
		t.Errorf("Width = %d, want 0", config.Width)
	}
	if config.Height != 0 {
		t.Errorf("Height = %d, want 0", config.Height)
	}
	if config.Title != "" {
		t.Errorf("Title = %q, want empty", config.Title)
	}
}

func TestTextLineZeroValues(t *testing.T) {
	line := TextLine{}

	// Verify zero values don't cause issues
	if line.Text != "" {
		t.Errorf("Text = %q, want empty", line.Text)
	}
	if line.X != 0 {
		t.Errorf("X = %v, want 0", line.X)
	}
	if line.Y != 0 {
		t.Errorf("Y = %v, want 0", line.Y)
	}
}
