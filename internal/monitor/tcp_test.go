package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTCPReader(t *testing.T) {
	t.Run("NewTCPReader", func(t *testing.T) {
		r := newTCPReader()
		if r == nil {
			t.Fatal("expected non-nil reader")
		}
		if r.procTCPPath != "/proc/net/tcp" {
			t.Errorf("unexpected path: %s", r.procTCPPath)
		}
	})
}

func TestParseTCPLine(t *testing.T) {
	r := newTCPReader()

	tests := []struct {
		name       string
		line       string
		wantLocal  string
		wantLPort  int
		wantRemote string
		wantRPort  int
		wantState  string
		wantErr    bool
	}{
		{
			name:       "established connection",
			line:       "   0: 0100007F:0050 0100007F:C000 01 00000000:00000000 00:00000000 00000000  1000        0 12345 1 0000000000000000 100 0 0 10 0",
			wantLocal:  "127.0.0.1",
			wantLPort:  80,
			wantRemote: "127.0.0.1",
			wantRPort:  49152,
			wantState:  "ESTABLISHED",
			wantErr:    false,
		},
		{
			name:       "listening socket",
			line:       "   1: 00000000:0016 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 54321 1 0000000000000000 100 0 0 10 0",
			wantLocal:  "0.0.0.0",
			wantLPort:  22,
			wantRemote: "0.0.0.0",
			wantRPort:  0,
			wantState:  "LISTEN",
			wantErr:    false,
		},
		{
			name:    "insufficient fields",
			line:    "   0: 0100007F:0050",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := r.parseTCPLine(tt.line, false)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if conn.LocalIP != tt.wantLocal {
				t.Errorf("LocalIP = %q, want %q", conn.LocalIP, tt.wantLocal)
			}
			if conn.LocalPort != tt.wantLPort {
				t.Errorf("LocalPort = %d, want %d", conn.LocalPort, tt.wantLPort)
			}
			if conn.RemoteIP != tt.wantRemote {
				t.Errorf("RemoteIP = %q, want %q", conn.RemoteIP, tt.wantRemote)
			}
			if conn.RemotePort != tt.wantRPort {
				t.Errorf("RemotePort = %d, want %d", conn.RemotePort, tt.wantRPort)
			}
			if conn.State != tt.wantState {
				t.Errorf("State = %q, want %q", conn.State, tt.wantState)
			}
		})
	}
}

func TestTCPReaderWithMockFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock /proc/net/tcp
	procTCPContent := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0050 0100007F:C000 01 00000000:00000000 00:00000000 00000000  1000        0 12345 1 0000000000000000 100 0 0 10 0
   1: 00000000:0016 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 54321 1 0000000000000000 100 0 0 10 0
`
	tcpPath := filepath.Join(tmpDir, "tcp")
	if err := os.WriteFile(tcpPath, []byte(procTCPContent), 0644); err != nil {
		t.Fatalf("failed to write mock file: %v", err)
	}

	r := newTCPReader()
	r.procTCPPath = tcpPath
	r.procTCP6Path = filepath.Join(tmpDir, "tcp6_nonexistent")

	stats, err := r.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats failed: %v", err)
	}

	if stats.TotalCount != 2 {
		t.Errorf("TotalCount = %d, want 2", stats.TotalCount)
	}
	if stats.ListenCount != 1 {
		t.Errorf("ListenCount = %d, want 1", stats.ListenCount)
	}
}

func TestCountInRange(t *testing.T) {
	tmpDir := t.TempDir()

	procTCPContent := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0050 0100007F:C000 01 00000000:00000000 00:00000000 00000000  1000        0 12345 1 0000000000000000 100 0 0 10 0
   1: 00000000:0016 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 54321 1 0000000000000000 100 0 0 10 0
   2: 0100007F:01BB 0100007F:D000 01 00000000:00000000 00:00000000 00000000  1000        0 12346 1 0000000000000000 100 0 0 10 0
`
	tcpPath := filepath.Join(tmpDir, "tcp")
	if err := os.WriteFile(tcpPath, []byte(procTCPContent), 0644); err != nil {
		t.Fatalf("failed to write mock file: %v", err)
	}

	r := newTCPReader()
	r.procTCPPath = tcpPath
	r.procTCP6Path = filepath.Join(tmpDir, "tcp6_nonexistent")

	// Count connections on ports 1-100
	count := r.CountInRange(1, 100)
	if count != 2 { // ports 80 and 22
		t.Errorf("CountInRange(1, 100) = %d, want 2", count)
	}

	// Count connections on ports 400-500
	count = r.CountInRange(400, 500)
	if count != 1 { // port 443
		t.Errorf("CountInRange(400, 500) = %d, want 1", count)
	}
}

func TestStateToString(t *testing.T) {
	r := newTCPReader()

	tests := []struct {
		state    string
		expected string
	}{
		{TCPEstablished, "ESTABLISHED"},
		{TCPListen, "LISTEN"},
		{TCPTimeWait, "TIME_WAIT"},
		{TCPClose, "CLOSE"},
		{"XX", "UNKNOWN"},
	}

	for _, tt := range tests {
		result := r.stateToString(tt.state)
		if result != tt.expected {
			t.Errorf("stateToString(%q) = %q, want %q", tt.state, result, tt.expected)
		}
	}
}
