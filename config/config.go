package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the rate limiter
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Redis     RedisConfig     `mapstructure:"redis"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port string `mapstructure:"port"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	IPLimit     int                   `mapstructure:"ip_limit"`
	IPBlockTime time.Duration         `mapstructure:"ip_block_time"`
	TokenLimits map[string]TokenLimit `mapstructure:"token_limits"`
}

// TokenLimit holds configuration for a specific token
type TokenLimit struct {
	Limit     int           `mapstructure:"limit"`
	BlockTime time.Duration `mapstructure:"block_time"`
}

// LoadConfig loads configuration from environment variables and .env file
func LoadConfig() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set default values
	setDefaults()

	// Enable reading from environment variables
	viper.AutomaticEnv()

	// Try to read .env file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("Error reading config file: %v", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Load token configurations from environment variables
	config.RateLimit.TokenLimits = loadTokenConfigs()

	return &config, nil
}

// loadTokenConfigs loads token-specific configurations from environment variables
func loadTokenConfigs() map[string]TokenLimit {
	tokenConfigs := make(map[string]TokenLimit)

	// Get all environment variables
	envVars := viper.AllSettings()

	// Look for token configuration patterns
	for key, value := range envVars {
		if _, ok := value.(string); ok {
			// Check for token limit pattern: RATE_LIMIT_TOKEN_<TOKEN>_LIMIT
			if len(key) > 25 && key[:25] == "RATE_LIMIT_TOKEN_" && key[len(key)-6:] == "_LIMIT" {
				tokenName := key[25 : len(key)-6]

				// Get the limit value
				limit := viper.GetInt(key)

				// Get the block time for this token
				blockTimeKey := "RATE_LIMIT_TOKEN_" + tokenName + "_BLOCK_TIME"
				blockTimeStr := viper.GetString(blockTimeKey)

				var blockTime time.Duration
				if blockTimeStr != "" {
					var err error
					blockTime, err = time.ParseDuration(blockTimeStr)
					if err != nil {
						log.Printf("Invalid block time for token %s: %v", tokenName, err)
						blockTime = time.Minute // Default block time
					}
				} else {
					blockTime = time.Minute // Default block time
				}

				tokenConfigs[tokenName] = TokenLimit{
					Limit:     limit,
					BlockTime: blockTime,
				}
			}
		}
	}

	return tokenConfigs
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("SERVER_PORT", "8080")

	// Redis defaults
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)

	// Rate limit defaults
	viper.SetDefault("RATE_LIMIT_IP_LIMIT", 10)
	viper.SetDefault("RATE_LIMIT_IP_BLOCK_TIME", "1m")
}
