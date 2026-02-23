package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// AgentConfigFile представляет конфигурацию агента из JSON файла
type AgentConfigFile struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	RateLimit      int64  `json:"rate_limit"`
	Key            string `json:"key"`
	CryptoKey      string `json:"crypto_key"`
}

// ServerConfigFile представляет конфигурацию сервера из JSON файла
type ServerConfigFile struct {
	Address       string `json:"address"`
	Restore       bool   `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	Key           string `json:"key"`
	AuditFile     string `json:"audit_file"`
	AuditURL      string `json:"audit_url"`
	CryptoKey     string `json:"crypto_key"`
}

// LoadAgentConfigFile загружает конфигурацию агента из JSON файла
func LoadAgentConfigFile(path string) (*AgentConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg AgentConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadServerConfigFile загружает конфигурацию сервера из JSON файла
func LoadServerConfigFile(path string) (*ServerConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ServerConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ParseDuration парсит строку длительности в секунды
func ParseDuration(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return int64(d.Seconds()), nil
}

// ParseDurationUint парсит строку длительности в uint64 секунды
func ParseDurationUint(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return uint64(d.Seconds()), nil
}

// Validate проверяет валидность конфигурации агента из файла
func (cfg *AgentConfigFile) Validate() error {
	if cfg.Address != "" {
		ha := NewHostAddress()
		if err := ha.Set(cfg.Address); err != nil {
			return fmt.Errorf("invalid address: %w", err)
		}
	}
	if cfg.ReportInterval != "" {
		if _, err := time.ParseDuration(cfg.ReportInterval); err != nil {
			return fmt.Errorf("invalid report_interval: %w", err)
		}
	}
	if cfg.PollInterval != "" {
		if _, err := time.ParseDuration(cfg.PollInterval); err != nil {
			return fmt.Errorf("invalid poll_interval: %w", err)
		}
	}
	return nil
}

// Validate проверяет валидность конфигурации сервера из файла
func (cfg *ServerConfigFile) Validate() error {
	if cfg.Address != "" {
		ha := NewHostAddress()
		if err := ha.Set(cfg.Address); err != nil {
			return fmt.Errorf("invalid address: %w", err)
		}
	}
	if cfg.StoreInterval != "" {
		if _, err := time.ParseDuration(cfg.StoreInterval); err != nil {
			return fmt.Errorf("invalid store_interval: %w", err)
		}
	}
	return nil
}
