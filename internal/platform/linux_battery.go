//go:build linux && !android
// +build linux,!android

package platform

import (
	"fmt"
	"os"
	"path/filepath"
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
	capacity, _ := readUint64File(filepath.Join(batteryPath, "capacity"))
	stats.Percent = float64(capacity)

	// Read status
	status, _ := readStringFile(filepath.Join(batteryPath, "status"))
	stats.Charging = strings.TrimSpace(status) == "Charging"

	// Read energy values (in µWh)
	energyNow, hasEnergy := readUint64File(filepath.Join(batteryPath, "energy_now"))
	energyFull, _ := readUint64File(filepath.Join(batteryPath, "energy_full"))

	// If energy_* files not available, try charge_* files (in µAh)
	if !hasEnergy {
		chargeNow, _ := readUint64File(filepath.Join(batteryPath, "charge_now"))
		chargeFull, _ := readUint64File(filepath.Join(batteryPath, "charge_full"))
		voltage, _ := readUint64File(filepath.Join(batteryPath, "voltage_now"))

		// Convert charge to energy: energy = charge * voltage
		if voltage > 0 {
			energyNow = chargeNow * voltage / 1000000 // Convert to µWh
			energyFull = chargeFull * voltage / 1000000
		}
	}

	stats.Current = energyNow
	stats.FullCapacity = energyFull

	// Read voltage
	voltage, _ := readUint64File(filepath.Join(batteryPath, "voltage_now"))
	stats.Voltage = float64(voltage) / 1000000.0 // Convert µV to V

	// Calculate time remaining
	powerNow, _ := readUint64File(filepath.Join(batteryPath, "power_now"))
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
