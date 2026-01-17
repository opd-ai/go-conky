package monitor

import (
	"testing"
	"time"
)

func TestMailConfig(t *testing.T) {
	reader := newMailReader()

	// Test invalid configs
	tests := []struct {
		name    string
		config  MailConfig
		wantErr bool
	}{
		{
			name: "empty name",
			config: MailConfig{
				Name: "",
				Type: "imap",
				Host: "mail.example.com",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			config: MailConfig{
				Name: "test",
				Type: "smtp",
				Host: "mail.example.com",
			},
			wantErr: true,
		},
		{
			name: "empty host",
			config: MailConfig{
				Name: "test",
				Type: "imap",
				Host: "",
			},
			wantErr: true,
		},
		{
			name: "valid imap config",
			config: MailConfig{
				Name:     "test-imap",
				Type:     "imap",
				Host:     "imap.example.com",
				Port:     993,
				Username: "user",
				Password: "pass",
				UseTLS:   true,
			},
			wantErr: false,
		},
		{
			name: "valid pop3 config",
			config: MailConfig{
				Name:     "test-pop3",
				Type:     "pop3",
				Host:     "pop.example.com",
				Port:     995,
				Username: "user",
				Password: "pass",
				UseTLS:   true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reader.AddAccount(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMailReaderDefaults(t *testing.T) {
	reader := newMailReader()

	config := MailConfig{
		Name: "test",
		Type: "imap",
		Host: "imap.example.com",
	}

	err := reader.AddAccount(config)
	if err != nil {
		t.Fatalf("AddAccount() error = %v", err)
	}

	// Verify defaults are applied
	reader.mu.RLock()
	account := reader.accounts["test"]
	reader.mu.RUnlock()

	if account.config.Folder != "INBOX" {
		t.Errorf("Expected Folder to be 'INBOX', got %s", account.config.Folder)
	}

	if account.config.Interval < 60*time.Second {
		t.Errorf("Expected Interval >= 60s, got %v", account.config.Interval)
	}
}

func TestMailReaderRemoveAccount(t *testing.T) {
	reader := newMailReader()

	config := MailConfig{
		Name: "test",
		Type: "imap",
		Host: "imap.example.com",
	}

	err := reader.AddAccount(config)
	if err != nil {
		t.Fatalf("AddAccount() error = %v", err)
	}

	reader.RemoveAccount("test")

	reader.mu.RLock()
	_, ok := reader.accounts["test"]
	reader.mu.RUnlock()

	if ok {
		t.Error("Expected account to be removed")
	}
}

func TestMailStatsWithoutAccounts(t *testing.T) {
	reader := newMailReader()

	stats, err := reader.ReadStats()
	if err != nil {
		t.Errorf("ReadStats() error = %v", err)
	}

	if len(stats.Accounts) != 0 {
		t.Errorf("Expected 0 accounts, got %d", len(stats.Accounts))
	}

	// Test count methods with no accounts
	if reader.GetTotalUnseen() != 0 {
		t.Errorf("Expected 0 unseen, got %d", reader.GetTotalUnseen())
	}

	if reader.GetTotalMessages() != 0 {
		t.Errorf("Expected 0 messages, got %d", reader.GetTotalMessages())
	}
}

func TestGetAccountStatsNotFound(t *testing.T) {
	reader := newMailReader()

	stats, ok := reader.GetAccountStats("nonexistent")
	if ok {
		t.Error("Expected account not found")
	}

	if stats.Name != "" {
		t.Errorf("Expected empty stats, got Name=%s", stats.Name)
	}
}

func TestGetUnseenCountNotFound(t *testing.T) {
	reader := newMailReader()

	count := reader.GetUnseenCount("nonexistent")
	if count != 0 {
		t.Errorf("Expected 0, got %d", count)
	}
}

func TestParseIMAPExists(t *testing.T) {
	reader := &mailAccountReader{}

	tests := []struct {
		name     string
		response string
		expected int
	}{
		{
			name:     "typical response",
			response: "* 172 EXISTS\r\n* 1 RECENT\r\na002 OK [READ-WRITE] SELECT completed",
			expected: 172,
		},
		{
			name:     "zero messages",
			response: "* 0 EXISTS\r\n* 0 RECENT\r\na002 OK [READ-WRITE] SELECT completed",
			expected: 0,
		},
		{
			name:     "no exists line",
			response: "a002 OK SELECT completed",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reader.parseIMAPExists(tt.response)
			if result != tt.expected {
				t.Errorf("parseIMAPExists() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestParseIMAPSearchCount(t *testing.T) {
	reader := &mailAccountReader{}

	tests := []struct {
		name     string
		response string
		expected int
	}{
		{
			name:     "multiple unseen",
			response: "* SEARCH 2 4 7 10\r\na003 OK SEARCH completed",
			expected: 4,
		},
		{
			name:     "no unseen",
			response: "* SEARCH\r\na003 OK SEARCH completed",
			expected: 0,
		},
		{
			name:     "single unseen",
			response: "* SEARCH 5\r\na003 OK SEARCH completed",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reader.parseIMAPSearchCount(tt.response)
			if result != tt.expected {
				t.Errorf("parseIMAPSearchCount() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestMailAccountStatsCaching(t *testing.T) {
	reader := &mailAccountReader{
		config: MailConfig{
			Name:     "test",
			Type:     "imap",
			Host:     "invalid.local",
			Port:     993,
			Interval: 1 * time.Hour, // Long interval to ensure cache is used
		},
		cache: MailAccountStats{
			Name:      "test",
			Type:      "imap",
			Unseen:    5,
			Total:     10,
			LastCheck: time.Now(),
		},
		lastCheck:   time.Now(),
		dialTimeout: 1 * time.Second,
		readTimeout: 1 * time.Second,
	}

	// Should return cached stats
	stats := reader.getStats()

	if stats.Unseen != 5 {
		t.Errorf("Expected cached unseen=5, got %d", stats.Unseen)
	}

	if stats.Total != 10 {
		t.Errorf("Expected cached total=10, got %d", stats.Total)
	}
}

func TestMailReaderConcurrency(t *testing.T) {
	reader := newMailReader()

	// Add some accounts
	for i := 0; i < 5; i++ {
		config := MailConfig{
			Name:     "test-" + string(rune('a'+i)),
			Type:     "imap",
			Host:     "imap.example.com",
			Interval: 1 * time.Hour, // Long interval to avoid actual connections
		}
		if err := reader.AddAccount(config); err != nil {
			t.Fatalf("Failed to add account: %v", err)
		}
	}

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = reader.ReadStats()
			_ = reader.GetTotalUnseen()
			_ = reader.GetTotalMessages()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMailReaderAccountOverwrite(t *testing.T) {
	reader := newMailReader()

	// Add initial config
	config1 := MailConfig{
		Name:     "test",
		Type:     "imap",
		Host:     "imap1.example.com",
		Port:     993,
		Username: "user1",
	}
	if err := reader.AddAccount(config1); err != nil {
		t.Fatalf("AddAccount() error = %v", err)
	}

	// Add with same name should overwrite
	config2 := MailConfig{
		Name:     "test",
		Type:     "pop3",
		Host:     "pop.example.com",
		Port:     995,
		Username: "user2",
	}
	if err := reader.AddAccount(config2); err != nil {
		t.Fatalf("AddAccount() error = %v", err)
	}

	reader.mu.RLock()
	account := reader.accounts["test"]
	reader.mu.RUnlock()

	if account.config.Type != "pop3" {
		t.Errorf("Expected type to be 'pop3', got %s", account.config.Type)
	}

	if account.config.Host != "pop.example.com" {
		t.Errorf("Expected host to be 'pop.example.com', got %s", account.config.Host)
	}
}
