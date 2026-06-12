package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	Host string `json:"host"`
	Port string `json:"port"`
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
