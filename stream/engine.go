package stream

import (
	"container/list"
	"context"
	"github.com/bino7/promise"
	"sync"
	"time"
)

type Engine interface {
	Sink
	Start()
	Cancel()
	Create() *GearCreator
	At(int) Gear
	Running() bool
	Every(duration time.Duration, do func() bool) Engine
	Do(do func() bool) Engine
	Put(interface{}) *promise.Promise
}

type promiseWithValue struct {
	promise *promise.Promise
	value   interface{}
}

type engine struct {
	Stream
	mu     sync.Mutex
	source Stream
	context.Context
	cancelFunc context.CancelFunc
	gear       Gear
	isRunning  bool
	available  bool
	curPromise *promiseWithValue
	promises   *list.List
}

func NewEngine(parent context.Context, stream Stream, gear Gear) Engine {
	if stream == nil {
		stream = make(chan interface{}, 1000)
	}
	var mu sync.Mutex
	ctx, cancelFunc := context.WithCancel(parent)
	return &engine{stream, mu, New(1000), ctx, cancelFunc, gear, false,
		true, nil, list.New()}
}

func (e *engine) Put(v interface{}) *promise.Promise {
	e.mu.Lock()
	defer e.mu.Unlock()

	p := promise.New(nil)
	e.promises.PushBack(&promiseWithValue{p, v})
	e.Stream <- v
	return p
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
					result := e.gear.Do(v)
					if e.curPromise == nil && e.promises.Len() > 0 {
						first := e.promises.Front()
						e.promises.Remove(first)
						e.curPromise = first.Value.(*promiseWithValue)
					}
					curP := e.curPromise
					if curP != nil && curP.value == v {
						err, ok := result.(error)
						if ok {
							curP.promise.Reject(err)
						} else {
							curP.promise.Resolve(result)
						}
					}
				}
			}
		}

	}()
}

func (e *engine) Cancel() {
	e.cancelFunc()
}

func (e *engine) Create() *GearCreator {
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

func (e *engine) SinkAvailable() bool {
	return e.available
}
