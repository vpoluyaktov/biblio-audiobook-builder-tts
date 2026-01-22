package dto

import (
	"biblio-audiobook-builder-tts/internal/parser"
)

// Base interface for all DTOs
type Dto interface{}

// Command DTOs
type ConvertCommand struct {
	Book     *parser.Book
	Voice    string
	Speed    float64
	Pitch    float64
	Provider string
}

// Status DTOs
type UpdateStatus struct {
	Message string
}

type SetBusyIndicator struct {
	Busy bool
}

type ConversionProgress struct {
	ChapterTitle string
	Progress     float64
}

// Result DTOs
type ConversionComplete struct {
	OutputPath string
}

type ConversionError struct {
	Chapter string
	Error   string
}
