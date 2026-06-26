package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	Host             string     `json:"host"`
	Port             string     `json:"port"`
	Engine           string     `json:"engine,omitempty"`
	Log              *LogConfig `json:"log,omitempty"`
	JWTSecret        []byte     `json:"jwt_secret,omitempty"`
	TokenExpireHours int        `json:"token_expire_hours,omitempty"`
	RedisAddr        string     `json:"redis_addr,omitempty"`
	RedisPassword    string     `json:"redis_password,omitempty"`
	RedisDB          int        `json:"redis_db,omitempty"`
	DBDriver         string     `json:"db_driver,omitempty"`
	SQLitePath       string     `json:"sqlite_path,omitempty"`
	MySQLDSN         string     `json:"mysql_dsn,omitempty"`
	PostgresDSN      string     `json:"postgres_dsn,omitempty"`
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
