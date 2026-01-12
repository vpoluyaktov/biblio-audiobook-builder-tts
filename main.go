package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"abb_tts/internal/config"
	"abb_tts/internal/server"
	"abb_tts/internal/tts"
	"abb_tts/internal/tui"
	"abb_tts/internal/utils"

	"golang.org/x/term"
)

// LogLevel represents logging verbosity
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

var currentLogLevel = LogLevelInfo

// parseLogLevel converts a string to LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN", "WARNING":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	default:
		return LogLevelInfo
	}
}

func main() {
	// Customize usage
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of abb_tts:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  --config string\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Path to configuration file (default \"abb_tts.config.yaml\")\n")
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
	configFile := flag.String("config", "abb_tts.config.yaml", "Path to configuration file")
	port := flag.String("port", "", "Port to run the server on (overrides config)")
	host := flag.String("host", "", "Host to bind the server to (overrides config)")
	noBrowser := flag.Bool("no-browser", false, "Don't automatically open browser")
	restart := flag.Bool("restart", false, "Kill any existing process on the port before starting")
	logLevel := flag.String("log-level", "INFO", "Log level: DEBUG, INFO, WARN, ERROR")
	headless := flag.Bool("headless", false, "Run without TUI (headless mode)")
	flag.Parse()

	// Set log level
	currentLogLevel = parseLogLevel(*logLevel)

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	config.SetInstance(cfg)

	// Override config with command line flags
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

	// Setup logging
	if cfg.LogFile != "" {
		logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Warning: Failed to open log file: %v", err)
		} else {
			log.SetOutput(logFile)
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
		log.Println("Shutdown signal received")
		cancel()
	}()

	// Create TTS service
	ttsService := tts.NewService(cfg)

	// Create and start server
	addr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
	srv := server.New(addr, cfg, ttsService)

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
		log.Fatalf("Server failed to start: %v", err)
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
			if err := tui.RunTUI(url, srv.GetStore(), ttsService, srv.GetHub()); err != nil {
				log.Printf("TUI error: %v", err)
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
		log.Println("Shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := srv.Stop(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	case err := <-errChan:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}

	log.Println("Server stopped")
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
		log.Printf("Cannot open browser on %s", runtime.GOOS)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}
