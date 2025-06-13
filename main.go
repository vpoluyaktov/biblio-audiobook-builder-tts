package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vpoluyaktov/abb_tts/internal/config"
	"github.com/vpoluyaktov/abb_tts/internal/controller"
	"github.com/vpoluyaktov/abb_tts/internal/monitoring"
	"github.com/vpoluyaktov/abb_tts/internal/mq"
	"github.com/vpoluyaktov/abb_tts/internal/tts"
	"github.com/vpoluyaktov/abb_tts/internal/ui"
)

// Min screen size for comfortable layout 45x125 characters
func main() {
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

	// Parse command line flags
	configFile := flag.String("config", "abb_tts.config.yaml", "Path to configuration file")
	enableMetrics := flag.Bool("enable-metrics", false, "Enable metrics collection")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	// Set the global config instance
	config.SetInstance(cfg)

	// Initialize metrics if enabled
	if *enableMetrics {
		monitoring.EnableMetrics()
		
		// Update uptime metric periodically
		startTime := time.Now()
		monitoring.SetGauge("app_uptime_seconds", 0, monitoring.Labels{
			"version": cfg.GetVersion(),
		})
		
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					monitoring.SetGauge("app_uptime_seconds",
						time.Since(startTime).Seconds(),
						monitoring.Labels{"version": cfg.GetVersion()})
				}
			}
		}()
	}

	execute(ctx)
}

func execute(ctx context.Context) {
	log.Println("Application started")

	// Create message queue dispatcher
	dispatcher := mq.NewDispatcher()

	// Create TTS service
	service := tts.NewService(config.Instance())

	// Create and initialize controllers
	conductor := controller.NewConductor(dispatcher)

	// Add controllers to the conductor
	ttsController := controller.NewTTSController(dispatcher, service)
	bookController := controller.NewBookController(dispatcher)
	
	conductor.AddController(ttsController)
	conductor.AddController(bookController)
	
	// Start the conductor
	conductor.Run()

	// Create and initialize UI
	app := ui.NewTUI()
	app.SetDispatcher(dispatcher)
	app.SetService(service)

	// Run the application in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- app.Run()
	}()

	// Wait for context cancellation or UI error
	select {
	case <-ctx.Done():
		log.Println("Shutting down application")
		app.Stop()
	case err := <-errChan:
		if err != nil {
			log.Fatalf("Application error: %v", err)
		}
	}

	log.Println("Application finished")
}
