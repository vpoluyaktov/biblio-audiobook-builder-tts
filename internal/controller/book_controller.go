package controller

import (
	"fmt"
	"strings"
	"github.com/vpoluyaktov/abb_tts/internal/dto"
	"github.com/vpoluyaktov/abb_tts/internal/mq"
	"github.com/vpoluyaktov/abb_tts/internal/monitoring"
	parser "github.com/vpoluyaktov/abb_tts/internal/parser"
)

type BookController struct {
	mq *mq.Dispatcher
	metrics *monitoring.Metrics
}

func NewBookController(dispatcher *mq.Dispatcher) *BookController {
	c := &BookController{
		mq: dispatcher,
		metrics: monitoring.NewMetrics("book_controller"),
	}
	dispatcher.RegisterHandler(mq.BookController, c.dispatchMessage)
	return c
}

func (c *BookController) dispatchMessage(m *mq.Message) {
	c.metrics.IncrementCounter("messages_received")
	
	switch dto := m.Dto.(type) {
	case *dto.ParseBookCommand:
		go c.parseBook(dto, m.From)
	default:
		m.UnsupportedTypeError(mq.BookController)
		c.metrics.IncrementCounter("unsupported_messages")
	}
}

func (c *BookController) parseBook(cmd *dto.ParseBookCommand, replyTo string) {
	c.metrics.IncrementCounter("parse_book_requests")
	
	var book *parser.Book
	var err error
	ext := strings.ToLower(cmd.FilePath)
	if strings.HasSuffix(ext, ".epub") {
		book, err = parser.NewEpubParser().ParseEpubFile(cmd.FilePath)
		if err == nil {
			c.metrics.IncrementCounter("epub_parsed_success")
		}
	} else if strings.HasSuffix(ext, ".fb2") {
		book, err = parser.NewFB2Parser().ParseFB2File(cmd.FilePath)
		if err == nil {
			c.metrics.IncrementCounter("fb2_parsed_success")
		}
	} else {
		err = fmt.Errorf("Unsupported file format: %s", cmd.FilePath)
	}
	
	result := &dto.BookParsedResult{Book: book}
	if err != nil {
		result.Error = err.Error()
		c.metrics.IncrementCounter("parse_errors")
	}
	
	c.mq.SendMessage(mq.BookController, replyTo, result, mq.PriorityNormal)
}

// checkMQ implements the controller interface
func (c *BookController) checkMQ() {
	// This is a no-op as we use the dispatcher's RegisterHandler mechanism
	// instead of polling for messages
}
