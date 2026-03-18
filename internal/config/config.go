// Package config handles layered configuration: flags, env vars, config file.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

const (
	DefaultNotionVersion = "2026-03-11"
	DefaultBaseURL       = "https://api.notion.com"
	DefaultTimeout       = 30 * time.Second
	DefaultRetry         = 3
)

// Config holds the resolved CLI configuration.
type Config struct {
	AuthToken      string
	ClientID       string
	ClientSecret   string
	NotionVersion  string
	BaseURL        string
	Timeout        time.Duration
	Retry          int
	OutputFormat   string // json, yaml, raw, pretty
	Quiet          bool
	IdempotencyKey string
	ExtraHeaders   map[string]string
	InputFile      string
	OutputFile     string
	Stdin          bool
}

// Load reads config from viper (flags + env + file) and returns a Config.
func Load() (*Config, error) {
	cfg := &Config{
		AuthToken:     viper.GetString("auth-token"),
		ClientID:      viper.GetString("client-id"),
		ClientSecret:  viper.GetString("client-secret"),
		NotionVersion: viper.GetString("notion-version"),
		BaseURL:       viper.GetString("base-url"),
		Timeout:       viper.GetDuration("timeout"),
		Retry:         viper.GetInt("retry"),
		Quiet:         viper.GetBool("quiet"),
		IdempotencyKey: viper.GetString("idempotency-key"),
		InputFile:     viper.GetString("input"),
		OutputFile:    viper.GetString("output"),
		Stdin:         viper.GetBool("stdin"),
	}

	// Resolve output format from convenience flags.
	switch {
	case viper.GetBool("json"):
		cfg.OutputFormat = "json"
	case viper.GetBool("yaml"):
		cfg.OutputFormat = "yaml"
	case viper.GetBool("raw"):
		cfg.OutputFormat = "raw"
	case viper.GetBool("pretty"):
		cfg.OutputFormat = "pretty"
	default:
		cfg.OutputFormat = "json"
	}

	// Defaults.
	if cfg.NotionVersion == "" {
		cfg.NotionVersion = DefaultNotionVersion
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.Retry == 0 {
		cfg.Retry = DefaultRetry
	}

	return cfg, nil
}

// InitViper sets up viper defaults, env binding, and config file search.
func InitViper() {
	viper.SetDefault("notion-version", DefaultNotionVersion)
	viper.SetDefault("base-url", DefaultBaseURL)
	viper.SetDefault("timeout", DefaultTimeout)
	viper.SetDefault("retry", DefaultRetry)

	viper.SetEnvPrefix("NOTION")
	viper.AutomaticEnv()

	// Bind common env vars.
	_ = viper.BindEnv("auth-token", "NOTION_AUTH_TOKEN", "NOTION_TOKEN")
	_ = viper.BindEnv("client-id", "NOTION_CLIENT_ID")
	_ = viper.BindEnv("client-secret", "NOTION_CLIENT_SECRET")

	// Config file search.
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".notion"))
	}
	viper.AddConfigPath(".")
	viper.SetConfigName("notion")
	viper.SetConfigType("yaml")

	// Silently ignore missing config file.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "warning: config file error: %v\n", err)
		}
	}
}
