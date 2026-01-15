//go:build android
// +build android

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// androidBatteryProvider implements BatteryProvider for Android systems.
// Android uses sysfs power_supply interface similar to Linux.
type androidBatteryProvider struct {
	powerSupplyPath string
}

func newAndroidBatteryProvider() *androidBatteryProvider {
	return &androidBatteryProvider{
		powerSupplyPath: "/sys/class/power_supply",
	}
}

func (b *androidBatteryProvider) Count() int {
	batteries, err := b.findBatteries()
	if err != nil {
		return 0
	}
	return len(batteries)
}

func (b *androidBatteryProvider) Stats(index int) (*BatteryStats, error) {
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
func (b *androidBatteryProvider) findBatteries() ([]string, error) {
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
func (b *androidBatteryProvider) readBatteryStats(batteryPath string) (*BatteryStats, error) {
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
	// Android often uses charge_* instead of energy_*
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

	// Android may also provide charge_counter (µAh)
	if energyNow == 0 {
		chargeCounter, _ := readUint64File(filepath.Join(batteryPath, "charge_counter"))
		chargeFull, _ := readUint64File(filepath.Join(batteryPath, "charge_full_design"))
		voltage, _ := readUint64File(filepath.Join(batteryPath, "voltage_now"))

		if voltage > 0 && chargeCounter > 0 {
			energyNow = chargeCounter * voltage / 1000000
			energyFull = chargeFull * voltage / 1000000
		}
	}

	stats.Current = energyNow
	stats.FullCapacity = energyFull

	// Read voltage
	voltage, _ := readUint64File(filepath.Join(batteryPath, "voltage_now"))
	stats.Voltage = float64(voltage) / 1000000.0 // Convert µV to V

	// Calculate time remaining
	// Android may provide current_now instead of power_now
	powerNow, hasPower := readUint64File(filepath.Join(batteryPath, "power_now"))
	if !hasPower {
		currentNow, _ := readUint64File(filepath.Join(batteryPath, "current_now"))
		if currentNow > 0 && voltage > 0 {
			// Power = Current * Voltage (both in µA and µV)
			powerNow = currentNow * voltage / 1000000
		}
	}

	if powerNow > 0 {
		if stats.Charging {
			// Time to full
			var remaining uint64
			if energyFull >= energyNow {
				remaining = energyFull - energyNow
			}
			stats.TimeRemaining = time.Duration(remaining*3600/powerNow) * time.Second
		} else {
			// Time to empty
			stats.TimeRemaining = time.Duration(energyNow*3600/powerNow) * time.Second
		}
	}

	return stats, nil
}
