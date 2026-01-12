package config

import (
	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Basic settings
	LogFile         string `mapstructure:"log_file"`
	OutputDir       string `mapstructure:"output_dir"`
	TempDir         string `mapstructure:"temp_dir"`
	DefaultVoice    string `mapstructure:"default_voice"`
	DefaultProvider string `mapstructure:"default_provider"`

	// Server settings
	ServerPort  string `mapstructure:"server_port"`
	ServerHost  string `mapstructure:"server_host"`
	OpenBrowser bool   `mapstructure:"open_browser"`

	// TTS settings
	BitRateKbs   int     `mapstructure:"bit_rate_kbs"`
	SampleRateHz int     `mapstructure:"sample_rate_hz"`
	DefaultSpeed float64 `mapstructure:"default_speed"`
	DefaultPitch float64 `mapstructure:"default_pitch"`

	// Cloud provider settings
	CloudAPIKey       string `mapstructure:"cloud_api_key"`
	GoogleTTSEndpoint string `mapstructure:"google_tts_endpoint"`
	AzureTTSEndpoint  string `mapstructure:"azure_tts_endpoint"`

	// Audiobookshelf integration
	AudiobookshelfURL      string `mapstructure:"audiobookshelf_url"`
	AudiobookshelfUser     string `mapstructure:"audiobookshelf_user"`
	AudiobookshelfPassword string `mapstructure:"audiobookshelf_password"`
	AudiobookshelfLibrary  string `mapstructure:"audiobookshelf_library"`
}

// Load reads configuration from file and environment variables
func Load(configFile string) (*Config, error) {
	var config Config

	// Basic settings
	viper.SetDefault("log_file", "abb_tts.log")
	viper.SetDefault("output_dir", "./output")
	viper.SetDefault("temp_dir", "./temp")
	viper.SetDefault("default_voice", "en-US")
	viper.SetDefault("default_provider", "espeak")

	// Server settings
	viper.SetDefault("server_port", "8080")
	viper.SetDefault("server_host", "0.0.0.0")
	viper.SetDefault("open_browser", true)

	// TTS settings
	viper.SetDefault("bit_rate_kbs", 128)
	viper.SetDefault("sample_rate_hz", 44100)
	viper.SetDefault("default_speed", 1.0)
	viper.SetDefault("default_pitch", 1.0)

	// Cloud provider settings
	viper.SetDefault("cloud_api_key", "")
	viper.SetDefault("google_tts_endpoint", "")
	viper.SetDefault("azure_tts_endpoint", "")

	// Audiobookshelf settings
	viper.SetDefault("audiobookshelf_url", "")
	viper.SetDefault("audiobookshelf_user", "admin")
	viper.SetDefault("audiobookshelf_password", "")
	viper.SetDefault("audiobookshelf_library", "TTS Books")

	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("abb_tts.config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.config/abb_tts")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
