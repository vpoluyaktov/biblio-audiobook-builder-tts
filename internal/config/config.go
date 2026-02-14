package config

import (
	"github.com/spf13/viper"
)

// Default database path
const DefaultDBPath = "./db/abb_tts.db"

// Config holds all configuration for the application
type Config struct {
	// Basic settings
	LogFile         string `mapstructure:"log_file"`
	TempDir         string `mapstructure:"temp_dir"`
	DefaultVoice    string `mapstructure:"default_voice"`
	DefaultProvider string `mapstructure:"default_provider"`

	// Server settings
	ServerPort  string `mapstructure:"server_port"`
	ServerHost  string `mapstructure:"server_host"`
	BasePath    string `mapstructure:"base_path"`
	OpenBrowser bool   `mapstructure:"open_browser"`

	// Authentication settings
	AuthMode      string `mapstructure:"auth_mode"`       // "internal" or "biblio-auth"
	BiblioAuthURL string `mapstructure:"biblio_auth_url"` // Biblio Auth service URL

	// TTS settings
	DefaultSpeed            float64 `mapstructure:"default_speed"`
	DefaultPitch            float64 `mapstructure:"default_pitch"`
	ChapterGapSeconds       float64 `mapstructure:"chapter_gap_seconds"`       // Silence between chapters (supports decimals like 2.5)
	PartGapSeconds          float64 `mapstructure:"part_gap_seconds"`          // Silence between parts (supports decimals)
	DetectPartSeparators    bool    `mapstructure:"detect_part_separators"`    // Enable part separator detection
	SentenceBreakMs         int     `mapstructure:"sentence_break_ms"`         // SSML break duration between sentences (default 300ms)
	ParagraphBreakMs        int     `mapstructure:"paragraph_break_ms"`        // SSML break duration between paragraphs (default 350ms)
	ConvertDashesToBreaks   bool    `mapstructure:"convert_dashes_to_breaks"`  // Convert inline dashes to SSML breaks
	DashBreakDurationMs     int     `mapstructure:"dash_break_duration_ms"`    // Duration of dash break in ms (default 250)
	ParenthesesBreakMs      int     `mapstructure:"parentheses_break_ms"`      // Duration of break around parenthetical text in ms (default 250)
	ColonBreakMs            int     `mapstructure:"colon_break_ms"`            // Duration of break replacing ':' in ms (default 250)
	TitleBreakMs            int     `mapstructure:"title_break_ms"`            // SSML break duration after titles (default 500ms)
	PronunciationDictFile   string  `mapstructure:"pronunciation_dict_file"`   // Path to pronunciation dictionary
	UseDefaultPronunciation bool    `mapstructure:"use_default_pronunciation"` // Use built-in pronunciation rules
	MaxFileSizeMB           int     `mapstructure:"max_file_size_mb"`          // Max M4B file size before splitting
	ConcurrentEncoders      int     `mapstructure:"concurrent_encoders"`       // Number of parallel M4B encoders

	// Audiobookshelf integration
	AudiobookshelfURL      string `mapstructure:"audiobookshelf_url"`
	AudiobookshelfUser     string `mapstructure:"audiobookshelf_user"`
	AudiobookshelfPassword string `mapstructure:"audiobookshelf_password"`
	AudiobookshelfLibrary  string `mapstructure:"audiobookshelf_library"`

	// Stress server integration (Russian text stress marking)
	StressServerURL string `mapstructure:"stress_server_url"`
}

// Load reads configuration from file and environment variables
func Load(configFile string) (*Config, error) {
	var config Config

	// Enable automatic environment variable binding with ABB_TTS_ prefix
	viper.SetEnvPrefix("ABB_TTS")
	viper.AutomaticEnv()

	// Explicitly bind environment variables for nested config
	viper.BindEnv("base_path")
	viper.BindEnv("server_port")
	viper.BindEnv("server_host")
	viper.BindEnv("auth_mode")
	viper.BindEnv("biblio_auth_url")
	viper.BindEnv("stress_server_url")

	// Basic settings
	viper.SetDefault("log_file", "biblio-audiobook-builder-tts.log")
	viper.SetDefault("temp_dir", "./temp")
	viper.SetDefault("default_voice", "en-US")
	viper.SetDefault("default_provider", "espeak")

	// Server settings
	viper.SetDefault("server_port", "8080")
	viper.SetDefault("server_host", "0.0.0.0")
	viper.SetDefault("base_path", "")
	viper.SetDefault("open_browser", true)

	// Authentication settings
	viper.SetDefault("auth_mode", "biblio-auth")
	viper.SetDefault("biblio_auth_url", "http://biblio-auth:80/auth")

	// TTS settings
	viper.SetDefault("default_speed", 1.0)
	viper.SetDefault("default_pitch", 1.0)
	viper.SetDefault("chapter_gap_seconds", 2.0)        // 2 seconds silence between chapters (supports decimals)
	viper.SetDefault("part_gap_seconds", 2.0)           // 2 seconds silence between parts (supports decimals)
	viper.SetDefault("detect_part_separators", true)    // Enable part separator detection by default
	viper.SetDefault("sentence_break_ms", 300)          // 300ms pause between sentences
	viper.SetDefault("paragraph_break_ms", 350)         // 350ms pause between paragraphs
	viper.SetDefault("convert_dashes_to_breaks", true)  // Convert inline dashes to SSML breaks by default
	viper.SetDefault("dash_break_duration_ms", 250)     // 250ms pause for dashes
	viper.SetDefault("parentheses_break_ms", 250)       // 250ms pause around parenthetical text
	viper.SetDefault("colon_break_ms", 250)             // 250ms pause replacing colons
	viper.SetDefault("title_break_ms", 500)             // 500ms pause after titles
	viper.SetDefault("pronunciation_dict_file", "")     // Custom pronunciation dictionary
	viper.SetDefault("use_default_pronunciation", true) // Use built-in pronunciation rules
	viper.SetDefault("max_file_size_mb", 250)           // 250MB max file size before splitting
	viper.SetDefault("concurrent_encoders", 2)          // 2 parallel M4B encoders

	// Audiobookshelf settings
	viper.SetDefault("audiobookshelf_url", "")
	viper.SetDefault("audiobookshelf_user", "admin")
	viper.SetDefault("audiobookshelf_password", "")
	viper.SetDefault("audiobookshelf_library", "TTS books")

	// Stress server settings
	viper.SetDefault("stress_server_url", "http://stress-silero:80/stress-silero")

	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("biblio-audiobook-builder-tts.config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.config/biblio-audiobook-builder-tts")
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
	if v, ok := dbConfig["default_speed"].(float64); ok {
		cfg.DefaultSpeed = v
	}
	if v, ok := dbConfig["default_pitch"].(float64); ok {
		cfg.DefaultPitch = v
	}
	if v, ok := dbConfig["chapter_gap_seconds"].(float64); ok {
		cfg.ChapterGapSeconds = v
	} else if v, ok := dbConfig["chapter_gap_seconds"].(int); ok {
		cfg.ChapterGapSeconds = float64(v)
	}
	if v, ok := dbConfig["part_gap_seconds"].(float64); ok {
		cfg.PartGapSeconds = v
	} else if v, ok := dbConfig["part_gap_seconds"].(int); ok {
		cfg.PartGapSeconds = float64(v)
	}
	if v, ok := dbConfig["detect_part_separators"].(bool); ok {
		cfg.DetectPartSeparators = v
	}
	if v, ok := dbConfig["sentence_break_ms"].(float64); ok {
		cfg.SentenceBreakMs = int(v)
	} else if v, ok := dbConfig["sentence_break_ms"].(int); ok {
		cfg.SentenceBreakMs = v
	}
	if v, ok := dbConfig["paragraph_break_ms"].(float64); ok {
		cfg.ParagraphBreakMs = int(v)
	} else if v, ok := dbConfig["paragraph_break_ms"].(int); ok {
		cfg.ParagraphBreakMs = v
	}
	if v, ok := dbConfig["convert_dashes_to_breaks"].(bool); ok {
		cfg.ConvertDashesToBreaks = v
	}
	if v, ok := dbConfig["dash_break_duration_ms"].(float64); ok {
		cfg.DashBreakDurationMs = int(v)
	} else if v, ok := dbConfig["dash_break_duration_ms"].(int); ok {
		cfg.DashBreakDurationMs = v
	}
	if v, ok := dbConfig["parentheses_break_ms"].(float64); ok {
		cfg.ParenthesesBreakMs = int(v)
	} else if v, ok := dbConfig["parentheses_break_ms"].(int); ok {
		cfg.ParenthesesBreakMs = v
	}
	if v, ok := dbConfig["colon_break_ms"].(float64); ok {
		cfg.ColonBreakMs = int(v)
	} else if v, ok := dbConfig["colon_break_ms"].(int); ok {
		cfg.ColonBreakMs = v
	}
	if v, ok := dbConfig["title_break_ms"].(float64); ok {
		cfg.TitleBreakMs = int(v)
	} else if v, ok := dbConfig["title_break_ms"].(int); ok {
		cfg.TitleBreakMs = v
	}
	if v, ok := dbConfig["pronunciation_dict_file"].(string); ok {
		cfg.PronunciationDictFile = v
	}
	if v, ok := dbConfig["use_default_pronunciation"].(bool); ok {
		cfg.UseDefaultPronunciation = v
	}
	if v, ok := dbConfig["max_file_size_mb"].(float64); ok {
		cfg.MaxFileSizeMB = int(v)
	} else if v, ok := dbConfig["max_file_size_mb"].(int); ok {
		cfg.MaxFileSizeMB = v
	}
	if v, ok := dbConfig["concurrent_encoders"].(float64); ok {
		cfg.ConcurrentEncoders = int(v)
	} else if v, ok := dbConfig["concurrent_encoders"].(int); ok {
		cfg.ConcurrentEncoders = v
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
	if v, ok := dbConfig["stress_server_url"].(string); ok {
		cfg.StressServerURL = v
	}

	return cfg
}
