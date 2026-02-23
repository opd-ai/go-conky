// Package render provides background rendering capabilities for conky-go.
package render

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// BackgroundMode specifies how the window background is rendered.
type BackgroundMode int

const (
	// BackgroundModeSolid draws a solid background color.
	BackgroundModeSolid BackgroundMode = iota
	// BackgroundModeNone draws no background (fully transparent).
	BackgroundModeNone
	// BackgroundModeGradient draws a gradient background.
	BackgroundModeGradient
	// BackgroundModePseudo uses a cached screenshot of the desktop as the background.
	BackgroundModePseudo
)

// GradientDirection specifies the direction of a gradient.
type GradientDirection int

const (
	// GradientDirectionVertical renders gradient from top to bottom.
	GradientDirectionVertical GradientDirection = iota
	// GradientDirectionHorizontal renders gradient from left to right.
	GradientDirectionHorizontal
	// GradientDirectionDiagonal renders gradient from top-left to bottom-right.
	GradientDirectionDiagonal
	// GradientDirectionRadial renders gradient from center outward.
	GradientDirectionRadial
)

// BackgroundRenderer is an interface for rendering window backgrounds.
// It allows for extensible background rendering strategies.
type BackgroundRenderer interface {
	// Draw renders the background to the screen.
	Draw(screen *ebiten.Image)
	// Mode returns the background mode.
	Mode() BackgroundMode
}

// SolidBackground renders a solid color background.
type SolidBackground struct {
	color     color.RGBA
	argbValue int  // Alpha value override (0-255) when ARGB is enabled
	argbOn    bool // Whether ARGB visual is enabled
}

// NewSolidBackground creates a new solid background renderer.
func NewSolidBackground(c color.RGBA) *SolidBackground {
	return &SolidBackground{
		color:     c,
		argbValue: 255,
		argbOn:    false,
	}
}

// WithARGB configures ARGB visual settings for the background.
// When enabled, the argbValue overrides the color's alpha channel.
func (sb *SolidBackground) WithARGB(enabled bool, value int) *SolidBackground {
	sb.argbOn = enabled
	// Clamp ARGB value to valid range
	if value < 0 {
		value = 0
	} else if value > 255 {
		value = 255
	}
	sb.argbValue = value
	return sb
}

// Draw renders the solid background to the screen.
func (sb *SolidBackground) Draw(screen *ebiten.Image) {
	c := sb.color
	if sb.argbOn {
		c.A = uint8(sb.argbValue)
	}
	screen.Fill(c)
}

// Mode returns BackgroundModeSolid.
func (sb *SolidBackground) Mode() BackgroundMode {
	return BackgroundModeSolid
}

// Color returns the background color.
func (sb *SolidBackground) Color() color.RGBA {
	return sb.color
}

// NoneBackground renders no background (fully transparent).
type NoneBackground struct{}

// NewNoneBackground creates a new none/transparent background renderer.
func NewNoneBackground() *NoneBackground {
	return &NoneBackground{}
}

// Draw clears the screen with fully transparent color.
func (nb *NoneBackground) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 0})
}

// Mode returns BackgroundModeNone.
func (nb *NoneBackground) Mode() BackgroundMode {
	return BackgroundModeNone
}

// GradientBackground renders a gradient color background.
type GradientBackground struct {
	startColor color.RGBA
	endColor   color.RGBA
	direction  GradientDirection
	argbValue  int  // Alpha value override (0-255) when ARGB is enabled
	argbOn     bool // Whether ARGB visual is enabled

	// mu protects cachedImage, cachedWidth, and cachedHeight from concurrent access
	mu sync.RWMutex
	// cachedImage holds the pre-rendered gradient to avoid recalculating every frame
	cachedImage *ebiten.Image
	// cachedWidth and cachedHeight track dimensions to detect when regeneration is needed
	cachedWidth  int
	cachedHeight int
}

// NewGradientBackground creates a new gradient background renderer.
func NewGradientBackground(startColor, endColor color.RGBA, direction GradientDirection) *GradientBackground {
	return &GradientBackground{
		startColor: startColor,
		endColor:   endColor,
		direction:  direction,
		argbValue:  255,
	}
}

// WithARGB configures ARGB visual settings for the gradient background.
// When enabled, the argbValue overrides both colors' alpha channel.
// This invalidates the cached gradient image, requiring regeneration on next draw.
// Thread-safe: protected by mutex.
func (gb *GradientBackground) WithARGB(enabled bool, value int) *GradientBackground {
	// Clamp ARGB value to valid range
	if value < 0 {
		value = 0
	} else if value > 255 {
		value = 255
	}

	gb.mu.Lock()
	defer gb.mu.Unlock()

	// Invalidate cache if settings changed
	if gb.argbOn != enabled || gb.argbValue != value {
		gb.invalidateCacheLocked()
	}
	gb.argbOn = enabled
	gb.argbValue = value
	return gb
}

// Draw renders the gradient background to the screen.
// The gradient is calculated once and cached; subsequent calls reuse the cached image.
// Thread-safe: protected by mutex.
func (gb *GradientBackground) Draw(screen *ebiten.Image) {
	bounds := screen.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w == 0 || h == 0 {
		return
	}

	gb.mu.Lock()
	defer gb.mu.Unlock()

	// Check if we need to regenerate the cached image
	if gb.cachedImage == nil || gb.cachedWidth != w || gb.cachedHeight != h {
		gb.generateCachedImageLocked(w, h)
	}

	// Draw the cached gradient image to the screen
	if gb.cachedImage != nil {
		op := &ebiten.DrawImageOptions{}
		screen.DrawImage(gb.cachedImage, op)
	}
}

// generateCachedImageLocked creates and caches the gradient image at the specified dimensions.
// Must be called with mu held.
func (gb *GradientBackground) generateCachedImageLocked(w, h int) {
	// Clean up old cached image
	if gb.cachedImage != nil {
		gb.cachedImage.Deallocate()
	}

	// Create a pixel buffer for the gradient
	pixels := make([]byte, w*h*4)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := gb.interpolationFactor(x, y, w, h)
			c := gb.lerpColor(t)

			idx := (y*w + x) * 4
			pixels[idx] = c.R
			pixels[idx+1] = c.G
			pixels[idx+2] = c.B
			pixels[idx+3] = c.A
		}
	}

	// Create and cache the gradient image
	gb.cachedImage = ebiten.NewImage(w, h)
	gb.cachedImage.WritePixels(pixels)
	gb.cachedWidth = w
	gb.cachedHeight = h
}

// invalidateCacheLocked clears the cached gradient image, forcing regeneration on next draw.
// Must be called with mu held.
func (gb *GradientBackground) invalidateCacheLocked() {
	if gb.cachedImage != nil {
		gb.cachedImage.Deallocate()
		gb.cachedImage = nil
	}
	gb.cachedWidth = 0
	gb.cachedHeight = 0
}

// interpolationFactor calculates the gradient interpolation factor (0.0 to 1.0)
// based on position and direction.
// Handles edge cases where w=1 or h=1 to prevent division by zero.
func (gb *GradientBackground) interpolationFactor(x, y, w, h int) float64 {
	switch gb.direction {
	case GradientDirectionHorizontal:
		if w <= 1 {
			return 0.0
		}
		return float64(x) / float64(w-1)
	case GradientDirectionDiagonal:
		// Diagonal from top-left to bottom-right
		var xRatio, yRatio float64
		if w > 1 {
			xRatio = float64(x) / float64(w-1)
		}
		if h > 1 {
			yRatio = float64(y) / float64(h-1)
		}
		return (xRatio + yRatio) / 2.0
	case GradientDirectionRadial:
		// For single pixel (1x1), use start color for consistency with other directions
		if w <= 1 && h <= 1 {
			return 0.0
		}
		// Radial from center outward
		cx := float64(w) / 2.0
		cy := float64(h) / 2.0
		dx := float64(x) - cx
		dy := float64(y) - cy
		// Max distance is to corner
		maxDist := math.Sqrt(cx*cx + cy*cy)
		if maxDist == 0 {
			return 0.0
		}
		dist := math.Sqrt(dx*dx + dy*dy)
		return math.Min(dist/maxDist, 1.0)
	default: // GradientDirectionVertical
		if h <= 1 {
			return 0.0
		}
		return float64(y) / float64(h-1)
	}
}

// lerpColor linearly interpolates between start and end colors.
func (gb *GradientBackground) lerpColor(t float64) color.RGBA {
	// Clamp t to [0, 1]
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	startR := float64(gb.startColor.R)
	startG := float64(gb.startColor.G)
	startB := float64(gb.startColor.B)
	startA := float64(gb.startColor.A)

	endR := float64(gb.endColor.R)
	endG := float64(gb.endColor.G)
	endB := float64(gb.endColor.B)
	endA := float64(gb.endColor.A)

	c := color.RGBA{
		R: uint8(startR + t*(endR-startR)),
		G: uint8(startG + t*(endG-startG)),
		B: uint8(startB + t*(endB-startB)),
		A: uint8(startA + t*(endA-startA)),
	}

	// Override alpha if ARGB is enabled
	if gb.argbOn {
		c.A = uint8(gb.argbValue)
	}

	return c
}

// Mode returns BackgroundModeGradient.
func (gb *GradientBackground) Mode() BackgroundMode {
	return BackgroundModeGradient
}

// StartColor returns the gradient start color.
func (gb *GradientBackground) StartColor() color.RGBA {
	return gb.startColor
}

// EndColor returns the gradient end color.
func (gb *GradientBackground) EndColor() color.RGBA {
	return gb.endColor
}

// Direction returns the gradient direction.
func (gb *GradientBackground) Direction() GradientDirection {
	return gb.direction
}

// HasCachedImage returns true if a gradient image has been cached.
// Thread-safe: protected by read lock.
func (gb *GradientBackground) HasCachedImage() bool {
	gb.mu.RLock()
	defer gb.mu.RUnlock()
	return gb.cachedImage != nil
}

// Close releases resources held by the GradientBackground.
// Should be called when the background is no longer needed.
// Thread-safe: protected by mutex.
func (gb *GradientBackground) Close() {
	gb.mu.Lock()
	defer gb.mu.Unlock()
	gb.invalidateCacheLocked()
}

// PseudoBackground renders a cached screenshot of the desktop as the background.
// This provides fake transparency when a compositor is not available.
// The screenshot is captured from the root window at the window's position.
type PseudoBackground struct {
	// cachedImage holds the cached screenshot as an Ebiten image
	cachedImage *ebiten.Image
	// fallbackColor is used when screenshot capture fails
	fallbackColor color.RGBA
	// windowX, windowY are the window position for capturing the correct region
	windowX, windowY int
	// width, height are the dimensions to capture
	width, height int
	// imageProvider is a function that captures the desktop region
	// It returns an image.Image of the desktop at the specified position and size
	imageProvider ScreenshotProvider
	// needsRefresh indicates the cached image should be refreshed
	needsRefresh bool
}

// ScreenshotProvider is a function type for capturing desktop screenshots.
// It takes the position (x, y) and dimensions (width, height) of the region to capture.
// Returns an image.Image of the captured region or an error.
type ScreenshotProvider func(x, y, width, height int) (*ebiten.Image, error)

// NewPseudoBackground creates a new pseudo-transparency background renderer.
// It requires the window position and dimensions for capturing the correct desktop region.
// The fallbackColor is used if screenshot capture fails.
func NewPseudoBackground(x, y, width, height int, fallbackColor color.RGBA) *PseudoBackground {
	return &PseudoBackground{
		fallbackColor: fallbackColor,
		windowX:       x,
		windowY:       y,
		width:         width,
		height:        height,
		needsRefresh:  true,
	}
}

// SetScreenshotProvider sets the function used to capture desktop screenshots.
// This allows for platform-specific implementations and easier testing.
func (pb *PseudoBackground) SetScreenshotProvider(provider ScreenshotProvider) {
	pb.imageProvider = provider
	pb.needsRefresh = true
}

// Refresh marks the cached screenshot as needing refresh.
// The actual capture will happen on the next Draw call.
func (pb *PseudoBackground) Refresh() {
	pb.needsRefresh = true
}

// SetPosition updates the window position for screenshot capture.
// Call Refresh() after changing position to update the cached image.
func (pb *PseudoBackground) SetPosition(x, y int) {
	if pb.windowX != x || pb.windowY != y {
		pb.windowX = x
		pb.windowY = y
		pb.needsRefresh = true
	}
}

// Draw renders the pseudo-transparent background to the screen.
// If the screenshot hasn't been captured yet or needs refresh, it attempts to capture.
// Falls back to the fallback color if capture fails.
func (pb *PseudoBackground) Draw(screen *ebiten.Image) {
	// Check if we need to refresh the cached image
	if pb.needsRefresh || pb.cachedImage == nil {
		pb.refreshCache()
	}

	// Draw the cached image if available
	if pb.cachedImage != nil {
		// Draw the cached screenshot as the background
		op := &ebiten.DrawImageOptions{}
		screen.DrawImage(pb.cachedImage, op)
		return
	}

	// Fallback to solid color if no screenshot available
	screen.Fill(pb.fallbackColor)
}

// refreshCache attempts to capture a new screenshot of the desktop.
func (pb *PseudoBackground) refreshCache() {
	pb.needsRefresh = false

	// Skip if no provider is set
	if pb.imageProvider == nil {
		return
	}

	// Attempt to capture the desktop region
	img, err := pb.imageProvider(pb.windowX, pb.windowY, pb.width, pb.height)
	if err != nil {
		// Keep using existing cached image or fallback color
		return
	}

	// Clean up old cached image
	if pb.cachedImage != nil {
		pb.cachedImage.Deallocate()
	}

	pb.cachedImage = img
}

// Mode returns BackgroundModePseudo.
func (pb *PseudoBackground) Mode() BackgroundMode {
	return BackgroundModePseudo
}

// FallbackColor returns the fallback background color.
func (pb *PseudoBackground) FallbackColor() color.RGBA {
	return pb.fallbackColor
}

// HasCachedImage returns true if a screenshot has been successfully cached.
func (pb *PseudoBackground) HasCachedImage() bool {
	return pb.cachedImage != nil
}

// Close releases resources held by the PseudoBackground.
// Should be called when the background is no longer needed.
func (pb *PseudoBackground) Close() {
	if pb.cachedImage != nil {
		pb.cachedImage.Deallocate()
		pb.cachedImage = nil
	}
}

// NewBackgroundRenderer creates a BackgroundRenderer based on the mode and color.
// For BackgroundModeNone, the color is ignored.
// For BackgroundModeSolid, a SolidBackground is created with the given color.
func NewBackgroundRenderer(mode BackgroundMode, bgColor color.RGBA, argbVisual bool, argbValue int) BackgroundRenderer {
	switch mode {
	case BackgroundModeNone:
		return NewNoneBackground()
	default:
		return NewSolidBackground(bgColor).WithARGB(argbVisual, argbValue)
	}
}

// NewGradientBackgroundRenderer creates a BackgroundRenderer for gradient mode.
// This is a convenience function that creates a GradientBackground with ARGB settings.
func NewGradientBackgroundRenderer(startColor, endColor color.RGBA, direction GradientDirection, argbVisual bool, argbValue int) BackgroundRenderer {
	return NewGradientBackground(startColor, endColor, direction).WithARGB(argbVisual, argbValue)
}

// NewPseudoBackgroundRenderer creates a BackgroundRenderer for pseudo-transparency mode.
// This creates a PseudoBackground that captures a screenshot of the desktop at the
// specified window position to simulate transparency without a compositor.
// The fallbackColor is used if screenshot capture fails.
func NewPseudoBackgroundRenderer(x, y, width, height int, fallbackColor color.RGBA, provider ScreenshotProvider) BackgroundRenderer {
	pb := NewPseudoBackground(x, y, width, height, fallbackColor)
	if provider != nil {
		pb.SetScreenshotProvider(provider)
	}
	return pb
}
