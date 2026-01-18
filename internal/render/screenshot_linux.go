//go:build linux

// Package render provides screen capture functionality for pseudo-transparency on Linux.
package render

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// X11ScreenshotProvider captures a region of the X11 root window.
// It connects to the X server and captures the specified rectangular region.
func X11ScreenshotProvider(x, y, width, height int) (*ebiten.Image, error) {
	// Connect to the X server
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to X server: %w", err)
	}
	defer conn.Close()

	// Get the setup information which includes the root window
	setup := xproto.Setup(conn)
	if len(setup.Roots) == 0 {
		return nil, fmt.Errorf("no screens found")
	}

	// Use the first (primary) screen
	screen := setup.Roots[0]
	rootWindow := screen.Root

	// Clamp coordinates to screen bounds
	screenWidth := int(screen.WidthInPixels)
	screenHeight := int(screen.HeightInPixels)

	// Adjust x and y if they're negative (shouldn't happen, but safety check)
	if x < 0 {
		width += x
		x = 0
	}
	if y < 0 {
		height += y
		y = 0
	}

	// Clamp width and height to screen bounds
	if x+width > screenWidth {
		width = screenWidth - x
	}
	if y+height > screenHeight {
		height = screenHeight - y
	}

	// Ensure valid dimensions
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid capture dimensions: %dx%d at (%d,%d)", width, height, x, y)
	}

	// Capture the region using GetImage
	reply, err := xproto.GetImage(
		conn,
		xproto.ImageFormatZPixmap,
		xproto.Drawable(rootWindow),
		int16(x), int16(y),
		uint16(width), uint16(height),
		0xFFFFFFFF, // All planes
	).Reply()
	if err != nil {
		return nil, fmt.Errorf("failed to capture screen region: %w", err)
	}

	// Convert the image data to an image.RGBA
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// The pixel data format depends on the visual depth
	// For most modern X11 servers, this is 24-bit or 32-bit color
	depth := int(reply.Depth)
	data := reply.Data

	switch depth {
	case 24, 32:
		// Data is in BGRX or BGRA format (4 bytes per pixel)
		for py := 0; py < height; py++ {
			for px := 0; px < width; px++ {
				idx := (py*width + px) * 4
				if idx+3 < len(data) {
					// X11 typically uses BGRX format
					b := data[idx]
					g := data[idx+1]
					r := data[idx+2]
					// Alpha is typically unused in root window captures
					img.SetRGBA(px, py, color.RGBA{R: r, G: g, B: b, A: 255})
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported color depth: %d", depth)
	}

	// Convert to Ebiten image
	ebitenImg := ebiten.NewImageFromImage(img)
	return ebitenImg, nil
}
