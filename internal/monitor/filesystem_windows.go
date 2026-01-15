//go:build windows

package monitor

// filesystemReader is a stub for Windows builds.
// Windows filesystem monitoring is implemented in the platform package.
type filesystemReader struct{}

// newFilesystemReader creates a stub filesystemReader for Windows.
func newFilesystemReader() *filesystemReader {
	return &filesystemReader{}
}

// ReadStats returns empty filesystem stats for Windows.
// Use the platform package for Windows filesystem monitoring.
func (r *filesystemReader) ReadStats() (FilesystemStats, error) {
	return FilesystemStats{}, nil
}
