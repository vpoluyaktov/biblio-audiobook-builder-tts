package app

import (
	"abb_tts/internal/config"
	"abb_tts/internal/parser"
	"abb_tts/internal/tts"
)

// App represents the main application
type App struct {
	cfg    *config.Config
	parser parser.Parser
	tts    tts.Service
}

// NewApp creates a new instance of the application
func NewApp(cfg *config.Config) *App {
	return &App{
		cfg:    cfg,
		parser: parser.NewParser(),
		tts:    tts.NewService(cfg),
	}
}

// Run starts the application
func (a *App) Run() error {
	// TODO: Implement TUI interface and main application logic
	return nil
}
