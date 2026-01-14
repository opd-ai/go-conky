//go:build windows
// +build windows

package platform

import (
	"fmt"
	"time"
	"unsafe"
)

var modKernel32GetSystemPowerStatus = modKernel32.NewProc("GetSystemPowerStatus")

const (
	AC_LINE_OFFLINE      = 0x00
	AC_LINE_ONLINE       = 0x01
	AC_LINE_BACKUP_POWER = 0x02
	AC_LINE_UNKNOWN      = 0xFF

	BATTERY_FLAG_HIGH       = 0x01
	BATTERY_FLAG_LOW        = 0x02
	BATTERY_FLAG_CRITICAL   = 0x04
	BATTERY_FLAG_CHARGING   = 0x08
	BATTERY_FLAG_NO_BATTERY = 0x80
	BATTERY_FLAG_UNKNOWN    = 0xFF

	BATTERY_PERCENTAGE_UNKNOWN = 255
	BATTERY_LIFE_UNKNOWN       = 0xFFFFFFFF
)

// systemPowerStatus matches the Windows SYSTEM_POWER_STATUS structure
type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

// windowsBatteryProvider implements BatteryProvider for Windows systems
type windowsBatteryProvider struct{}

func newWindowsBatteryProvider() *windowsBatteryProvider {
	return &windowsBatteryProvider{}
}

func (b *windowsBatteryProvider) getPowerStatus() (*systemPowerStatus, error) {
	var status systemPowerStatus
	ret, _, err := modKernel32GetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&status)))
	if ret == 0 {
		return nil, fmt.Errorf("GetSystemPowerStatus failed: %w", err)
	}
	return &status, nil
}

func (b *windowsBatteryProvider) Count() int {
	status, err := b.getPowerStatus()
	if err != nil {
		return 0
	}

	// If no battery flag is set, there's no battery
	if status.BatteryFlag&BATTERY_FLAG_NO_BATTERY != 0 {
		return 0
	}

	// Windows doesn't directly report multiple batteries via this API
	// For simplicity, we report 1 battery if any battery is present
	return 1
}

func (b *windowsBatteryProvider) Stats(index int) (*BatteryStats, error) {
	if index != 0 {
		return nil, fmt.Errorf("battery index %d out of range", index)
	}

	status, err := b.getPowerStatus()
	if err != nil {
		return nil, err
	}

	// Check if battery is present
	if status.BatteryFlag&BATTERY_FLAG_NO_BATTERY != 0 {
		return nil, fmt.Errorf("no battery present")
	}

	// Determine if charging
	charging := false
	if status.ACLineStatus == AC_LINE_ONLINE {
		charging = true
	}
	// Additional check: if battery flag indicates charging
	if status.BatteryFlag&BATTERY_FLAG_CHARGING != 0 {
		charging = true
	}

	// Get battery percentage
	var percent float64
	if status.BatteryLifePercent != BATTERY_PERCENTAGE_UNKNOWN {
		percent = float64(status.BatteryLifePercent)
	}

	// Get time remaining
	var timeRemaining time.Duration
	if status.BatteryLifeTime != BATTERY_LIFE_UNKNOWN {
		timeRemaining = time.Duration(status.BatteryLifeTime) * time.Second
	}

	return &BatteryStats{
		Percent:       percent,
		TimeRemaining: timeRemaining,
		Charging:      charging,
		FullCapacity:  0, // Not available through this API
		Current:       0, // Not available through this API
		Voltage:       0, // Not available through this API
	}, nil
}
