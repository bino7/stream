package stream

import (
	"context"
	"fmt"
	"github.com/bino7/kube/lib"
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
	Cancel()
	Close()
	Then(apply interface{}) Streams
	Catch(apply ErrorHandleFunc) Streams
	Pipe(...Streams) Streams
	Await() (interface{}, error)
}

type staticStreams struct {
	state       int
	ctx         context.Context
	cancelFunc  context.CancelFunc
	then        []interface{}
	catch       []ErrorHandleFunc
	downStreams []Stream
	result      interface{}
	err         error
	mutex       *sync.Mutex
}

type streams struct {
	*staticStreams
	Stream
	name        string
	state       int
	ctx         context.Context
	cancelFunc  context.CancelFunc
	then        []interface{}
	catch       []ErrorHandleFunc
	downStreams map[string]Streams
	result      interface{}
	err         error
	mutex       *sync.Mutex
	wg          sync.WaitGroup
}

func Once(name string, ctx context.Context, handles ...interface{}) Streams {
	return newStaticStreams(name, ctx, handles...)
}

func newStaticStreams(name string, ctx context.Context, handles ...interface{}) Streams {
	ctx, cancelFunc := context.WithCancel(ctx)
	if handles == nil {
		handles = make([]interface{}, 0)
	}
	var s = &streams{
		Stream:      nil,
		name:        name,
		ctx:         ctx,
		cancelFunc:  cancelFunc,
		then:        handles,
		catch:       make([]ErrorHandleFunc, 0),
		downStreams: make(map[string]Streams),
		result:      nil,
		err:         nil,
		mutex:       &sync.Mutex{},
	}
	return s
}

func With(name string, ctx context.Context, buffSize int, handles ...interface{}) Streams {
	loop := func(s *streams) {
		for {
			select {
			case <-s.ctx.Done():
				s.Cancel()
				break
			case v := <-s.Stream:
				s.Resolve(v)
			}
		}
	}
	return newStreams(name, ctx, buffSize, loop, handles...)
}

func newStreams(name string, ctx context.Context, n int, loop func(*streams), handles ...interface{}) Streams {
	ctx, cancelFunc := context.WithCancel(ctx)
	if handles == nil {
		handles = make([]interface{}, 0)
	}
	var s = &streams{
		Stream:      New(n),
		name:        name,
		ctx:         ctx,
		cancelFunc:  cancelFunc,
		then:        handles,
		catch:       make([]ErrorHandleFunc, 0),
		downStreams: make(map[string]Streams),
		result:      nil,
		err:         nil,
		mutex:       &sync.Mutex{},
	}
	go loop(s)
	return s
}

func (s *streams) Name() string {
	return s.name
}
func (s *streams) Input() Stream {
	return s.Stream
}
func (s *streams) Cancel() {
	s.cancelFunc()
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
func (s *streams) resolveResult(result interface{}, err error) bool {
	s.result = result
	s.err = err
	if err != nil {
		s.reject(err)
		return false
	}

	if s.result == nil {
		return false
	}
	switch s.result.(type) {
	case *promise.Promise:
		p := s.result.(*promise.Promise)
		res, err := p.Await()
		if err != nil {
			s.reject(err)
			return false
		}
		s.result = res
	case error:
		err := s.result.(error)
		s.reject(err)
		return false
	}
	return true
}
func (s *streams) resolve(resolution interface{}) {
	if !s.resolveResult(resolution, nil) {
		return
	}
	for _, apply := range s.then {
		switch apply.(type) {
		case *Handler:
			fn := apply.(*Handler)
			var v interface{}
			if fn.Before != nil {
				v = fn.Before
			} else {
				v = s.ctx.Value(NodeFuncKey(BeforeHandlePrefix, fn))
			}
			if v != nil {
				before := v.(HandleFunc)
				if !s.resolveResult(before(s.result)) {
					break
				}
			}
			r, err := fn.Eval(s.result)
			if !s.resolveResult(r, err) {
				break
			}

			if fn.After != nil {
				v = fn.After
			} else {
				v = s.ctx.Value(NodeFuncKey(AfterHandlePrefix, fn))
			}
			if v != nil {
				after := v.(HandleFunc)
				if !s.resolveResult(after(s.result)) {
					break
				}
			}

			if fn.Pipes != nil {
				for _, p := range fn.Pipes {
					p <- s.result
				}
			}
		case SliceHandler:
			fn := apply.(SliceHandler)
			var v interface{}
			if fn.Before != nil {
				v = fn.Before
			} else {
				v = s.ctx.Value(NodeFuncKey(BeforeHandlePrefix, fn))
			}
			if v != nil {
				before := v.(HandleFunc)
				if !s.resolveResult(before(s.result)) {
					break
				}
			}
			if !s.resolveResult(fn.Eval(s.result)) {
				break
			}

			if fn.After != nil {
				v = fn.After
			} else {
				v = s.ctx.Value(NodeFuncKey(AfterHandlePrefix, fn))
			}
			if v != nil {
				after := v.(HandleFunc)
				if !s.resolveResult(after(s.result)) {
					break
				}
			}

			if fn.Pipes != nil {
				for _, p := range fn.Pipes {
					p <- s.result
				}
			}
		case HandleFunc:
		case func(interface{}) (interface{}, error):
			fn := apply.(func(interface{}) (interface{}, error))
			if v := s.ctx.Value(NodeFuncKey(BeforeHandlePrefix, fn)); v != nil {
				before := v.(func(interface{}) (interface{}, error))
				if !s.resolveResult(before(s.result)) {
					break
				}
			}
			if !s.resolveResult(fn(s.result)) {
				break
			}
			if v := s.ctx.Value(NodeFuncKey(AfterHandlePrefix, fn)); v != nil {
				after := v.(func(interface{}) (interface{}, error))
				if !s.resolveResult(after(s.result)) {
					break
				}
			}
		}

	}
}

func NodeFuncKey(prefix string, fn interface{}) string {
	switch fn.(type) {
	case *Handler:
		return Streams2NodeFuncKey(prefix, fn.(*Handler))
	case HandleFunc:
		return HandleFuncKey(prefix, fn.(HandleFunc))
	default:
		return ""
	}
}

func Streams2NodeFuncKey(prefix string, fn *Handler) string {
	name := fn.Name
	if name == "" {
		return HandleFuncKey(prefix, fn.Apply)
	}
	return fmt.Sprintf("%s-%s", prefix, fn.Name)
}
func HandleFuncKey(prefix string, fn HandleFunc) string {
	pkg, name, _ := lib.FuncName(fn)
	return fmt.Sprintf("%s-%s-%s", prefix, pkg, name)
}

func (s *streams) reject(err error) {
	if err != nil {
		s.err = err
	}
	for _, fn := range s.catch {
		err := fn(s.err)
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
	s.Stream.Close()
	s.Cancel()
}
func (s *streams) Then(apply interface{}) Streams {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	switch apply.(type) {
	case *Handler:
		s.then = append(s.then, apply)
	case HandleFunc:
		s.then = append(s.then, apply)
	case func(interface{}) (interface{}, error):
		s.then = append(s.then, apply)
	default:
		return s
	}
	return s
}
func (s *streams) Catch(apply ErrorHandleFunc) Streams {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.catch = append(s.catch, apply)
	return s
}
func (s *streams) Pipe(streams ...Streams) Streams {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, strm := range streams {
		s.downStreams[strm.Name()] = strm
	}
	return s
}

func (s *streams) Await() (interface{}, error) {
	s.wg.Wait()
	return s.result, s.err
}
