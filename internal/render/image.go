// Package render provides Ebiten-based rendering capabilities for conky-go.
// This file implements image loading and bitmap drawing capabilities for
// displaying PNG, JPEG, and GIF images in the rendering engine.
package render

import (
	"fmt"
	"image"
	// Register image decoders for common formats
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// ImageWidget displays an image loaded from a file or reader.
// It supports PNG, JPEG, and GIF formats and provides options for
// positioning, scaling, rotation, and opacity.
type ImageWidget struct {
	x, y          float64       // Position (top-left corner)
	width, height float64       // Display dimensions (0 = original size)
	scaleX        float64       // Horizontal scale factor
	scaleY        float64       // Vertical scale factor
	rotation      float64       // Rotation angle in radians
	opacity       float64       // Opacity (0.0 = transparent, 1.0 = opaque)
	image         *ebiten.Image // The loaded Ebiten image
	originalW     int           // Original image width
	originalH     int           // Original image height
	centerOrigin  bool          // If true, position is center; otherwise top-left
	mu            sync.RWMutex
}

// NewImageWidget creates a new empty ImageWidget at the specified position.
// Use LoadFromFile or LoadFromReader to load an image.
func NewImageWidget(x, y float64) *ImageWidget {
	return &ImageWidget{
		x:        x,
		y:        y,
		scaleX:   1.0,
		scaleY:   1.0,
		rotation: 0,
		opacity:  1.0,
	}
}

// LoadFromFile loads an image from a file path.
// Supported formats: PNG, JPEG, GIF.
// Returns an error if the file cannot be read or decoded.
func (iw *ImageWidget) LoadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	return iw.LoadFromReader(file)
}

// LoadFromReader loads an image from an io.Reader.
// Supported formats: PNG, JPEG, GIF.
// Returns an error if the image cannot be decoded.
func (iw *ImageWidget) LoadFromReader(r io.Reader) error {
	img, _, err := image.Decode(r)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	iw.mu.Lock()
	defer iw.mu.Unlock()

	// Convert to Ebiten image
	iw.image = ebiten.NewImageFromImage(img)
	bounds := img.Bounds()
	iw.originalW = bounds.Dx()
	iw.originalH = bounds.Dy()

	return nil
}

// LoadFromImage loads an Ebiten image directly.
// This is useful when the image is already in memory.
func (iw *ImageWidget) LoadFromImage(img *ebiten.Image) {
	iw.mu.Lock()
	defer iw.mu.Unlock()

	iw.image = img
	if img != nil {
		bounds := img.Bounds()
		iw.originalW = bounds.Dx()
		iw.originalH = bounds.Dy()
	} else {
		iw.originalW = 0
		iw.originalH = 0
	}
}

// SetPosition sets the position of the image.
// By default, this is the top-left corner. Use SetCenterOrigin(true) to
// make this the center of the image.
func (iw *ImageWidget) SetPosition(x, y float64) {
	iw.mu.Lock()
	defer iw.mu.Unlock()
	iw.x = x
	iw.y = y
}

// Position returns the current position of the image.
func (iw *ImageWidget) Position() (x, y float64) {
	iw.mu.RLock()
	defer iw.mu.RUnlock()
	return iw.x, iw.y
}

// SetSize sets the display dimensions of the image.
// If width or height is 0, the original dimension is used.
// This overrides any scale settings.
func (iw *ImageWidget) SetSize(width, height float64) {
	iw.mu.Lock()
	defer iw.mu.Unlock()
	iw.width = width
	iw.height = height

	// Calculate scale factors if original dimensions are available
	if iw.originalW > 0 && iw.originalH > 0 {
		if width > 0 {
			iw.scaleX = width / float64(iw.originalW)
		} else {
			iw.scaleX = 1.0
		}
		if height > 0 {
			iw.scaleY = height / float64(iw.originalH)
		} else {
			iw.scaleY = 1.0
		}
	}
}

// Size returns the display dimensions of the image.
// Returns (0, 0) if no image is loaded.
func (iw *ImageWidget) Size() (width, height float64) {
	iw.mu.RLock()
	defer iw.mu.RUnlock()
	if iw.width > 0 && iw.height > 0 {
		return iw.width, iw.height
	}
	return float64(iw.originalW) * iw.scaleX, float64(iw.originalH) * iw.scaleY
}

// OriginalSize returns the original dimensions of the loaded image.
// Returns (0, 0) if no image is loaded.
func (iw *ImageWidget) OriginalSize() (width, height int) {
	iw.mu.RLock()
	defer iw.mu.RUnlock()
	return iw.originalW, iw.originalH
}

// SetScale sets the scale factors for the image.
// This overrides any size settings.
func (iw *ImageWidget) SetScale(scaleX, scaleY float64) {
	iw.mu.Lock()
	defer iw.mu.Unlock()
	if scaleX > 0 {
		iw.scaleX = scaleX
	}
	if scaleY > 0 {
		iw.scaleY = scaleY
	}
	// Clear explicit size settings
	iw.width = 0
	iw.height = 0
}

// SetUniformScale sets both scale factors to the same value.
func (iw *ImageWidget) SetUniformScale(scale float64) {
	iw.SetScale(scale, scale)
}

// Scale returns the current scale factors.
func (iw *ImageWidget) Scale() (scaleX, scaleY float64) {
	iw.mu.RLock()
	defer iw.mu.RUnlock()
	return iw.scaleX, iw.scaleY
}

// SetRotation sets the rotation angle in radians.
// Positive values rotate clockwise.
func (iw *ImageWidget) SetRotation(radians float64) {
	iw.mu.Lock()
	defer iw.mu.Unlock()
	iw.rotation = radians
}

// Rotation returns the current rotation angle in radians.
func (iw *ImageWidget) Rotation() float64 {
	iw.mu.RLock()
	defer iw.mu.RUnlock()
	return iw.rotation
}

// SetOpacity sets the opacity of the image.
// Value should be between 0.0 (fully transparent) and 1.0 (fully opaque).
// Values outside this range are clamped.
func (iw *ImageWidget) SetOpacity(opacity float64) {
	iw.mu.Lock()
	defer iw.mu.Unlock()
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	iw.opacity = opacity
}

// Opacity returns the current opacity value.
func (iw *ImageWidget) Opacity() float64 {
	iw.mu.RLock()
	defer iw.mu.RUnlock()
	return iw.opacity
}

// SetCenterOrigin sets whether the position represents the center of the image.
// If true, the image is drawn centered at the position.
// If false (default), the position is the top-left corner.
func (iw *ImageWidget) SetCenterOrigin(centered bool) {
	iw.mu.Lock()
	defer iw.mu.Unlock()
	iw.centerOrigin = centered
}

// CenterOrigin returns whether the position represents the center.
func (iw *ImageWidget) CenterOrigin() bool {
	iw.mu.RLock()
	defer iw.mu.RUnlock()
	return iw.centerOrigin
}

// IsLoaded returns true if an image has been successfully loaded.
func (iw *ImageWidget) IsLoaded() bool {
	iw.mu.RLock()
	defer iw.mu.RUnlock()
	return iw.image != nil
}

// Draw renders the image onto the given screen.
// Does nothing if no image is loaded.
func (iw *ImageWidget) Draw(screen *ebiten.Image) {
	iw.mu.RLock()
	defer iw.mu.RUnlock()

	if iw.image == nil {
		return
	}

	// Create draw options
	op := &ebiten.DrawImageOptions{}

	// Apply transformations in correct order:
	// 1. Translate to origin for rotation/scaling (if center origin)
	// 2. Scale
	// 3. Rotate
	// 4. Translate to final position

	if iw.centerOrigin {
		// Move origin to center of image for rotation
		op.GeoM.Translate(-float64(iw.originalW)/2, -float64(iw.originalH)/2)
	}

	// Apply scale
	op.GeoM.Scale(iw.scaleX, iw.scaleY)

	// Apply rotation
	if iw.rotation != 0 {
		op.GeoM.Rotate(iw.rotation)
	}

	// Apply translation to final position
	op.GeoM.Translate(iw.x, iw.y)

	// Apply opacity
	if iw.opacity < 1.0 {
		op.ColorScale.ScaleAlpha(float32(iw.opacity))
	}

	// Draw the image
	screen.DrawImage(iw.image, op)
}

// Clear removes the loaded image and resets dimensions.
func (iw *ImageWidget) Clear() {
	iw.mu.Lock()
	defer iw.mu.Unlock()

	if iw.image != nil {
		iw.image.Deallocate()
		iw.image = nil
	}
	iw.originalW = 0
	iw.originalH = 0
	iw.width = 0
	iw.height = 0
}

// ImageLoader provides utilities for loading images without creating a widget.
// This is useful for caching and sharing images between multiple widgets.
type ImageLoader struct{}

// NewImageLoader creates a new ImageLoader.
func NewImageLoader() *ImageLoader {
	return &ImageLoader{}
}

// LoadFile loads an image from a file path.
// Returns the Ebiten image and its dimensions.
func (il *ImageLoader) LoadFile(path string) (*ebiten.Image, int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	return il.LoadReader(file)
}

// LoadReader loads an image from an io.Reader.
// Returns the Ebiten image and its dimensions.
func (il *ImageLoader) LoadReader(r io.Reader) (*ebiten.Image, int, int, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	ebitenImg := ebiten.NewImageFromImage(img)

	return ebitenImg, bounds.Dx(), bounds.Dy(), nil
}

// ImageCache provides caching for loaded images to avoid reloading.
type ImageCache struct {
	cache map[string]*ebiten.Image
	mu    sync.RWMutex
}

// NewImageCache creates a new image cache.
func NewImageCache() *ImageCache {
	return &ImageCache{
		cache: make(map[string]*ebiten.Image),
	}
}

// Load loads an image from a file, using the cache if available.
// Uses double-checked locking to prevent race conditions and duplicate loads.
func (ic *ImageCache) Load(path string) (*ebiten.Image, error) {
	// Check cache first (read lock)
	ic.mu.RLock()
	if img, ok := ic.cache[path]; ok {
		ic.mu.RUnlock()
		return img, nil
	}
	ic.mu.RUnlock()

	// Acquire write lock and check again (double-checked locking)
	ic.mu.Lock()
	defer ic.mu.Unlock()

	// Check if another goroutine loaded it while we were waiting for the lock
	if img, ok := ic.cache[path]; ok {
		return img, nil
	}

	// Load from file
	loader := NewImageLoader()
	img, _, _, err := loader.LoadFile(path)
	if err != nil {
		return nil, err
	}

	// Store in cache
	ic.cache[path] = img

	return img, nil
}

// Get retrieves an image from the cache without loading.
// Returns nil if the image is not cached.
func (ic *ImageCache) Get(path string) *ebiten.Image {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.cache[path]
}

// Remove removes an image from the cache and deallocates it.
func (ic *ImageCache) Remove(path string) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if img, ok := ic.cache[path]; ok {
		img.Deallocate()
		delete(ic.cache, path)
	}
}

// Clear removes all images from the cache and deallocates them.
func (ic *ImageCache) Clear() {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	for _, img := range ic.cache {
		img.Deallocate()
	}
	ic.cache = make(map[string]*ebiten.Image)
}

// Size returns the number of images in the cache.
func (ic *ImageCache) Size() int {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return len(ic.cache)
}
