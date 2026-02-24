package conky

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestConfigWatcher_DetectsFileChange(t *testing.T) {
	// Create a temporary directory and config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	// Create initial config file
	if err := os.WriteFile(configPath, []byte("initial content"), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	var reloadCount atomic.Int32
	var lastError atomic.Value

	watcher, err := newConfigWatcher(
		configPath,
		50*time.Millisecond, // Short debounce for testing
		func() error {
			reloadCount.Add(1)
			return nil
		},
		func(err error) {
			lastError.Store(err)
		},
	)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	watcher.Start()
	defer watcher.Stop()

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(configPath, []byte("modified content"), 0644); err != nil {
		t.Fatalf("failed to modify config file: %v", err)
	}

	// Wait for debounce and reload
	time.Sleep(200 * time.Millisecond)

	if count := reloadCount.Load(); count != 1 {
		t.Errorf("expected 1 reload, got %d", count)
	}

	if err := lastError.Load(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigWatcher_DebounceMultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	if err := os.WriteFile(configPath, []byte("initial"), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	var reloadCount atomic.Int32

	watcher, err := newConfigWatcher(
		configPath,
		100*time.Millisecond,
		func() error {
			reloadCount.Add(1)
			return nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	watcher.Start()
	defer watcher.Stop()

	time.Sleep(50 * time.Millisecond)

	// Multiple rapid writes should be debounced to single reload
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(configPath, []byte("content "+string(rune('0'+i))), 0644); err != nil {
			t.Fatalf("failed to modify config file: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Wait for debounce period to complete
	time.Sleep(200 * time.Millisecond)

	// Should have only 1 reload due to debouncing
	if count := reloadCount.Load(); count != 1 {
		t.Errorf("expected 1 reload (debounced), got %d", count)
	}
}

func TestConfigWatcher_StopPreventsReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	if err := os.WriteFile(configPath, []byte("initial"), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	var reloadCount atomic.Int32

	watcher, err := newConfigWatcher(
		configPath,
		50*time.Millisecond,
		func() error {
			reloadCount.Add(1)
			return nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	watcher.Start()
	time.Sleep(50 * time.Millisecond)

	// Stop the watcher
	watcher.Stop()

	// Modify the file after stop
	if err := os.WriteFile(configPath, []byte("modified"), 0644); err != nil {
		t.Fatalf("failed to modify config file: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Should have no reloads since watcher was stopped
	if count := reloadCount.Load(); count != 0 {
		t.Errorf("expected 0 reloads after stop, got %d", count)
	}
}

func TestConfigWatcher_HandlesAtomicSave(t *testing.T) {
	// Test that watcher handles atomic saves (write to temp, rename)
	// This is how editors like vim, emacs save files
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	if err := os.WriteFile(configPath, []byte("initial"), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	var reloadCount atomic.Int32

	watcher, err := newConfigWatcher(
		configPath,
		50*time.Millisecond,
		func() error {
			reloadCount.Add(1)
			return nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	watcher.Start()
	defer watcher.Stop()

	time.Sleep(50 * time.Millisecond)

	// Simulate atomic save: write to temp file, then rename
	tempPath := configPath + ".tmp"
	if err := os.WriteFile(tempPath, []byte("atomic save content"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	if err := os.Rename(tempPath, configPath); err != nil {
		t.Fatalf("failed to rename temp file: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Should detect the rename/create event
	if count := reloadCount.Load(); count < 1 {
		t.Errorf("expected at least 1 reload for atomic save, got %d", count)
	}
}

func TestConfigWatcher_IgnoresOtherFiles(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")
	otherPath := filepath.Join(tmpDir, "other.txt")

	if err := os.WriteFile(configPath, []byte("config"), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	var reloadCount atomic.Int32

	watcher, err := newConfigWatcher(
		configPath,
		50*time.Millisecond,
		func() error {
			reloadCount.Add(1)
			return nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	watcher.Start()
	defer watcher.Stop()

	time.Sleep(50 * time.Millisecond)

	// Create and modify a different file in the same directory
	if err := os.WriteFile(otherPath, []byte("other content"), 0644); err != nil {
		t.Fatalf("failed to create other file: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Should not have triggered reload
	if count := reloadCount.Load(); count != 0 {
		t.Errorf("expected 0 reloads for other file, got %d", count)
	}
}

func TestConfigWatcher_ErrorCallback(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.conf")

	if err := os.WriteFile(configPath, []byte("config"), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	var errorReceived atomic.Bool
	reloadErr := os.ErrClosed // Simulate an error during reload

	watcher, err := newConfigWatcher(
		configPath,
		50*time.Millisecond,
		func() error {
			return reloadErr
		},
		func(err error) {
			errorReceived.Store(true)
		},
	)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	watcher.Start()
	defer watcher.Stop()

	time.Sleep(50 * time.Millisecond)

	// Trigger a reload that will fail
	if err := os.WriteFile(configPath, []byte("modified"), 0644); err != nil {
		t.Fatalf("failed to modify config file: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if !errorReceived.Load() {
		t.Error("expected error callback to be called")
	}
}
