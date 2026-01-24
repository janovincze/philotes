package postgres

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Name != "postgres" {
		t.Errorf("Name = %q, want %q", cfg.Name, "postgres")
	}
	if cfg.SlotName != "philotes_cdc" {
		t.Errorf("SlotName = %q, want %q", cfg.SlotName, "philotes_cdc")
	}
	if cfg.PublicationName != "philotes_pub" {
		t.Errorf("PublicationName = %q, want %q", cfg.PublicationName, "philotes_pub")
	}
	if cfg.ReconnectInterval != 5*time.Second {
		t.Errorf("ReconnectInterval = %v, want %v", cfg.ReconnectInterval, 5*time.Second)
	}
	if cfg.MaxReconnectAttempts != 0 {
		t.Errorf("MaxReconnectAttempts = %d, want %d", cfg.MaxReconnectAttempts, 0)
	}
	if cfg.EventBufferSize != 1000 {
		t.Errorf("EventBufferSize = %d, want %d", cfg.EventBufferSize, 1000)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name: "valid config",
			config: Config{
				ConnectionURL:   "postgres://user:pass@localhost:5432/db",
				SlotName:        "test_slot",
				PublicationName: "test_pub",
			},
			wantErr: nil,
		},
		{
			name: "missing connection URL",
			config: Config{
				SlotName:        "test_slot",
				PublicationName: "test_pub",
			},
			wantErr: ErrMissingConnectionURL,
		},
		{
			name: "missing slot name",
			config: Config{
				ConnectionURL:   "postgres://user:pass@localhost:5432/db",
				PublicationName: "test_pub",
			},
			wantErr: ErrMissingSlotName,
		},
		{
			name: "missing publication name",
			config: Config{
				ConnectionURL: "postgres://user:pass@localhost:5432/db",
				SlotName:      "test_slot",
			},
			wantErr: ErrMissingPublicationName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
