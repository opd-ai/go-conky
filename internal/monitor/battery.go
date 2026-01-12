package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// BatteryInfo contains information about a single battery.
type BatteryInfo struct {
	// Name is the power supply name (e.g., "BAT0", "BAT1").
	Name string
	// Present indicates if the battery is present.
	Present bool
	// Status is the charging status ("Charging", "Discharging", "Full", "Not charging").
	Status string
	// Capacity is the current charge level as a percentage (0-100).
	Capacity int
	// CapacityLevel is the capacity level string ("Normal", "Low", "Critical", "Full").
	CapacityLevel string
	// EnergyNow is the current energy in microWatt-hours (µWh).
	EnergyNow uint64
	// EnergyFull is the full charge capacity in microWatt-hours (µWh).
	EnergyFull uint64
	// EnergyFullDesign is the designed full charge capacity in microWatt-hours (µWh).
	EnergyFullDesign uint64
	// PowerNow is the current power draw/charge rate in microWatts (µW).
	PowerNow uint64
	// VoltageNow is the current voltage in microVolts (µV).
	VoltageNow uint64
	// VoltageMinDesign is the minimum design voltage in microVolts (µV).
	VoltageMinDesign uint64
	// ChargeNow is the current charge in microAmp-hours (µAh), for charge-based batteries.
	ChargeNow uint64
	// ChargeFull is the full charge in microAmp-hours (µAh), for charge-based batteries.
	ChargeFull uint64
	// ChargeFullDesign is the design full charge in microAmp-hours (µAh).
	ChargeFullDesign uint64
	// CurrentNow is the current current draw in microAmps (µA), for charge-based batteries.
	CurrentNow uint64
	// Technology is the battery technology (e.g., "Li-ion", "Li-poly").
	Technology string
	// Manufacturer is the battery manufacturer.
	Manufacturer string
	// ModelName is the battery model name.
	ModelName string
	// SerialNumber is the battery serial number.
	SerialNumber string
	// CycleCount is the number of charge cycles (if available).
	CycleCount int
	// Health is the battery health percentage (EnergyFull/EnergyFullDesign * 100).
	Health float64
	// TimeToEmpty is the estimated time to empty in seconds (if discharging).
	TimeToEmpty float64
	// TimeToFull is the estimated time to full in seconds (if charging).
	TimeToFull float64
}

// ACAdapterInfo contains information about an AC adapter.
type ACAdapterInfo struct {
	// Name is the power supply name (e.g., "AC", "ADP1").
	Name string
	// Online indicates if the adapter is connected.
	Online bool
}

// BatteryStats contains battery and power supply statistics.
type BatteryStats struct {
	// Batteries contains battery information keyed by battery name.
	Batteries map[string]BatteryInfo
	// ACAdapters contains AC adapter information keyed by adapter name.
	ACAdapters map[string]ACAdapterInfo
	// ACOnline indicates if any AC adapter is connected.
	ACOnline bool
	// TotalCapacity is the weighted average capacity across all batteries.
	TotalCapacity float64
	// TotalEnergyNow is the sum of EnergyNow across all batteries.
	TotalEnergyNow uint64
	// TotalEnergyFull is the sum of EnergyFull across all batteries.
	TotalEnergyFull uint64
	// IsCharging indicates if any battery is charging.
	IsCharging bool
	// IsDischarging indicates if any battery is discharging.
	IsDischarging bool
}

// batteryReader reads battery information from /sys/class/power_supply.
type batteryReader struct {
	mu              sync.Mutex
	powerSupplyPath string
}

// newBatteryReader creates a new batteryReader with default paths.
func newBatteryReader() *batteryReader {
	return &batteryReader{
		powerSupplyPath: "/sys/class/power_supply",
	}
}

// ReadStats reads current battery and power supply statistics.
func (r *batteryReader) ReadStats() (BatteryStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := BatteryStats{
		Batteries:  make(map[string]BatteryInfo),
		ACAdapters: make(map[string]ACAdapterInfo),
	}

	// Check if power_supply directory exists
	if _, err := os.Stat(r.powerSupplyPath); os.IsNotExist(err) {
		return stats, nil // No power supply support, return empty stats
	}

	entries, err := os.ReadDir(r.powerSupplyPath)
	if err != nil {
		return stats, fmt.Errorf("reading %s: %w", r.powerSupplyPath, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		devicePath := filepath.Join(r.powerSupplyPath, entry.Name())
		supplyType, err := r.readStringFile(filepath.Join(devicePath, "type"))
		if err != nil {
			continue // Skip if we can't determine the type
		}

		switch strings.ToLower(supplyType) {
		case "battery":
			battery, err := r.readBattery(devicePath, entry.Name())
			if err == nil {
				stats.Batteries[battery.Name] = battery
				if battery.Status == "Charging" {
					stats.IsCharging = true
				}
				if battery.Status == "Discharging" {
					stats.IsDischarging = true
				}
				stats.TotalEnergyNow += battery.EnergyNow
				stats.TotalEnergyFull += battery.EnergyFull
			}
		case "mains", "ups":
			adapter, err := r.readACAdapter(devicePath, entry.Name())
			if err == nil {
				stats.ACAdapters[adapter.Name] = adapter
				if adapter.Online {
					stats.ACOnline = true
				}
			}
		}
	}

	// Calculate total capacity as weighted average
	if stats.TotalEnergyFull > 0 {
		stats.TotalCapacity = float64(stats.TotalEnergyNow) / float64(stats.TotalEnergyFull) * 100
	}

	return stats, nil
}

// readBattery reads battery information from a power supply device path.
func (r *batteryReader) readBattery(devicePath, name string) (BatteryInfo, error) {
	battery := BatteryInfo{
		Name: name,
	}

	// Check if battery is present
	if present, err := r.readIntFile(filepath.Join(devicePath, "present")); err == nil {
		battery.Present = present == 1
	} else {
		// If no present file, assume present if we can read other attributes
		battery.Present = true
	}

	// Read status
	if status, err := r.readStringFile(filepath.Join(devicePath, "status")); err == nil {
		battery.Status = status
	}

	// Read capacity (percentage)
	if capacity, err := r.readIntFile(filepath.Join(devicePath, "capacity")); err == nil {
		battery.Capacity = int(capacity)
	}

	// Read capacity level
	if level, err := r.readStringFile(filepath.Join(devicePath, "capacity_level")); err == nil {
		battery.CapacityLevel = level
	}

	// Read energy values (for energy-based batteries)
	if val, err := r.readUint64File(filepath.Join(devicePath, "energy_now")); err == nil {
		battery.EnergyNow = val
	}
	if val, err := r.readUint64File(filepath.Join(devicePath, "energy_full")); err == nil {
		battery.EnergyFull = val
	}
	if val, err := r.readUint64File(filepath.Join(devicePath, "energy_full_design")); err == nil {
		battery.EnergyFullDesign = val
	}

	// Read power and voltage
	if val, err := r.readUint64File(filepath.Join(devicePath, "power_now")); err == nil {
		battery.PowerNow = val
	}
	if val, err := r.readUint64File(filepath.Join(devicePath, "voltage_now")); err == nil {
		battery.VoltageNow = val
	}
	if val, err := r.readUint64File(filepath.Join(devicePath, "voltage_min_design")); err == nil {
		battery.VoltageMinDesign = val
	}

	// Read charge values (for charge-based batteries, alternative to energy values)
	// Conversion formula: energy (µWh) = charge (µAh) × voltage (µV) / 1000000
	// The division is by the constant 1000000, not by VoltageNow.
	if val, err := r.readUint64File(filepath.Join(devicePath, "charge_now")); err == nil {
		battery.ChargeNow = val
		// Convert to energy if energy values not available
		if battery.EnergyNow == 0 && battery.VoltageNow > 0 {
			battery.EnergyNow = val * battery.VoltageNow / 1000000
		}
	}
	if val, err := r.readUint64File(filepath.Join(devicePath, "charge_full")); err == nil {
		battery.ChargeFull = val
		if battery.EnergyFull == 0 && battery.VoltageNow > 0 {
			battery.EnergyFull = val * battery.VoltageNow / 1000000
		}
	}
	if val, err := r.readUint64File(filepath.Join(devicePath, "charge_full_design")); err == nil {
		battery.ChargeFullDesign = val
		if battery.EnergyFullDesign == 0 && battery.VoltageNow > 0 {
			battery.EnergyFullDesign = val * battery.VoltageNow / 1000000
		}
	}
	if val, err := r.readUint64File(filepath.Join(devicePath, "current_now")); err == nil {
		battery.CurrentNow = val
	}

	// Read battery info
	if tech, err := r.readStringFile(filepath.Join(devicePath, "technology")); err == nil {
		battery.Technology = tech
	}
	if mfr, err := r.readStringFile(filepath.Join(devicePath, "manufacturer")); err == nil {
		battery.Manufacturer = mfr
	}
	if model, err := r.readStringFile(filepath.Join(devicePath, "model_name")); err == nil {
		battery.ModelName = model
	}
	if serial, err := r.readStringFile(filepath.Join(devicePath, "serial_number")); err == nil {
		battery.SerialNumber = serial
	}

	// Read cycle count
	if cycles, err := r.readIntFile(filepath.Join(devicePath, "cycle_count")); err == nil {
		battery.CycleCount = int(cycles)
	}

	// Calculate health
	if battery.EnergyFullDesign > 0 {
		battery.Health = float64(battery.EnergyFull) / float64(battery.EnergyFullDesign) * 100
	}

	// Calculate time to empty/full
	if battery.PowerNow > 0 {
		if battery.Status == "Discharging" && battery.EnergyNow > 0 {
			// Time to empty in seconds
			battery.TimeToEmpty = float64(battery.EnergyNow) / float64(battery.PowerNow) * 3600
		} else if battery.Status == "Charging" && battery.EnergyFull > battery.EnergyNow {
			// Time to full in seconds
			remaining := battery.EnergyFull - battery.EnergyNow
			battery.TimeToFull = float64(remaining) / float64(battery.PowerNow) * 3600
		}
	}

	return battery, nil
}

// readACAdapter reads AC adapter information from a power supply device path.
func (r *batteryReader) readACAdapter(devicePath, name string) (ACAdapterInfo, error) {
	adapter := ACAdapterInfo{
		Name: name,
	}

	// Read online status
	if online, err := r.readIntFile(filepath.Join(devicePath, "online")); err == nil {
		adapter.Online = online == 1
	}

	return adapter, nil
}

// readStringFile reads a string value from a sysfs file.
func (r *batteryReader) readStringFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// readIntFile reads an integer value from a sysfs file.
func (r *batteryReader) readIntFile(path string) (int64, error) {
	str, err := r.readStringFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(str, 10, 64)
}

// readUint64File reads an unsigned integer value from a sysfs file.
func (r *batteryReader) readUint64File(path string) (uint64, error) {
	str, err := r.readStringFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(str, 10, 64)
}
