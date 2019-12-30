package stream

import (
	"context"
	"time"
)

type Engine interface {
	Start()
	Cancel()
	Create() *gearCreator
	At(int) Gear
	Running() bool
	Every(duration time.Duration, do func() bool) Engine
	Input() Stream
}

type engine struct {
	Stream
	context.Context
	cancelFunc context.CancelFunc
	gear       Gear
	isRunning  bool
}

func NewEngine(parent context.Context, stream Stream, gear Gear) Engine {
	if stream == nil {
		stream = make(chan interface{}, 1000)
	}
	ctx, cancelFunc := context.WithCancel(parent)
	return &engine{stream, ctx, cancelFunc, gear, false}
}

func (e *engine) Start() {
	go func() {
		e.isRunning = true
		for {
			select {
			case <-e.Context.Done():
				break
			case v := <-e.Stream:
				e.gear.Do(v)
			}
		}
	}()
}

func (e *engine) Cancel() {
	e.cancelFunc()
}

func (e *engine) Create() *gearCreator {
	return newGearCreator(e.Context, e.gear)
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

func (e *engine) Input() Stream {
	return e.Stream
}
