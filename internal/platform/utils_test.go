package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeMultiplyDivide(t *testing.T) {
	testCases := []struct {
		name     string
		a        uint64
		b        uint64
		divisor  uint64
		expected uint64
	}{
		{
			name:     "simple case",
			a:        100,
			b:        200,
			divisor:  10,
			expected: 2000,
		},
		{
			name:     "zero divisor returns 0",
			a:        100,
			b:        200,
			divisor:  0,
			expected: 0,
		},
		{
			name:     "zero a returns 0",
			a:        0,
			b:        200,
			divisor:  10,
			expected: 0,
		},
		{
			name:     "zero b returns 0",
			a:        100,
			b:        0,
			divisor:  10,
			expected: 0,
		},
		{
			name:     "large values without overflow",
			a:        1000000,
			b:        1000000,
			divisor:  1000000,
			expected: 1000000,
		},
		{
			name:     "battery charge calculation example",
			a:        5000000,   // charge in µAh
			b:        4200000,   // voltage in µV
			divisor:  1000000,   // convert to µWh
			expected: 21000000,  // 5000000 * 4200000 / 1000000 = 21000000000000 / 1000000 = 21000000
		},
		{
			name:     "values that would overflow in naive multiplication",
			a:        ^uint64(0) / 2,
			b:        3,
			divisor:  2,
			expected: 13835058055282163710, // Computed using slow path with precision preservation
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := safeMultiplyDivide(tc.a, tc.b, tc.divisor)
			if result != tc.expected {
				t.Errorf("safeMultiplyDivide(%d, %d, %d) = %d, want %d",
					tc.a, tc.b, tc.divisor, result, tc.expected)
			}
		})
	}
}

func TestSafeMultiplyDivide_PrecisionPreservation(t *testing.T) {
	// Test that the slow path preserves precision correctly
	// Using values that would overflow but can be computed with the slow path
	a := uint64(10000000000) // 10 billion
	b := uint64(2000000000)  // 2 billion
	divisor := uint64(1000000000)

	// Expected: (10B * 2B) / 1B = 20B
	// But 10B * 2B = 20,000,000,000,000,000,000 which overflows uint64
	// With the slow path: (10B / 1B) * 2B + (10B % 1B) * 2B / 1B
	//                   = 10 * 2B + 0 * 2B / 1B = 20B

	result := safeMultiplyDivide(a, b, divisor)
	expected := uint64(20000000000)

	if result != expected {
		t.Errorf("safeMultiplyDivide(%d, %d, %d) = %d, want %d",
			a, b, divisor, result, expected)
	}
}

func TestReadUint64File(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test_uint64")

	// Test successful read
	if err := os.WriteFile(testPath, []byte("12345\n"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	value, ok := readUint64File(testPath)
	if !ok {
		t.Error("readUint64File should return true for valid file")
	}
	if value != 12345 {
		t.Errorf("readUint64File = %d, want 12345", value)
	}

	// Test file not found
	value, ok = readUint64File(filepath.Join(tmpDir, "nonexistent"))
	if ok {
		t.Error("readUint64File should return false for nonexistent file")
	}
	if value != 0 {
		t.Errorf("readUint64File should return 0 for nonexistent file, got %d", value)
	}

	// Test invalid content
	if err := os.WriteFile(testPath, []byte("not a number\n"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	value, ok = readUint64File(testPath)
	if ok {
		t.Error("readUint64File should return false for invalid content")
	}
}

func TestReadStringFile(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test_string")

	// Test successful read
	if err := os.WriteFile(testPath, []byte("  hello world  \n"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	value, ok := readStringFile(testPath)
	if !ok {
		t.Error("readStringFile should return true for valid file")
	}
	if value != "hello world" {
		t.Errorf("readStringFile = %q, want %q", value, "hello world")
	}

	// Test file not found
	value, ok = readStringFile(filepath.Join(tmpDir, "nonexistent"))
	if ok {
		t.Error("readStringFile should return false for nonexistent file")
	}
	if value != "" {
		t.Errorf("readStringFile should return empty string for nonexistent file, got %q", value)
	}
}

func TestReadInt64File(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test_int64")

	// Test successful read with positive number
	if err := os.WriteFile(testPath, []byte("12345\n"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	value, ok := readInt64File(testPath)
	if !ok {
		t.Error("readInt64File should return true for valid file")
	}
	if value != 12345 {
		t.Errorf("readInt64File = %d, want 12345", value)
	}

	// Test negative number
	if err := os.WriteFile(testPath, []byte("-5000\n"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	value, ok = readInt64File(testPath)
	if !ok {
		t.Error("readInt64File should return true for negative number")
	}
	if value != -5000 {
		t.Errorf("readInt64File = %d, want -5000", value)
	}

	// Test file not found
	value, ok = readInt64File(filepath.Join(tmpDir, "nonexistent"))
	if ok {
		t.Error("readInt64File should return false for nonexistent file")
	}
}

func TestParseUint64(t *testing.T) {
	testCases := []struct {
		input    string
		expected uint64
	}{
		{"12345", 12345},
		{"0", 0},
		{"", 0},
		{"not a number", 0},
		{"18446744073709551615", 18446744073709551615}, // max uint64
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := parseUint64(tc.input)
			if result != tc.expected {
				t.Errorf("parseUint64(%q) = %d, want %d", tc.input, result, tc.expected)
			}
		})
	}
}
