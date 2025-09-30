package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ServerCfg struct {
	Addr       string `yaml:"addr"`
	AdminToken string `yaml:"admin_token"`
}
type PostgresCfg struct {
	DSN string `yaml:"dsn"`
}
type RedisCfg struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}
type SecurityCfg struct {
	TokenTTLSeconds int `yaml:"token_ttl_seconds"`
}

type Config struct {
	Server   ServerCfg   `yaml:"server"`
	Postgres PostgresCfg `yaml:"postgres"`
	Redis    RedisCfg    `yaml:"redis"`
	Security SecurityCfg `yaml:"security"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
