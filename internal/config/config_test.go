package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg == nil {
		t.Fatal("Config is nil")
	}

	if cfg.Server.Port == "" {
		t.Error("Server port is empty")
	}

	if cfg.Redis.Host == "" {
		t.Error("Redis host is empty")
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		key          string
		defaultValue string
		expected     string
	}{
		{"NON_EXISTENT_VAR", "default", "default"},
		{"PATH", "", ""}, // PATH should exist on most systems
	}

	for _, test := range tests {
		result := getEnv(test.key, test.defaultValue)
		if test.expected == "" {
			// For PATH, just check it's not empty
			if test.key == "PATH" && result == "" {
				t.Errorf("Expected non-empty PATH")
			}
		} else if result != test.expected {
			t.Errorf("getEnv(%s, %s) = %s, want %s",
				test.key, test.defaultValue, result, test.expected)
		}
	}
}
