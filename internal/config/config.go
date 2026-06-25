package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	GitHub GitHubConfig `mapstructure:"github"`
	Paths  PathsConfig  `mapstructure:"paths"`
	Log    LogConfig    `mapstructure:"log"`
}

type GitHubConfig struct {
	ClientID     string `mapstructure:"client_id"`
	RegistryRepo string `mapstructure:"registry_repo"`
	DefaultOwner string `mapstructure:"default_owner"`
}

type PathsConfig struct {
	PluginsDir string `mapstructure:"plugins_dir"`
	ThemesDir  string `mapstructure:"themes_dir"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

var (
	ErrConfigNotFound = errors.New("config file not found")
	ErrInvalidConfig  = errors.New("invalid config format")
)

func Load() (*Config, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(configDir)

	viper.SetDefault("github.client_id", "")
	viper.SetDefault("github.registry_repo", "cortex/registry")
	viper.SetDefault("github.default_owner", "")
	viper.SetDefault("paths.plugins_dir", getDefaultPluginsDir())
	viper.SetDefault("paths.themes_dir", getDefaultThemesDir())
	viper.SetDefault("log.level", "info")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return loadDefaults(), nil
		}
		return nil, ErrInvalidConfig
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, ErrInvalidConfig
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	viper.Set("github", cfg.GitHub)
	viper.Set("paths", cfg.Paths)
	viper.Set("log", cfg.Log)

	configPath := filepath.Join(configDir, "config.json")
	return viper.WriteConfigAs(configPath)
}

func loadDefaults() *Config {
	return &Config{
		GitHub: GitHubConfig{
			RegistryRepo: "cortex/registry",
		},
		Paths: PathsConfig{
			PluginsDir: getDefaultPluginsDir(),
			ThemesDir:  getDefaultThemesDir(),
		},
		Log: LogConfig{
			Level: "info",
		},
	}
}

func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "cortex"), nil
}

func getDefaultPluginsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cortex", "plugins")
}

func getDefaultThemesDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cortex", "themes")
}

func GetCacheDir() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "cache"), nil
}
