package stream

import (
	"context"
)

type Engine interface {
	Start()
	Cancel()
	Create() *gearCreator
	At(int) Gear
}

type engine struct {
	Stream
	context.Context
	cancelFunc context.CancelFunc
	gear       Gear
}

func NewEngine(parent context.Context, stream Stream, gear Gear) Engine {
	if stream == nil {
		stream = make(chan interface{}, 1000)
	}
	ctx, cancelFunc := context.WithCancel(parent)
	return &engine{stream, ctx, cancelFunc, gear}
}

func (e *engine) Start() {
	go func() {
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
