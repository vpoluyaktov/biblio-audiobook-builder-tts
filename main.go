package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"biblio-audiobook-builder-tts/internal/config"
	"biblio-audiobook-builder-tts/internal/logger"
	"biblio-audiobook-builder-tts/internal/server"
	"biblio-audiobook-builder-tts/internal/storage"
	"biblio-audiobook-builder-tts/internal/tts"
	"biblio-audiobook-builder-tts/internal/tui"
	"biblio-audiobook-builder-tts/internal/utils"

	"golang.org/x/term"
)

func main() {
	// Customize usage
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of biblio-audiobook-builder-tts:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  --db string\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Path to SQLite database file (default \"biblio-audiobook-builder-tts.db\")\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  --port string\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Port to run the server on (overrides config)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  --host string\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Host to bind the server to (overrides config)\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  --no-browser\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Don't automatically open browser\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  --restart\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Kill any existing process on the port before starting\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  --log-level string\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Log level: DEBUG, INFO, WARN, ERROR (default \"INFO\")\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  --headless\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Run without TUI (headless mode)\n")
	}

	// Parse command line flags
	dbPath := flag.String("db", getEnvOrDefault("ABB_TTS_DATABASE_PATH", config.DefaultDBPath), "Path to SQLite database file")
	port := flag.String("port", "", "Port to run the server on (overrides config)")
	host := flag.String("host", "", "Host to bind the server to (overrides config)")
	noBrowser := flag.Bool("no-browser", false, "Don't automatically open browser")
	restart := flag.Bool("restart", false, "Kill any existing process on the port before starting")
	logLevel := flag.String("log-level", "INFO", "Log level: DEBUG, INFO, WARN, ERROR")
	headless := flag.Bool("headless", false, "Run without TUI (headless mode)")
	flag.Parse()

	// Set log level
	logger.SetLevelFromString(*logLevel)

	// Initialize database
	db, err := storage.NewDB(*dbPath)
	if err != nil {
		logger.Fatal("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize database with defaults if empty
	if err := db.InitializeDefaults(); err != nil {
		logger.Fatal("Failed to initialize database: %v", err)
	}

	// Load configuration from database
	dbConfig, err := db.GetAllConfig()
	if err != nil {
		logger.Fatal("Failed to load configuration from database: %v", err)
	}

	// Convert to app config
	appConfigMap := dbConfig.ToAppConfig()
	cfg := config.LoadFromDB(appConfigMap)
	config.SetInstance(cfg)

	// Override config with environment variables
	if envPort := os.Getenv("ABB_TTS_PORT"); envPort != "" {
		cfg.ServerPort = envPort
	}
	if envHost := os.Getenv("ABB_TTS_HOST"); envHost != "" {
		cfg.ServerHost = envHost
	}
	if envOutputDir := os.Getenv("ABB_TTS_OUTPUT_DIR"); envOutputDir != "" {
		cfg.OutputDir = envOutputDir
	}
	if envTempDir := os.Getenv("ABB_TTS_TEMP_DIR"); envTempDir != "" {
		cfg.TempDir = envTempDir
	}

	// Override config with command line flags (highest priority)
	if *port != "" {
		cfg.ServerPort = *port
	}
	if *host != "" {
		cfg.ServerHost = *host
	}
	if *noBrowser {
		cfg.OpenBrowser = false
	}

	// Handle restart flag - kill existing process on the port
	if *restart {
		err := utils.KillProcessOnPort(cfg.ServerPort)
		if err == nil {
			fmt.Printf("Killed existing process on port %s\n", cfg.ServerPort)
			// Give process time to clean up
			time.Sleep(500 * time.Millisecond)
		} else if !strings.Contains(err.Error(), "no process found") {
			// Real error (not just "no process found")
			fmt.Printf("Warning: Failed to kill process on port %s: %v\n", cfg.ServerPort, err)
		}
		// If no process found, silently continue
	}

	// Setup logging (truncate log file on each start)
	if cfg.LogFile != "" {
		logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			logger.Warn("Failed to open log file: %v", err)
		} else {
			logger.SetOutput(logFile)
			defer logFile.Close()
		}
	}

	// Create a context that will be canceled on SIGINT or SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Shutdown signal received")
		cancel()
	}()

	// Create TTS service with database support
	logger.Debug("Creating TTS service with database provider support")
	ttsService := tts.NewServiceWithDB(cfg, db)

	// Create and start server
	addr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
	srv := server.New(addr, cfg, ttsService)
	srv.SetDB(db) // Enable config persistence to database

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Give server time to start
	time.Sleep(500 * time.Millisecond)

	// Check if server failed to start
	select {
	case err := <-errChan:
		logger.Fatal("Server failed to start: %v", err)
	default:
		// Server started successfully
	}

	url := fmt.Sprintf("http://localhost:%s", cfg.ServerPort)

	// Open browser if enabled
	if cfg.OpenBrowser {
		go openBrowser(url)
	}

	// Determine if we should run TUI
	// Run TUI if: not headless, and running in a terminal
	runTUI := !*headless && term.IsTerminal(int(os.Stdin.Fd()))

	if runTUI {
		// Run TUI - it will handle shutdown
		go func() {
			if err := tui.RunTUI(url, srv.GetDB(), ttsService, srv.GetHub()); err != nil {
				logger.Error("TUI error: %v", err)
			}
			// TUI exited, trigger shutdown
			cancel()
		}()
	} else {
		// Headless mode
		fmt.Printf("🎧 Audiobook Builder TTS Server running at %s\n", url)
		fmt.Println("Press Ctrl+C to stop")
	}

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		logger.Info("Shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := srv.Stop(shutdownCtx); err != nil {
			logger.Error("Server shutdown error: %v", err)
		}
	case err := <-errChan:
		if err != nil {
			logger.Fatal("Server error: %v", err)
		}
	}

	logger.Info("Server stopped")
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// openBrowser opens the default browser to the given URL
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		logger.Warn("Cannot open browser on %s", runtime.GOOS)
		return
	}

	if err := cmd.Start(); err != nil {
		logger.Warn("Failed to open browser: %v", err)
	}
}
