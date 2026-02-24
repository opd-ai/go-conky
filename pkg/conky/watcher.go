package conky

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// DefaultWatchDebounce is the default debounce interval for file watch events.
const DefaultWatchDebounce = 500 * time.Millisecond

// configWatcher monitors configuration files for changes and triggers reloads.
type configWatcher struct {
	watcher   *fsnotify.Watcher
	filePath  string
	debounce  time.Duration
	onReload  func() error
	onError   func(error)
	stopCh    chan struct{}
	stoppedCh chan struct{}
	mu        sync.Mutex
	running   bool
}

// newConfigWatcher creates a new configuration file watcher.
// onReload is called when the file changes (after debouncing).
// onError is called when errors occur during watching.
func newConfigWatcher(filePath string, debounce time.Duration, onReload func() error, onError func(error)) (*configWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if debounce <= 0 {
		debounce = DefaultWatchDebounce
	}

	// Watch the directory containing the config file, not the file itself.
	// This handles editors that atomically rename files (vim, emacs, etc.).
	dir := filepath.Dir(filePath)
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, err
	}

	return &configWatcher{
		watcher:   watcher,
		filePath:  filePath,
		debounce:  debounce,
		onReload:  onReload,
		onError:   onError,
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}, nil
}

// Start begins watching for file changes in a goroutine.
func (cw *configWatcher) Start() {
	cw.mu.Lock()
	if cw.running {
		cw.mu.Unlock()
		return
	}
	cw.running = true
	cw.mu.Unlock()

	go cw.watchLoop()
}

// Stop stops the file watcher and waits for cleanup.
func (cw *configWatcher) Stop() {
	cw.mu.Lock()
	if !cw.running {
		cw.mu.Unlock()
		return
	}
	cw.mu.Unlock()

	close(cw.stopCh)
	<-cw.stoppedCh
}

// watchLoop is the main event loop for file watching with debouncing.
func (cw *configWatcher) watchLoop() {
	defer close(cw.stoppedCh)
	defer cw.watcher.Close()

	absPath, _ := filepath.Abs(cw.filePath)
	baseName := filepath.Base(cw.filePath)

	var debounceTimer *time.Timer
	var debounceCh <-chan time.Time

	for {
		select {
		case <-cw.stopCh:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			cw.mu.Lock()
			cw.running = false
			cw.mu.Unlock()
			return

		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}

			// Check if this event is for our config file
			eventBase := filepath.Base(event.Name)
			eventAbs, _ := filepath.Abs(event.Name)

			if eventBase != baseName && eventAbs != absPath {
				continue
			}

			// Only react to write/create/rename events (covers atomic saves)
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}

			// Debounce: reset the timer on each event
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.NewTimer(cw.debounce)
			debounceCh = debounceTimer.C

		case <-debounceCh:
			// Debounce period elapsed, trigger reload
			if cw.onReload != nil {
				if err := cw.onReload(); err != nil && cw.onError != nil {
					cw.onError(err)
				}
			}
			debounceTimer = nil
			debounceCh = nil

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			if cw.onError != nil {
				cw.onError(err)
			}
		}
	}
}
