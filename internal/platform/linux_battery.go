package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// linuxBatteryProvider implements BatteryProvider for Linux systems.
type linuxBatteryProvider struct {
	powerSupplyPath string
}

func newLinuxBatteryProvider() *linuxBatteryProvider {
	return &linuxBatteryProvider{
		powerSupplyPath: "/sys/class/power_supply",
	}
}

func (b *linuxBatteryProvider) Count() int {
	batteries, err := b.findBatteries()
	if err != nil {
		return 0
	}
	return len(batteries)
}

func (b *linuxBatteryProvider) Stats(index int) (*BatteryStats, error) {
	batteries, err := b.findBatteries()
	if err != nil {
		return nil, err
	}

	if index < 0 || index >= len(batteries) {
		return nil, fmt.Errorf("battery index %d out of range (0-%d)", index, len(batteries)-1)
	}

	batteryPath := filepath.Join(b.powerSupplyPath, batteries[index])
	return b.readBatteryStats(batteryPath)
}

// findBatteries returns a list of battery power supply names.
func (b *linuxBatteryProvider) findBatteries() ([]string, error) {
	entries, err := os.ReadDir(b.powerSupplyPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", b.powerSupplyPath, err)
	}

	var batteries []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		typePath := filepath.Join(b.powerSupplyPath, entry.Name(), "type")
		typeData, err := os.ReadFile(typePath)
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(typeData)) == "Battery" {
			batteries = append(batteries, entry.Name())
		}
	}

	return batteries, nil
}

// readBatteryStats reads battery statistics from sysfs.
func (b *linuxBatteryProvider) readBatteryStats(batteryPath string) (*BatteryStats, error) {
	stats := &BatteryStats{}

	// Read capacity (percentage)
	capacity, _ := b.readUint64File(filepath.Join(batteryPath, "capacity"))
	stats.Percent = float64(capacity)

	// Read status
	status, _ := b.readStringFile(filepath.Join(batteryPath, "status"))
	stats.Charging = strings.TrimSpace(status) == "Charging"

	// Read energy values (in µWh)
	energyNow, hasEnergy := b.readUint64File(filepath.Join(batteryPath, "energy_now"))
	energyFull, _ := b.readUint64File(filepath.Join(batteryPath, "energy_full"))

	// If energy_* files not available, try charge_* files (in µAh)
	if !hasEnergy {
		chargeNow, _ := b.readUint64File(filepath.Join(batteryPath, "charge_now"))
		chargeFull, _ := b.readUint64File(filepath.Join(batteryPath, "charge_full"))
		voltage, _ := b.readUint64File(filepath.Join(batteryPath, "voltage_now"))

		// Convert charge to energy: energy = charge * voltage
		if voltage > 0 {
			energyNow = chargeNow * voltage / 1000000 // Convert to µWh
			energyFull = chargeFull * voltage / 1000000
		}
	}

	stats.Current = energyNow
	stats.FullCapacity = energyFull

	// Read voltage
	voltage, _ := b.readUint64File(filepath.Join(batteryPath, "voltage_now"))
	stats.Voltage = float64(voltage) / 1000000.0 // Convert µV to V

	// Calculate time remaining
	powerNow, _ := b.readUint64File(filepath.Join(batteryPath, "power_now"))
	if powerNow > 0 {
		if stats.Charging {
			// Time to full
			remaining := energyFull - energyNow
			stats.TimeRemaining = time.Duration(remaining*3600/powerNow) * time.Second
		} else {
			// Time to empty
			stats.TimeRemaining = time.Duration(energyNow*3600/powerNow) * time.Second
		}
	}

	return stats, nil
}

// readUint64File reads a uint64 value from a sysfs file.
func (b *linuxBatteryProvider) readUint64File(path string) (uint64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}

	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, false
	}

	return value, true
}

// readStringFile reads a string value from a sysfs file.
func (b *linuxBatteryProvider) readStringFile(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}

	return strings.TrimSpace(string(data)), true
}
