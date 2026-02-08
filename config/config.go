package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Reddit struct {
		ClientID     string
		ClientSecret string
	}
	Mistral struct {
		APIKey string
		Model  string
	}
	Database struct {
		Host     string
		Port     string
		User     string
		Password string
		DBName   string
		Type     string // "postgres" or "sqlite"
	}
	Crawler struct {
		Subreddits      []string
		PostLimit       int
		RateLimitSecs   int
		SharingKeywords []string
	}
}

func Load() (*Config, error) {
	v := viper.New()

	// Set config file name and paths
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	// Set defaults
	setDefaults(v)

	// Read config file (optional - will use env vars if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; using defaults and env vars
	}

	cfg := &Config{}

	// Reddit config
	cfg.Reddit.ClientID = v.GetString("reddit.client_id")
	cfg.Reddit.ClientSecret = v.GetString("reddit.client_secret")

	// Mistral config
	cfg.Mistral.APIKey = v.GetString("mistral.api_key")
	cfg.Mistral.Model = v.GetString("mistral.model")

	// Database config
	cfg.Database.Type = v.GetString("database.type")
	cfg.Database.Host = v.GetString("database.host")
	cfg.Database.Port = v.GetString("database.port")
	cfg.Database.User = v.GetString("database.user")
	cfg.Database.Password = v.GetString("database.password")
	cfg.Database.DBName = v.GetString("database.dbname")

	// Crawler config
	cfg.Crawler.Subreddits = v.GetStringSlice("crawler.subreddits")
	cfg.Crawler.PostLimit = v.GetInt("crawler.post_limit")
	cfg.Crawler.RateLimitSecs = v.GetInt("crawler.rate_limit_secs")
	cfg.Crawler.SharingKeywords = v.GetStringSlice("crawler.sharing_keywords")

	// Validate required fields
	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Database defaults
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", "5432")
	v.SetDefault("database.dbname", "ideas.db")

	// Crawler defaults
	v.SetDefault("crawler.subreddits", []string{
		"SideProject",
		"Entrepreneur",
		"startups",
		"Business_Ideas",
		"roastmystartup",
	})
	v.SetDefault("crawler.post_limit", 25)
	v.SetDefault("crawler.rate_limit_secs", 2)
	v.SetDefault("crawler.sharing_keywords", []string{
		"share what you're building",
		"share what you are building",
		"show off your project",
		"what are you working on",
		"showcase",
		"share your startup",
		"show your side project",
	})
}

func validate(cfg *Config) error {
	if cfg.Reddit.ClientID == "" {
		return fmt.Errorf("reddit.client_id is required")
	}
	if cfg.Reddit.ClientSecret == "" {
		return fmt.Errorf("reddit.client_secret is required")
	}
	if cfg.Mistral.APIKey == "" {
		return fmt.Errorf("mistral.api_key is required")
	}
	if cfg.Database.DBName == "" {
		return fmt.Errorf("database.dbname is required")
	}
	return nil
}
