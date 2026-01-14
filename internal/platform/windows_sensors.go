// +build windows

package platform

// windowsSensorProvider implements SensorProvider for Windows systems.
// Note: Hardware sensor monitoring on Windows is limited without third-party drivers.
// Advanced sensor monitoring would require WMI queries for specific hardware or
// third-party tools like OpenHardwareMonitor.
type windowsSensorProvider struct{}

func newWindowsSensorProvider() *windowsSensorProvider {
	return &windowsSensorProvider{}
}

func (s *windowsSensorProvider) Temperatures() ([]SensorReading, error) {
	// Temperature sensors are not easily accessible on Windows without WMI
	// or third-party libraries. Return empty for now.
	// Future enhancement: Use WMI to query Win32_TemperatureProbe
	return []SensorReading{}, nil
}

func (s *windowsSensorProvider) Fans() ([]SensorReading, error) {
	// Fan sensors are not easily accessible on Windows without WMI
	// or third-party libraries. Return empty for now.
	// Future enhancement: Use WMI to query Win32_Fan
	return []SensorReading{}, nil
}
