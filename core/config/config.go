package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	Host string    `json:"host"`
	Port string    `json:"port"`
	Log  *LogConfig `json:"log,omitempty"`
}

type LogConfig struct {
	Debug  string `json:"debug"`
	Access string `json:"access"`
	Panic  string `json:"panic"`
}

func Load(path, envHost, envPort, defaultHost, defaultPort string) *Config {
	cfg := &Config{Host: defaultHost, Port: defaultPort}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			log.Printf("config: failed to parse %s: %v, using defaults", path, err)
		}
	} else {
		log.Printf("config: %s not found, using defaults", path)
	}
	if h := os.Getenv(envHost); h != "" {
		cfg.Host = h
	}
	if p := os.Getenv(envPort); p != "" {
		cfg.Port = p
	}
	return cfg
}

func (c *Config) Address() string {
	return c.Host + ":" + c.Port
}
