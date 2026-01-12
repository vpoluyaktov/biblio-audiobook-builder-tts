package controller

import (
	"time"

	"abb_tts/internal/mq"
)

type controller interface {
	checkMQ()
}

type Conductor struct {
	dispatcher  *mq.Dispatcher
	controllers []controller
}

func NewConductor(dispatcher *mq.Dispatcher) *Conductor {
	c := &Conductor{
		dispatcher:  dispatcher,
		controllers: make([]controller, 0),
	}
	return c
}

func (c *Conductor) AddController(ctrl controller) {
	c.controllers = append(c.controllers, ctrl)
}

func (c *Conductor) startEventListener() {
	for {
		for _, p := range c.controllers {
			p.checkMQ()
		}
		time.Sleep(mq.PullFrequency)
	}
}

func (c *Conductor) Run() {
	go c.startEventListener()
}
