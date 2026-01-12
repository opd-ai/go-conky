//go:build !noebiten

package render

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// createTestImage creates a small test image for testing purposes.
func createTestImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	return img
}

// createTestPNG creates a PNG-encoded test image as bytes.
func createTestPNG(width, height int) []byte {
	img := createTestImage(width, height)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestNewImageWidget(t *testing.T) {
	iw := NewImageWidget(100, 200)

	if iw.x != 100 || iw.y != 200 {
		t.Errorf("position = (%v, %v), want (100, 200)", iw.x, iw.y)
	}
	if iw.scaleX != 1.0 || iw.scaleY != 1.0 {
		t.Errorf("scale = (%v, %v), want (1.0, 1.0)", iw.scaleX, iw.scaleY)
	}
	if iw.rotation != 0 {
		t.Errorf("rotation = %v, want 0", iw.rotation)
	}
	if iw.opacity != 1.0 {
		t.Errorf("opacity = %v, want 1.0", iw.opacity)
	}
	if iw.image != nil {
		t.Error("image should be nil initially")
	}
}

func TestImageWidgetLoadFromReader(t *testing.T) {
	iw := NewImageWidget(0, 0)
	pngData := createTestPNG(64, 48)

	err := iw.LoadFromReader(bytes.NewReader(pngData))
	if err != nil {
		t.Fatalf("LoadFromReader failed: %v", err)
	}

	if !iw.IsLoaded() {
		t.Error("IsLoaded should be true after loading")
	}

	w, h := iw.OriginalSize()
	if w != 64 || h != 48 {
		t.Errorf("OriginalSize = (%v, %v), want (64, 48)", w, h)
	}
}

func TestImageWidgetLoadFromReaderInvalid(t *testing.T) {
	iw := NewImageWidget(0, 0)

	err := iw.LoadFromReader(bytes.NewReader([]byte("not an image")))
	if err == nil {
		t.Error("LoadFromReader should fail for invalid data")
	}
}

func TestImageWidgetLoadFromFile(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")

	pngData := createTestPNG(32, 32)
	if err := os.WriteFile(tmpFile, pngData, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	iw := NewImageWidget(0, 0)
	err := iw.LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if !iw.IsLoaded() {
		t.Error("IsLoaded should be true after loading")
	}

	w, h := iw.OriginalSize()
	if w != 32 || h != 32 {
		t.Errorf("OriginalSize = (%v, %v), want (32, 32)", w, h)
	}
}

func TestImageWidgetLoadFromFileNotFound(t *testing.T) {
	iw := NewImageWidget(0, 0)

	err := iw.LoadFromFile("/nonexistent/path/to/image.png")
	if err == nil {
		t.Error("LoadFromFile should fail for non-existent file")
	}
}

func TestImageWidgetLoadFromImage(t *testing.T) {
	testImg := createTestImage(50, 40)
	ebitenImg := ebiten.NewImageFromImage(testImg)

	iw := NewImageWidget(10, 20)
	iw.LoadFromImage(ebitenImg)

	if !iw.IsLoaded() {
		t.Error("IsLoaded should be true after loading")
	}

	w, h := iw.OriginalSize()
	if w != 50 || h != 40 {
		t.Errorf("OriginalSize = (%v, %v), want (50, 40)", w, h)
	}
}

func TestImageWidgetLoadFromImageNil(t *testing.T) {
	iw := NewImageWidget(0, 0)

	// Load a real image first
	pngData := createTestPNG(32, 32)
	if err := iw.LoadFromReader(bytes.NewReader(pngData)); err != nil {
		t.Fatalf("initial load failed: %v", err)
	}

	// Load nil to clear
	iw.LoadFromImage(nil)

	if iw.IsLoaded() {
		t.Error("IsLoaded should be false after loading nil")
	}

	w, h := iw.OriginalSize()
	if w != 0 || h != 0 {
		t.Errorf("OriginalSize = (%v, %v), want (0, 0)", w, h)
	}
}

func TestImageWidgetSetPosition(t *testing.T) {
	iw := NewImageWidget(0, 0)
	iw.SetPosition(50, 75)

	x, y := iw.Position()
	if x != 50 || y != 75 {
		t.Errorf("Position = (%v, %v), want (50, 75)", x, y)
	}
}

func TestImageWidgetSetSize(t *testing.T) {
	iw := NewImageWidget(0, 0)
	pngData := createTestPNG(100, 50)
	if err := iw.LoadFromReader(bytes.NewReader(pngData)); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	iw.SetSize(200, 100)

	w, h := iw.Size()
	if w != 200 || h != 100 {
		t.Errorf("Size = (%v, %v), want (200, 100)", w, h)
	}

	// Check scale factors were updated
	scaleX, scaleY := iw.Scale()
	if scaleX != 2.0 || scaleY != 2.0 {
		t.Errorf("Scale = (%v, %v), want (2.0, 2.0)", scaleX, scaleY)
	}
}

func TestImageWidgetSizeWithNoImage(t *testing.T) {
	iw := NewImageWidget(0, 0)

	w, h := iw.Size()
	if w != 0 || h != 0 {
		t.Errorf("Size = (%v, %v), want (0, 0) for unloaded image", w, h)
	}
}

func TestImageWidgetSetScale(t *testing.T) {
	iw := NewImageWidget(0, 0)
	pngData := createTestPNG(100, 50)
	if err := iw.LoadFromReader(bytes.NewReader(pngData)); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	iw.SetScale(2.0, 3.0)

	scaleX, scaleY := iw.Scale()
	if scaleX != 2.0 || scaleY != 3.0 {
		t.Errorf("Scale = (%v, %v), want (2.0, 3.0)", scaleX, scaleY)
	}

	// Size should reflect the scale
	w, h := iw.Size()
	if w != 200 || h != 150 {
		t.Errorf("Size = (%v, %v), want (200, 150)", w, h)
	}
}

func TestImageWidgetSetScaleIgnoresNonPositive(t *testing.T) {
	iw := NewImageWidget(0, 0)
	iw.SetScale(2.0, 3.0)

	// Should ignore non-positive values
	iw.SetScale(0, -1)

	scaleX, scaleY := iw.Scale()
	if scaleX != 2.0 || scaleY != 3.0 {
		t.Errorf("Scale = (%v, %v), want (2.0, 3.0) (unchanged)", scaleX, scaleY)
	}
}

func TestImageWidgetSetUniformScale(t *testing.T) {
	iw := NewImageWidget(0, 0)
	iw.SetUniformScale(2.5)

	scaleX, scaleY := iw.Scale()
	if scaleX != 2.5 || scaleY != 2.5 {
		t.Errorf("Scale = (%v, %v), want (2.5, 2.5)", scaleX, scaleY)
	}
}

func TestImageWidgetSetRotation(t *testing.T) {
	iw := NewImageWidget(0, 0)
	iw.SetRotation(math.Pi / 4)

	if iw.Rotation() != math.Pi/4 {
		t.Errorf("Rotation = %v, want %v", iw.Rotation(), math.Pi/4)
	}
}

func TestImageWidgetSetOpacity(t *testing.T) {
	iw := NewImageWidget(0, 0)

	iw.SetOpacity(0.5)
	if iw.Opacity() != 0.5 {
		t.Errorf("Opacity = %v, want 0.5", iw.Opacity())
	}

	// Test clamping
	iw.SetOpacity(-0.5)
	if iw.Opacity() != 0 {
		t.Errorf("Opacity = %v, want 0 (clamped)", iw.Opacity())
	}

	iw.SetOpacity(1.5)
	if iw.Opacity() != 1 {
		t.Errorf("Opacity = %v, want 1 (clamped)", iw.Opacity())
	}
}

func TestImageWidgetSetCenterOrigin(t *testing.T) {
	iw := NewImageWidget(0, 0)

	if iw.CenterOrigin() {
		t.Error("CenterOrigin should be false by default")
	}

	iw.SetCenterOrigin(true)
	if !iw.CenterOrigin() {
		t.Error("CenterOrigin should be true after setting")
	}
}

func TestImageWidgetDraw(t *testing.T) {
	iw := NewImageWidget(50, 50)
	pngData := createTestPNG(32, 32)
	if err := iw.LoadFromReader(bytes.NewReader(pngData)); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	screen := ebiten.NewImage(200, 200)

	// Should not panic
	iw.Draw(screen)

	// Test with various settings
	iw.SetScale(2.0, 2.0)
	iw.Draw(screen)

	iw.SetRotation(math.Pi / 4)
	iw.Draw(screen)

	iw.SetOpacity(0.5)
	iw.Draw(screen)

	iw.SetCenterOrigin(true)
	iw.Draw(screen)
}

func TestImageWidgetDrawNoImage(t *testing.T) {
	iw := NewImageWidget(50, 50)
	screen := ebiten.NewImage(200, 200)

	// Should not panic when no image is loaded
	iw.Draw(screen)
}

func TestImageWidgetClear(t *testing.T) {
	iw := NewImageWidget(0, 0)
	pngData := createTestPNG(32, 32)
	if err := iw.LoadFromReader(bytes.NewReader(pngData)); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if !iw.IsLoaded() {
		t.Fatal("image should be loaded")
	}

	iw.Clear()

	if iw.IsLoaded() {
		t.Error("IsLoaded should be false after Clear")
	}

	w, h := iw.OriginalSize()
	if w != 0 || h != 0 {
		t.Errorf("OriginalSize = (%v, %v), want (0, 0)", w, h)
	}
}

func TestImageWidgetConcurrentAccess(t *testing.T) {
	iw := NewImageWidget(0, 0)
	pngData := createTestPNG(32, 32)
	if err := iw.LoadFromReader(bytes.NewReader(pngData)); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	screen := ebiten.NewImage(200, 200)

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			iw.SetPosition(float64(i), float64(i))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			iw.SetScale(float64(i%3)+1, float64(i%3)+1)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			iw.SetOpacity(float64(i) / 100.0)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			iw.Draw(screen)
		}
	}()

	wg.Wait()
}

// ImageLoader tests

func TestNewImageLoader(t *testing.T) {
	loader := NewImageLoader()
	if loader == nil {
		t.Error("NewImageLoader returned nil")
	}
}

func TestImageLoaderLoadFile(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")

	pngData := createTestPNG(64, 48)
	if err := os.WriteFile(tmpFile, pngData, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	loader := NewImageLoader()
	img, w, h, err := loader.LoadFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if img == nil {
		t.Error("loaded image is nil")
	}
	if w != 64 || h != 48 {
		t.Errorf("dimensions = (%v, %v), want (64, 48)", w, h)
	}
}

func TestImageLoaderLoadFileNotFound(t *testing.T) {
	loader := NewImageLoader()
	_, _, _, err := loader.LoadFile("/nonexistent/path.png")
	if err == nil {
		t.Error("LoadFile should fail for non-existent file")
	}
}

func TestImageLoaderLoadReader(t *testing.T) {
	pngData := createTestPNG(32, 24)

	loader := NewImageLoader()
	img, w, h, err := loader.LoadReader(bytes.NewReader(pngData))
	if err != nil {
		t.Fatalf("LoadReader failed: %v", err)
	}

	if img == nil {
		t.Error("loaded image is nil")
	}
	if w != 32 || h != 24 {
		t.Errorf("dimensions = (%v, %v), want (32, 24)", w, h)
	}
}

func TestImageLoaderLoadReaderInvalid(t *testing.T) {
	loader := NewImageLoader()
	_, _, _, err := loader.LoadReader(bytes.NewReader([]byte("not an image")))
	if err == nil {
		t.Error("LoadReader should fail for invalid data")
	}
}

// ImageCache tests

func TestNewImageCache(t *testing.T) {
	cache := NewImageCache()
	if cache == nil {
		t.Error("NewImageCache returned nil")
	}
	if cache.Size() != 0 {
		t.Errorf("initial cache size = %v, want 0", cache.Size())
	}
}

func TestImageCacheLoad(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")

	pngData := createTestPNG(32, 32)
	if err := os.WriteFile(tmpFile, pngData, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cache := NewImageCache()

	// First load should read from file
	img1, err := cache.Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if img1 == nil {
		t.Error("loaded image is nil")
	}
	if cache.Size() != 1 {
		t.Errorf("cache size = %v, want 1", cache.Size())
	}

	// Second load should use cache
	img2, err := cache.Load(tmpFile)
	if err != nil {
		t.Fatalf("second Load failed: %v", err)
	}
	if img1 != img2 {
		t.Error("cached image should be the same instance")
	}
}

func TestImageCacheLoadNotFound(t *testing.T) {
	cache := NewImageCache()
	_, err := cache.Load("/nonexistent/path.png")
	if err == nil {
		t.Error("Load should fail for non-existent file")
	}
}

func TestImageCacheGet(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")

	pngData := createTestPNG(32, 32)
	if err := os.WriteFile(tmpFile, pngData, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cache := NewImageCache()

	// Get before load should return nil
	if cache.Get(tmpFile) != nil {
		t.Error("Get should return nil for uncached image")
	}

	// Load the image
	_, err := cache.Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Get after load should return the image
	if cache.Get(tmpFile) == nil {
		t.Error("Get should return image after loading")
	}
}

func TestImageCacheRemove(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")

	pngData := createTestPNG(32, 32)
	if err := os.WriteFile(tmpFile, pngData, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cache := NewImageCache()

	// Load the image
	_, err := cache.Load(tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cache.Size() != 1 {
		t.Errorf("cache size = %v, want 1", cache.Size())
	}

	// Remove the image
	cache.Remove(tmpFile)

	if cache.Size() != 0 {
		t.Errorf("cache size after remove = %v, want 0", cache.Size())
	}
	if cache.Get(tmpFile) != nil {
		t.Error("Get should return nil after Remove")
	}
}

func TestImageCacheRemoveNonExistent(t *testing.T) {
	cache := NewImageCache()

	// Should not panic when removing non-existent key
	cache.Remove("/nonexistent/path.png")

	if cache.Size() != 0 {
		t.Errorf("cache size = %v, want 0", cache.Size())
	}
}

func TestImageCacheClear(t *testing.T) {
	// Create temporary PNG files
	tmpDir := t.TempDir()

	cache := NewImageCache()

	for i := 0; i < 3; i++ {
		tmpFile := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".png")
		pngData := createTestPNG(32, 32)
		if err := os.WriteFile(tmpFile, pngData, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		if _, err := cache.Load(tmpFile); err != nil {
			t.Fatalf("Load failed: %v", err)
		}
	}

	if cache.Size() != 3 {
		t.Errorf("cache size = %v, want 3", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("cache size after clear = %v, want 0", cache.Size())
	}
}

func TestImageCacheConcurrentAccess(t *testing.T) {
	// Create temporary PNG files
	tmpDir := t.TempDir()
	var tmpFiles []string

	for i := 0; i < 5; i++ {
		tmpFile := filepath.Join(tmpDir, "test"+string(rune('0'+i))+".png")
		pngData := createTestPNG(32, 32)
		if err := os.WriteFile(tmpFile, pngData, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		tmpFiles = append(tmpFiles, tmpFile)
	}

	cache := NewImageCache()

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			idx := i % len(tmpFiles)
			_, _ = cache.Load(tmpFiles[idx])
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			idx := i % len(tmpFiles)
			_ = cache.Get(tmpFiles[idx])
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = cache.Size()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			idx := i % len(tmpFiles)
			cache.Remove(tmpFiles[idx])
		}
	}()

	wg.Wait()
}
