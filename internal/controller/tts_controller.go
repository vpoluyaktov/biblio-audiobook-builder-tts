package controller

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vpoluyaktov/abb_tts/internal/dto"
	"github.com/vpoluyaktov/abb_tts/internal/monitoring"
	"github.com/vpoluyaktov/abb_tts/internal/mq"
	"github.com/vpoluyaktov/abb_tts/internal/tts"
)

type TTSController struct {
	mq      *mq.Dispatcher
	service tts.Service
	metrics *monitoring.Metrics
}

func NewTTSController(dispatcher *mq.Dispatcher, service tts.Service) *TTSController {
	c := &TTSController{
		mq:      dispatcher,
		service: service,
		metrics: monitoring.NewMetrics("tts_controller"),
	}
	c.mq.RegisterHandler(mq.TTSController, c.dispatchMessage)
	return c
}

func (c *TTSController) checkMQ() {
	m, err := c.mq.GetMessage(mq.TTSController)
	if err != nil {
		return
	}
	if m != nil {
		c.dispatchMessage(m)
	}
}

func (c *TTSController) dispatchMessage(m *mq.Message) {
	c.metrics.IncrementCounter("messages_received")
	
	switch cmd := m.Dto.(type) {
	case *dto.ConvertCommand:
		go c.convert(cmd)
		c.metrics.IncrementCounter("convert_commands")
	default:
		m.UnsupportedTypeError(mq.TTSController)
		c.metrics.IncrementCounter("unsupported_messages")
	}
}

func (c *TTSController) convert(cmd *dto.ConvertCommand) {
	c.metrics.IncrementCounter("conversion_started")
	
	c.mq.SendMessage(mq.TTSController, mq.Footer, &dto.UpdateStatus{Message: "Converting text to speech..."}, mq.PriorityNormal)
	c.mq.SendMessage(mq.TTSController, mq.Footer, &dto.SetBusyIndicator{Busy: true}, mq.PriorityNormal)

	// Create output directory
	outputDir := filepath.Join("output", cmd.Book.Title)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		c.metrics.IncrementCounter("directory_creation_errors")
		c.mq.SendMessage(mq.TTSController, mq.BookPage, &dto.ConversionError{
			Chapter: "",
			Error:   fmt.Sprintf("Failed to create output directory: %v", err),
		}, mq.PriorityHigh)
		return
	}

	// Convert each chapter
	for i, chapter := range cmd.Book.Chapters {
		// Update progress
		c.mq.SendMessage(mq.TTSController, mq.Footer, &dto.UpdateStatus{
			Message: fmt.Sprintf("Converting chapter %d/%d: %s", i+1, len(cmd.Book.Chapters), chapter.Title),
		}, mq.PriorityNormal)

		c.mq.SendMessage(mq.TTSController, mq.BookPage, &dto.ConversionProgress{
			ChapterTitle: chapter.Title,
			Progress:     float64(i) / float64(len(cmd.Book.Chapters)),
		}, mq.PriorityNormal)

		// Convert chapter text to speech
		c.metrics.IncrementCounter("chapter_conversion_attempts")
		reader, err := c.service.ConvertToSpeech(chapter.Content, &tts.ConversionOptions{
			Voice:    cmd.Voice,
			Speed:    cmd.Speed,
			Pitch:    cmd.Pitch,
			Provider: cmd.Provider,
		})

		if err != nil {
			c.metrics.IncrementCounter("speech_conversion_errors")
			c.mq.SendMessage(mq.TTSController, mq.BookPage, &dto.ConversionError{
				Chapter: chapter.Title,
				Error:   err.Error(),
			}, mq.PriorityHigh)
			continue
		}

		// Save audio to file
		outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.mp3", chapter.Title))
		outputFile, err := os.Create(outputPath)
		if err != nil {
			c.metrics.IncrementCounter("file_creation_errors")
			c.mq.SendMessage(mq.TTSController, mq.BookPage, &dto.ConversionError{
				Chapter: chapter.Title,
				Error:   fmt.Sprintf("Failed to create output file: %v", err),
			}, mq.PriorityHigh)
			continue
		}
		defer outputFile.Close()

		if _, err := outputFile.ReadFrom(reader); err != nil {
			c.metrics.IncrementCounter("file_write_errors")
			c.mq.SendMessage(mq.TTSController, mq.BookPage, &dto.ConversionError{
				Chapter: chapter.Title,
				Error:   fmt.Sprintf("Failed to save audio file: %v", err),
			}, mq.PriorityHigh)
			continue
		}
		
		// Track successful chapter conversion
		c.metrics.IncrementCounter("chapter_conversion_success")
		// File is closed by defer
	}

	c.mq.SendMessage(mq.TTSController, mq.Footer, &dto.SetBusyIndicator{Busy: false}, mq.PriorityNormal)
	c.mq.SendMessage(mq.TTSController, mq.Footer, &dto.UpdateStatus{Message: ""}, mq.PriorityNormal)
	c.mq.SendMessage(mq.TTSController, mq.BookPage, &dto.ConversionComplete{
		OutputPath: outputDir,
	}, mq.PriorityNormal)
	
	c.metrics.IncrementCounter("conversion_completed")
}
