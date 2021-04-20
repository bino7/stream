package stream

import (
	"context"
	"github.com/bino7/promise"
	"sync"
)

const (
	OnErrorHandle      = "OnErrorHandle"
	BeforeHandlePrefix = "Before"
	AfterHandlePrefix  = "After"
)

type Streams interface {
	Name() string
	Input() Stream
	Resolve(resolution interface{})
	Close()
	Then(apply interface{}) Streams
	Catch(apply ErrorHandleFunc) Streams
	Await() (interface{}, error)
	Len() int
	Done() <-chan struct{}
}

type streams struct {
	Stream
	name       string
	state      int
	ctx        context.Context
	cancelFunc context.CancelFunc
	then       []*handler
	catch      []ErrorHandleFunc
	result     interface{}
	err        error
	mutex      *sync.Mutex
	wg         sync.WaitGroup
	done       chan struct{}
}

type OnceStreams struct {
	*streams
}

func Once(name string, ctx context.Context, steps ...interface{}) (*OnceStreams, error) {
	s, err := newStreams(name, ctx, -1, steps...)
	if err != nil {
		return nil, err
	}
	return &OnceStreams{s}, nil
}

func NewStreams(name string, ctx context.Context, buffSize int, steps ...interface{}) (Streams, error) {
	s, err := newStreams(name, ctx, buffSize, steps...)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				s.Close()
				break
			case v := <-s.Stream:
				s.Resolve(v)
			}
		}
	}()
	return s, nil
}

func newStreams(name string, ctx context.Context, buffSize int, steps ...interface{}) (*streams, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	handlers, err := toHandlers(steps...)

	if err != nil {
		return nil, err
	}

	var input Stream
	if buffSize >= 0 {
		input = New(buffSize)
	}

	var s *streams

	s = &streams{
		Stream:     input,
		name:       name,
		ctx:        ctx,
		cancelFunc: cancelFunc,
		then:       handlers,
		catch:      make([]ErrorHandleFunc, 0),
		result:     nil,
		err:        nil,
		mutex:      &sync.Mutex{},
		done:       make(chan struct{}),
	}

	return s, nil
}

func (s *streams) Len() int {
	return len(s.then)
}

func (s *streams) Name() string {
	return s.name
}

func (s *streams) Input() Stream {
	return s.Stream
}

func (s *streams) Resolve(resolution interface{}) {
	if resolution == nil {
		return
	}
	s.mutex.Lock()
	s.wg = sync.WaitGroup{}
	s.wg.Add(1)
	s.resolve(resolution)
	s.wg.Done()
	s.mutex.Unlock()
}

func (s *streams) resolved(result interface{}, err error) bool {
	s.result = result
	s.err = err
	if err != nil {
		s.reject(err)
		return true
	}

	if s.result == nil {
		return true
	}

	switch s.result.(type) {
	case *promise.Promise:
		p := s.result.(*promise.Promise)
		res, err := p.Await()
		return s.resolved(res, err)
	case error:
		err := s.result.(error)
		s.err = err
		s.reject(err)
		return true
	}
	return false
}

func (s *streams) resolve(resolution interface{}) {
	if s.resolved(resolution, nil) {
		return
	}
	startStep := 0
loop:
	for i, fn := range s.then {
		if i < startStep {
			continue
		}
		result, err := fn.Apply(s.result)
		switch result.(type) {
		case *_goto:
			startStep = result.(*_goto).step
			goto loop
		}
		if s.resolved(result, err) {
			return
		}
	}
}

func (s *streams) reject(err error) {
	if err != nil {
		s.err = err
	}
	for _, fn := range s.catch {
		err = fn(s.err)
		if err != nil {
			s.err = err
		}
	}
	if v := s.ctx.Value(OnErrorHandle); v != nil {
		if onErrorHandle, ok := v.(ErrorHandleFunc); ok {
			_ = onErrorHandle(s.err)
		}
	}
}

func (s *streams) Close() {
	s.cancelFunc()
	s.done <- struct{}{}
	s.Stream.Close()
}

func (s *streams) Then(step interface{}) Streams {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	handlers, _ := toHandlers(step)
	if handlers != nil {
		for _, h := range handlers {
			h.step = s.Len()
			h.prefixFunc = func() string {
				return s.name
			}
			s.then = append(s.then, h)
		}
	}
	return s
}
func (s *streams) Catch(apply ErrorHandleFunc) Streams {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.catch = append(s.catch, apply)
	return s
}

func (s *streams) Await() (interface{}, error) {
	s.wg.Wait()
	return s.result, s.err
}

func (s *streams) Done() <-chan struct{} {
	return s.done
}
