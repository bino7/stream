package stream

import (
	"container/ring"
	"sync"
)

type Available interface {
	Available() bool
}
type Source interface {
	SourceAvailable() bool
	Source() <-chan interface{}
}

type Sink interface {
	SinkAvailable() bool
	Input() chan<- interface{}
	Accept(interface{}) bool
}

type Dispatcher struct {
	sync.Mutex
	Engine
	source Source
	sinks  *ring.Ring
}

/*func NewDispatcher(ctx context.Context, source Source) *Dispatcher {
	var mu sync.Mutex
	engine := NewEngine(ctx, FromOutbound(source.Source()), nil)
	sinks := ring.New(0)
	dispatcher := &Dispatcher{mu, engine, source, sinks}
	engine.Create().Do(func(v interface{}) interface{} {
		dispatcher.Lock()
		defer dispatcher.Unlock()
		for r := sinks; r != sinks; {
			sink := r.Value.(Sink)
			if !sink.SinkAvailable() {
				r = r.Next()
				dispatcher.UnRegister(sink)
			} else if sink.Accept(v) {
				sink.Input() <- v
				r = r.Next()
			}
		}
		return nil
	})
	return dispatcher
}*/

func (d *Dispatcher) Register(sink Sink) {
	d.Lock()
	defer d.Unlock()
	r := ring.New(1)
	r.Value = sink
	d.sinks.Link(r)
}
func (d *Dispatcher) UnRegister(sink Sink) {
	d.Lock()
	defer d.Unlock()
	sinks := d.sinks
	n := 0
	remove := func(v interface{}) {
		s := v.(Sink)
		if s == sink {
			sinks.Unlink(n)
		}
		n++
	}
	sinks.Do(remove)
}
