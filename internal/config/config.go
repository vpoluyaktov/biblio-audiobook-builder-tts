package config

import (
	"github.com/spf13/viper"
)

// Default database path
const DefaultDBPath = "abb_tts.db"

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
	BitRateKbs              int     `mapstructure:"bit_rate_kbs"`
	SampleRateHz            int     `mapstructure:"sample_rate_hz"`
	DefaultSpeed            float64 `mapstructure:"default_speed"`
	DefaultPitch            float64 `mapstructure:"default_pitch"`
	ChapterGapSeconds       int     `mapstructure:"chapter_gap_seconds"`       // Silence between chapters
	PronunciationDictFile   string  `mapstructure:"pronunciation_dict_file"`   // Path to pronunciation dictionary
	UseDefaultPronunciation bool    `mapstructure:"use_default_pronunciation"` // Use built-in pronunciation rules
	MaxFileSizeMB           int     `mapstructure:"max_file_size_mb"`          // Max M4B file size before splitting

	// Cloud provider settings
	CloudAPIKey       string `mapstructure:"cloud_api_key"`
	GoogleTTSEndpoint string `mapstructure:"google_tts_endpoint"`
	AzureTTSEndpoint  string `mapstructure:"azure_tts_endpoint"`
	GoogleAPIKey      string `mapstructure:"google_api_key"` // Google Cloud TTS API key

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
	viper.SetDefault("chapter_gap_seconds", 2)          // 2 seconds silence between chapters
	viper.SetDefault("pronunciation_dict_file", "")     // Custom pronunciation dictionary
	viper.SetDefault("use_default_pronunciation", true) // Use built-in pronunciation rules
	viper.SetDefault("max_file_size_mb", 2000)          // 2GB max file size before splitting

	// Cloud provider settings
	viper.SetDefault("cloud_api_key", "")
	viper.SetDefault("google_tts_endpoint", "")
	viper.SetDefault("azure_tts_endpoint", "")
	viper.SetDefault("google_api_key", "")

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

// LoadFromDB loads configuration from a storage.Config struct
func LoadFromDB(dbConfig map[string]interface{}) *Config {
	cfg := &Config{}

	if v, ok := dbConfig["log_file"].(string); ok {
		cfg.LogFile = v
	}
	if v, ok := dbConfig["output_dir"].(string); ok {
		cfg.OutputDir = v
	}
	if v, ok := dbConfig["temp_dir"].(string); ok {
		cfg.TempDir = v
	}
	if v, ok := dbConfig["default_voice"].(string); ok {
		cfg.DefaultVoice = v
	}
	if v, ok := dbConfig["default_provider"].(string); ok {
		cfg.DefaultProvider = v
	}
	if v, ok := dbConfig["server_port"].(string); ok {
		cfg.ServerPort = v
	}
	if v, ok := dbConfig["server_host"].(string); ok {
		cfg.ServerHost = v
	}
	if v, ok := dbConfig["open_browser"].(bool); ok {
		cfg.OpenBrowser = v
	}
	if v, ok := dbConfig["bit_rate_kbs"].(int); ok {
		cfg.BitRateKbs = v
	}
	if v, ok := dbConfig["sample_rate_hz"].(int); ok {
		cfg.SampleRateHz = v
	}
	if v, ok := dbConfig["default_speed"].(float64); ok {
		cfg.DefaultSpeed = v
	}
	if v, ok := dbConfig["default_pitch"].(float64); ok {
		cfg.DefaultPitch = v
	}
	if v, ok := dbConfig["chapter_gap_seconds"].(int); ok {
		cfg.ChapterGapSeconds = v
	}
	if v, ok := dbConfig["pronunciation_dict_file"].(string); ok {
		cfg.PronunciationDictFile = v
	}
	if v, ok := dbConfig["use_default_pronunciation"].(bool); ok {
		cfg.UseDefaultPronunciation = v
	}
	if v, ok := dbConfig["max_file_size_mb"].(int); ok {
		cfg.MaxFileSizeMB = v
	}
	if v, ok := dbConfig["cloud_api_key"].(string); ok {
		cfg.CloudAPIKey = v
	}
	if v, ok := dbConfig["google_tts_endpoint"].(string); ok {
		cfg.GoogleTTSEndpoint = v
	}
	if v, ok := dbConfig["azure_tts_endpoint"].(string); ok {
		cfg.AzureTTSEndpoint = v
	}
	if v, ok := dbConfig["google_api_key"].(string); ok {
		cfg.GoogleAPIKey = v
	}
	if v, ok := dbConfig["audiobookshelf_url"].(string); ok {
		cfg.AudiobookshelfURL = v
	}
	if v, ok := dbConfig["audiobookshelf_user"].(string); ok {
		cfg.AudiobookshelfUser = v
	}
	if v, ok := dbConfig["audiobookshelf_password"].(string); ok {
		cfg.AudiobookshelfPassword = v
	}
	if v, ok := dbConfig["audiobookshelf_library"].(string); ok {
		cfg.AudiobookshelfLibrary = v
	}

	return cfg
}
