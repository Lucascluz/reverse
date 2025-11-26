package config

import (
	"os"
	"testing"
)

// TestLoad tests loading a valid config file
func TestLoad(t *testing.T) {
	// Create temp config file
	content := `
proxy:
  targets:
    - "http://localhost:8081"
cache:
  disabled: false
  default_ttl: 5m
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Test loading
	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify loaded values
	if cfg.Proxy.Host != DefaultHost {
		t.Errorf("Expected default host %s, got %s", DefaultHost, cfg.Proxy.Host)
	}
}

// TestLoadInvalidFile tests error handling
func TestLoadInvalidFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error loading nonexistent file")
	}
}

// TestLoadInvalidYAML tests error handling for malformed YAML
func TestLoadInvalidYAML(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "bad-*.yaml")
	defer os.Remove(tmpfile.Name())
	
	tmpfile.Write([]byte("invalid: yaml: content: {{"))
	tmpfile.Close()

	_, err := Load(tmpfile.Name())
	if err == nil {
		t.Error("Expected error loading invalid YAML")
	}
}
