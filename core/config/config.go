package config

import (
	"encoding/json"
	"log"
	"os"
)

const (
	defaultHost = "127.0.0.1"
	defaultPort = "9720"
)

type Config struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

func Load(path string) *Config {
	cfg := &Config{Host: defaultHost, Port: defaultPort}
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			log.Printf("config: failed to parse %s: %v, using defaults", path, err)
		}
	} else {
		log.Printf("config: %s not found, using defaults", path)
	}
	if h := os.Getenv("TOY_HOST"); h != "" {
		cfg.Host = h
	}
	if p := os.Getenv("TOY_PORT"); p != "" {
		cfg.Port = p
	}
	return cfg
}

func (c *Config) Address() string {
	return c.Host + ":" + c.Port
}
