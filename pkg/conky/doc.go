// Package conky provides the public API for embedding the go-conky system monitor.
// It allows third-party applications to run go-conky as a library component
// with full lifecycle management and configuration flexibility.
//
// # Basic Usage
//
// The simplest way to use conky is to create an instance from a configuration file:
//
//	c, err := conky.New("/path/to/config", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer c.Stop()
//
//	if err := c.Start(); err != nil {
//		log.Fatal(err)
//	}
//
// # Configuration Sources
//
// Conky supports three configuration sources:
//
//   - Disk file: Use [New] to load from a filesystem path
//   - Embedded FS: Use [NewFromFS] to load from an [io/fs.FS]
//   - io.Reader: Use [NewFromReader] for dynamic configurations
//
// # Lifecycle Management
//
// The [Conky] interface provides full lifecycle control:
//
//   - [Conky.Start] begins the rendering loop
//   - [Conky.Stop] gracefully shuts down the instance
//   - [Conky.Restart] reloads configuration and restarts
//   - [Conky.IsRunning] checks if the instance is active
//
// All methods are thread-safe and can be called from any goroutine.
//
// # Error Handling
//
// Runtime errors are reported through [ErrorHandler]:
//
//	c.SetErrorHandler(func(err error) {
//		log.Printf("conky error: %v", err)
//	})
//
// The handler is called asynchronously; do not block in the handler.
//
// # Headless Mode
//
// For applications that only need system monitoring data:
//
//	c, _ := conky.New("/path/to/config", &conky.Options{
//		Headless: true,
//	})
//	c.Start()
//	// Use c.Status() or access monitor data
package conky
