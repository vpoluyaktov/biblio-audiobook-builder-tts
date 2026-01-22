package mq

import (
	"time"

	"biblio-audiobook-builder-tts/internal/dto"
)

const (
	PullFrequency = 100 * time.Millisecond
)

type MessagePriority int

type Message struct {
	From       string
	To         string
	Dto        dto.Dto
	Priority   MessagePriority
	CreatedAt  time.Time
	ExpiresAt  time.Time
	MessageID  string
	RetryCount int
	MaxRetries int
	Metadata   map[string]any
}

type CallBackFunc func(*Message)

func (m *Message) UnsupportedTypeError(recipient string) {
	m.Dto = &dto.ConversionError{
		Chapter: "",
		Error:   "Unsupported message type for " + recipient,
	}
}
