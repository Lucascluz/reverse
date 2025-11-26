package config

import (
	"testing"
	"time"
)

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    Config
		wantErr  bool
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name: "minimal config with only targets",
			input: Config{
				Proxy: ProxyConfig{
					Targets: []string{"http://localhost:8081"},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				// Check proxy defaults
				if cfg.Proxy.Host != DefaultHost {
					t.Errorf("Expected host %s, got %s", DefaultHost, cfg.Proxy.Host)
				}
				if cfg.Proxy.Port != DefaultPort {
					t.Errorf("Expected port %s, got %s", DefaultPort, cfg.Proxy.Port)
				}

				// Check cache defaults
				if cfg.Cache.Disabled != false {
					t.Error("Expected cache to be enabled by default (Disabled = false)")
				}
				if cfg.Cache.DefaultTTL != DefaultTTL {
					t.Errorf("Expected DefaultTTL %v, got %v", DefaultTTL, cfg.Cache.DefaultTTL)
				}
				if cfg.Cache.MaxAge != DefaultMaxAge {
					t.Errorf("Expected MaxAge %v, got %v", DefaultMaxAge, cfg.Cache.MaxAge)
				}
				if cfg.Cache.PurgeInterval != DefaultPurgeInterval {
					t.Errorf("Expected PurgeInterval %v, got %v", DefaultPurgeInterval, cfg.Cache.PurgeInterval)
				}
			},
		},
		{
			name: "config with explicit values",
			input: Config{
				Proxy: ProxyConfig{
					Host:    "0.0.0.0",
					Port:    "9090",
					Targets: []string{"http://localhost:8081"},
				},
				Cache: CacheConfig{
					Disabled: true,
					DefaultTTL:    10 * time.Minute,
					MaxAge:        2 * time.Hour,
					PurgeInterval: 5 * time.Minute,
				},
			},
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				// Should keep explicit values
				if cfg.Proxy.Host != "0.0.0.0" {
					t.Errorf("Expected host 0.0.0.0, got %s", cfg.Proxy.Host)
				}
				if cfg.Cache.Disabled != false {
					t.Error("Expected cache to be disabled")
				}
				if cfg.Cache.DefaultTTL != 10*time.Minute {
					t.Errorf("Expected DefaultTTL 10m, got %v", cfg.Cache.DefaultTTL)
				}
			},
		},
		{
			name: "config without targets should fail",
			input: Config{
				Proxy: ProxyConfig{
					Host: "localhost",
					Port: "8080",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.applyDefaults()

			if (err != nil) != tt.wantErr {
				t.Errorf("applyDefaults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil && err == nil {
				tt.validate(t, &tt.input)
			}
		})
	}
}