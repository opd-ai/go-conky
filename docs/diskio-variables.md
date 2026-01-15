# Disk I/O Variables Implementation

## Overview

This document describes the implementation of disk I/O variables in conky-go, added on January 15, 2026, to address Gap #1 in AUDIT.md.

## Implemented Variables

### `${diskio}`
- **Purpose**: Returns total disk I/O speed (read + write) for a device or all devices
- **Usage**: 
  - `${diskio}` - Total I/O for all devices
  - `${diskio sda}` - Total I/O for specific device
- **Example Output**: `4.5MiB/s`, `1.5MiB/s`

### `${diskio_read}`
- **Purpose**: Returns disk read speed
- **Usage**:
  - `${diskio_read}` - Total read speed for all devices
  - `${diskio_read sda}` - Read speed for specific device
- **Example Output**: `3.0MiB/s`, `1.0MiB/s`

### `${diskio_write}`
- **Purpose**: Returns disk write speed
- **Usage**:
  - `${diskio_write}` - Total write speed for all devices
  - `${diskio_write sda}` - Write speed for specific device
- **Example Output**: `1.5MiB/s`, `512.0KiB/s`

## Technical Implementation

### Data Source
The variables use the existing `DiskIOStats` collected by `internal/monitor/diskio.go`:
- Reads from `/proc/diskstats` on Linux
- Tracks read/write bytes per second and operations per second
- Maintains statistics per device (sda, sdb, etc.)

### Speed Formatting
Uses the `formatSpeed()` function with automatic unit scaling:
- B/s for bytes per second < 1 KiB
- KiB/s for speeds 1 KiB - 1 MiB
- MiB/s for speeds 1 MiB - 1 GiB
- GiB/s for speeds >= 1 GiB

### Error Handling
- Returns `"0B/s"` for nonexistent devices
- Aggregates across all devices when no device specified
- Handles empty disk statistics gracefully

## Example Usage

```bash
# Traditional Conky configuration
TEXT
Disk I/O: ${diskio}
SSD Read: ${diskio_read sda}
HDD Write: ${diskio_write sdb}
```

```lua
-- Lua configuration
function conky_main()
    return string.format([[
Total I/O: %s
System Disk: %s read, %s write
]], 
    conky_parse("${diskio}"),
    conky_parse("${diskio_read sda}"),
    conky_parse("${diskio_write sda}")
    )
end
```

## Testing

Comprehensive tests added to `internal/lua/api_test.go`:
- Tests all three variables with and without device arguments
- Tests nonexistent device handling
- Verifies correct speed formatting and aggregation
- Uses mock data with realistic read/write speeds

## Impact

This implementation adds 3 commonly-used Conky variables, bringing the total from ~42 to ~45 implemented variables. These variables are essential for system monitoring configurations and complete the disk I/O monitoring capabilities alongside the existing filesystem variables.