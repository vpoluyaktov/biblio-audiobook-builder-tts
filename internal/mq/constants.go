package mq

// Message queue priorities
const (
	PriorityLow    = 0
	PriorityNormal = 1
	PriorityHigh   = 2
)

// Message queue destinations
const (
	BookPage      = "book_page"
	BookController = "book_controller"
	ConfigPage    = "config_page"
	Frame         = "frame"
	TTSController = "tts_controller"
	Footer        = "footer"
)
