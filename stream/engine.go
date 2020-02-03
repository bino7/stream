package stream

import (
	"context"
	"time"
)

type Engine interface {
	Sink
	Start()
	Cancel()
	Create() *gearCreator
	At(int) Gear
	Running() bool
	Every(duration time.Duration, do func() bool) Engine
	Do(do func() bool) Engine
}

type engine struct {
	Stream
	context.Context
	cancelFunc context.CancelFunc
	gear       Gear
	isRunning  bool
	available  bool
}

func NewEngine(parent context.Context, stream Stream, gear Gear) Engine {
	if stream == nil {
		stream = make(chan interface{}, 1000)
	}
	ctx, cancelFunc := context.WithCancel(parent)
	return &engine{stream, ctx, cancelFunc, gear, false, true}
}

func (e *engine) Start() {
	go func() {
		e.isRunning = true
		if e.gear == nil {
			e.Cancel()
		} else {
			for {
				select {
				case <-e.Context.Done():
					break
				case v := <-e.Stream:
					e.gear.Do(v)
				}
			}
		}

	}()
}

func (e *engine) Cancel() {
	e.cancelFunc()
}

func (e *engine) Create() *gearCreator {
	return newGearCreator(e.Context, e.gear, e.setFirstGear)
}

func (e *engine) setFirstGear(g Gear) {
	e.gear = g
}

func (e *engine) At(n int) Gear {
	return nextNthGear(e.gear, n)
}
func (e *engine) Running() bool {
	return e.isRunning
}

func (e *engine) Every(duration time.Duration, do func() bool) Engine {
	go func() {
		for {
			select {
			case <-e.Done():
				return
			case <-time.Tick(duration):
				if !do() {
					return
				}
			}
		}
	}()
	return e
}

func (e *engine) Do(do func() bool) Engine {
	go func() {
		for {
			select {
			case <-e.Done():
				return
			default:
				if !do() {
					return
				}
			}
		}
	}()
	return e
}

func (e *engine) Input() chan<- interface{} {
	return e.Stream
}

func (e *engine) Accept(interface{}) bool {
	return true
}

func (e *engine) Available() bool {
	return e.available
}
