package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	LastVault string `json:"lastVault,omitempty"`
}

func Dir() (string, error) {
	if custom := os.Getenv("CORTEX_CONFIG_DIR"); custom != "" {
		return custom, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "cortex"), nil
}

func Load() Config {
	dir, err := Dir()
	if err != nil {
		return Config{}
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return Config{}
	}
	var cfg Config
	if json.Unmarshal(data, &cfg) != nil {
		return Config{}
	}
	return cfg
}

func Save(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), append(data, '\n'), 0o644)
}

func RememberVault(path string) error {
	if path == "" {
		return errors.New("vault path is empty")
	}
	cfg := Load()
	cfg.LastVault = path
	return Save(cfg)
}
