package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vpoluyaktov/abb_tts/internal/dto"
	"github.com/vpoluyaktov/abb_tts/internal/monitoring"
)

type Dispatcher struct {
	mu              sync.RWMutex
	messageChannels map[string]chan *Message
	listeners       map[string]CallBackFunc
	ctx            context.Context
	cancel         context.CancelFunc
	metrics        *monitoring.Metrics
}

func NewDispatcher() *Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	d := &Dispatcher{
		messageChannels: make(map[string]chan *Message),
		listeners:       make(map[string]CallBackFunc),
		ctx:            ctx,
		cancel:         cancel,
		metrics:        monitoring.NewMetrics("dispatcher"),
	}
	return d
}

func (d *Dispatcher) RegisterHandler(recipient string, handler CallBackFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.listeners[recipient] = handler

	if _, exists := d.messageChannels[recipient]; !exists {
		ch := make(chan *Message, 100)
		d.messageChannels[recipient] = ch
		go d.processMessages(recipient, ch)
	}
}

func (d *Dispatcher) SendMessage(from string, to string, dto dto.Dto, priority MessagePriority) error {
	msg := &Message{
		From:       from,
		To:         to,
		Dto:        dto,
		Priority:   priority,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(5 * time.Minute),
		MessageID:  generateMessageID(),
		MaxRetries: 3,
		Metadata:   make(map[string]any),
	}

	d.mu.Lock()
	ch, exists := d.messageChannels[to]
	if !exists {
		ch = make(chan *Message, 100)
		d.messageChannels[to] = ch
		go d.processMessages(to, ch)
	}
	d.mu.Unlock()

	select {
	case ch <- msg:
		d.metrics.IncrementCounter("messages_sent")
		return nil
	default:
		d.metrics.IncrementCounter("messages_dropped")
		return fmt.Errorf("failed to send message to %s: channel full", to)
	}
}

func (d *Dispatcher) GetMessage(recipient string) (*Message, error) {
	d.mu.RLock()
	ch, exists := d.messageChannels[recipient]
	d.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no channel for recipient: %s", recipient)
	}

	select {
	case msg := <-ch:
		if msg != nil {
			d.metrics.IncrementCounter("messages_received")
		}
		return msg, nil
	default:
		return nil, nil
	}
}

func (d *Dispatcher) processMessages(recipient string, ch chan *Message) {
	for {
		select {
		case <-d.ctx.Done():
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}

			if time.Now().After(msg.ExpiresAt) {
				d.metrics.IncrementCounter("messages_expired")
				continue
			}

			if err := d.handleMessage(msg); err != nil {
				d.metrics.IncrementCounter("messages_failed")
				if msg.RetryCount < msg.MaxRetries {
					msg.RetryCount++
					time.Sleep(time.Duration(msg.RetryCount*msg.RetryCount) * time.Second)
					ch <- msg
				}
			} else {
				d.metrics.IncrementCounter("messages_processed")
			}
		}
	}
}

func (d *Dispatcher) handleMessage(msg *Message) error {
	d.mu.RLock()
	listener, exists := d.listeners[msg.To]
	d.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no listener registered for recipient: %s", msg.To)
	}

	defer func() {
		if r := recover(); r != nil {
			d.metrics.IncrementCounter("message_panics")
		}
	}()

	listener(msg)
	return nil
}

func (d *Dispatcher) Shutdown(timeout time.Duration) error {
	d.cancel()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan bool)
	go func() {
		d.mu.Lock()
		defer d.mu.Unlock()

		for _, ch := range d.messageChannels {
			close(ch)
		}
		done <- true
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("shutdown timed out")
	case <-done:
		return nil
	}
}

func generateMessageID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}
