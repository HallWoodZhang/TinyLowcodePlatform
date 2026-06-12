package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()

	validCfg := filepath.Join(tmpDir, "valid.json")
	os.WriteFile(validCfg, []byte(`{"host":"10.0.0.1","port":"8080"}`), 0644)

	badCfg := filepath.Join(tmpDir, "bad.json")
	os.WriteFile(badCfg, []byte(`{invalid`), 0644)

	tests := []struct {
		name         string
		path         string
		envHost      string
		envPort      string
		defaultHost  string
		defaultPort  string
		setEnvHost   string
		setEnvPort   string
		wantHost     string
		wantPort     string
		wantAddress  string
	}{
		{
			name:     "defaults when file missing",
			path:     filepath.Join(tmpDir, "missing.json"),
			wantHost: "127.0.0.1", wantPort: "9000", wantAddress: "127.0.0.1:9000",
			defaultHost: "127.0.0.1", defaultPort: "9000",
		},
		{
			name:     "load from valid JSON",
			path:     validCfg,
			wantHost: "10.0.0.1", wantPort: "8080", wantAddress: "10.0.0.1:8080",
			defaultHost: "0.0.0.0", defaultPort: "9999",
		},
		{
			name:     "bad JSON falls back to defaults",
			path:     badCfg,
			wantHost: "1.2.3.4", wantPort: "5555", wantAddress: "1.2.3.4:5555",
			defaultHost: "1.2.3.4", defaultPort: "5555",
		},
		{
			name:     "env host overrides file",
			path:     validCfg,
			envHost:  "MY_HOST", envPort: "MY_PORT",
			setEnvHost: "192.168.1.1",
			wantHost: "192.168.1.1", wantPort: "8080", wantAddress: "192.168.1.1:8080",
			defaultHost: "0.0.0.0", defaultPort: "9999",
		},
		{
			name:     "env port overrides file",
			path:     validCfg,
			envHost:  "MY_HOST", envPort: "MY_PORT",
			setEnvPort: "7070",
			wantHost: "10.0.0.1", wantPort: "7070", wantAddress: "10.0.0.1:7070",
			defaultHost: "0.0.0.0", defaultPort: "9999",
		},
		{
			name:     "env overrides both file and defaults",
			path:     validCfg,
			envHost:  "MY_HOST", envPort: "MY_PORT",
			setEnvHost: "10.10.10.10", setEnvPort: "1234",
			wantHost: "10.10.10.10", wantPort: "1234", wantAddress: "10.10.10.10:1234",
			defaultHost: "0.0.0.0", defaultPort: "9999",
		},
		{
			name:     "env overrides defaults when file missing",
			path:     filepath.Join(tmpDir, "nope.json"),
			envHost:  "H", envPort: "P",
			setEnvHost: "::1", setEnvPort: "9090",
			wantHost: "::1", wantPort: "9090", wantAddress: "::1:9090",
			defaultHost: "0.0.0.0", defaultPort: "1",
		},
		{
			name:     "empty env does not override",
			path:     validCfg,
			envHost:  "EMPTY_HOST", envPort: "EMPTY_PORT",
			// don't set these env vars — they remain unset
			wantHost: "10.0.0.1", wantPort: "8080", wantAddress: "10.0.0.1:8080",
			defaultHost: "0.0.0.0", defaultPort: "9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnvHost != "" {
				os.Setenv(tt.envHost, tt.setEnvHost)
				defer os.Unsetenv(tt.envHost)
			}
			if tt.setEnvPort != "" {
				os.Setenv(tt.envPort, tt.setEnvPort)
				defer os.Unsetenv(tt.envPort)
			}
			if tt.envHost == "" {
				tt.envHost = "UNUSED_HOST"
			}
			if tt.envPort == "" {
				tt.envPort = "UNUSED_PORT"
			}

			cfg := Load(tt.path, tt.envHost, tt.envPort, tt.defaultHost, tt.defaultPort)

			if cfg.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", cfg.Host, tt.wantHost)
			}
			if cfg.Port != tt.wantPort {
				t.Errorf("Port = %q, want %q", cfg.Port, tt.wantPort)
			}
			if cfg.Address() != tt.wantAddress {
				t.Errorf("Address() = %q, want %q", cfg.Address(), tt.wantAddress)
			}
		})
	}
}
