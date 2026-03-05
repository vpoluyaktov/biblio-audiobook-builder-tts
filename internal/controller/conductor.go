package controller

import (
	"biblio-audiobook-builder-tts/internal/mq"
)

// Conductor manages controller lifecycle
// Controllers register themselves with the dispatcher using RegisterHandler
// so no polling is needed - the conductor just keeps track of controllers
type Conductor struct {
	dispatcher  *mq.Dispatcher
	controllers []interface{}
}

func NewConductor(dispatcher *mq.Dispatcher) *Conductor {
	c := &Conductor{
		dispatcher:  dispatcher,
		controllers: make([]interface{}, 0),
	}
	return c
}

// AddController registers a controller with the conductor
// The controller should have already registered its message handlers with the dispatcher
func (c *Conductor) AddController(ctrl interface{}) {
	c.controllers = append(c.controllers, ctrl)
}

// Run starts the conductor (currently a no-op since controllers use event-driven handlers)
// Kept for API compatibility and future extensions
func (c *Conductor) Run() {
	// Controllers use RegisterHandler mechanism, so no polling loop needed
	// This method is kept for API compatibility
}
