package stream

import (
	"container/ring"
	"context"
	"fmt"
	"sync"
)

type Streams struct {
	Stream
	state      int
	ctx        context.Context
	cancelFunc context.CancelFunc
	then       []func(data interface{}) interface{}
	catch      []func(err error) error
	outbounds  []Stream
	result     interface{}
	err        error
	mutex      *sync.Mutex
}

func NewStreams(ctx context.Context, buffSize int) *Streams {
	loop := func(s *Streams) {
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
	return newStreams(ctx, buffSize, loop)
}

func newStreams(ctx context.Context, n int, loop func(*Streams)) *Streams {
	ctx, cancelFunc := context.WithCancel(ctx)
	var s = &Streams{
		Stream:     New(n),
		ctx:        ctx,
		cancelFunc: cancelFunc,
		then:       make([]func(interface{}) interface{}, 0),
		catch:      make([]func(error) error, 0),
		outbounds:  make([]Stream, 0),
		result:     nil,
		err:        nil,
		mutex:      &sync.Mutex{},
	}
	go loop(s)
	return s
}

func (s *Streams) Put(v interface{}) {
	fmt.Println("put ", v, len(s.Stream))
	s.Stream <- v
	fmt.Println("putted ", v)
}

func (s *Streams) Cancel() {
	s.cancelFunc()
}

func (s *Streams) Then(apply HandleFunc) *Streams {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.then = append(s.then, apply)
	return s
}

func (s *Streams) Filter(apply FilterableFunc) *Streams {
	s.Then(func(v interface{}) interface{} {
		if apply(v) {
			return v
		}
		return nil
	})
	return s
}

func (s *Streams) When(test FilterableFunc, apply HandleFunc) *Streams {
	s.Then(func(v interface{}) interface{} {
		if test(v) {
			return apply(v)
		}
		return v
	})
	return s
}

func (s *Streams) Max(receiver interface{}, apply CompareFunc) *Streams {
	s.Then(func(v interface{}) interface{} {
		if receiver == nil || apply(receiver, v) < 0 {
			receiver = &v
		}
		return v
	})
	return s
}

func (s *Streams) Min(receiver interface{}, apply CompareFunc) *Streams {
	s.Then(func(v interface{}) interface{} {
		if receiver == nil || apply(receiver, v) > 0 {
			receiver = &v
		}
		return v
	})
	return s
}

func (s *Streams) Resolve(resolution interface{}) {
	s.mutex.Lock()
	s.result = resolution
	for _, fn := range s.then {
		result := fn(s.result)
		if result != nil {
			switch result := s.result.(type) {
			case error:
				err := result.(error)
				s.mutex.Unlock()
				s.Reject(err)
				return
			default:
				s.result = result
			}
		} else {
			s.result = nil
		}
	}
	if s.result != nil {
		if s.broadcast(0) == false {
			s.unHandle(s.result)
		}
	}
	s.mutex.Unlock()
}

func (s *Streams) broadcast(i int) bool {
	handled := false
	for j := i; j < len(s.outbounds); j++ {
		b := s.outbounds[j]
		if b.IsClosed() {
			lastOne := i == len(s.outbounds)-1
			s.outbounds = append(s.outbounds[0:i], s.outbounds[i+1:len(s.outbounds)]...)
			if !lastOne {
				return handled || s.broadcast(i)
			}
			break
		}
		if b.Accept(s.result) {
			b <- s.result
			handled = true
		}
	}
	return handled
}

func (s *Streams) Reject(err error) {
	s.mutex.Lock()
	if err != nil {
		s.err = err
	}
	for _, fn := range s.catch {
		err := fn(s.err)
		if err != nil {
			s.err = err
		}
	}
	s.mutex.Unlock()
}

func (s *Streams) Bind(bounder Stream) {
	s.mutex.Lock()
	s.outbounds = append(s.outbounds, bounder)
	s.mutex.Unlock()
}

func (s *Streams) Close() {
	if !s.IsClosed() {
		s.cancelFunc()
		s.Stream.Close()
	}
}

func (s *Streams) unHandle(v interface{}) {
	fmt.Println("unHandle ", v)
}

type RoundRobin struct {
	*Streams
	ring        *ring.Ring
	creatWorker func(context.Context, int) *Streams
}

func NewRoundRobin(ctx context.Context, buffSize, n int, creatWorker func(context.Context, int) *Streams) *RoundRobin {
	var roundRobin *RoundRobin
	loop := func(s *Streams) {
		for {
			select {
			case <-s.ctx.Done():
				s.Cancel()
				break
			case v := <-s.Stream:
				roundRobin.Resolve(v)
			}
		}
	}
	first := roundRobin.ring
	first.Value = creatWorker(ctx, buffSize)
	for next := first.Next(); next != first; next = next.Next() {
		next.Value = creatWorker(ctx, buffSize)
	}
	roundRobin = &RoundRobin{
		Streams:     newStreams(ctx, 0, loop),
		ring:        first,
		creatWorker: creatWorker,
	}

	return roundRobin
}

func (s *RoundRobin) Resolve(resolution interface{}) {
	s.mutex.Lock()
	if resolution != nil {
		switch resolution.(type) {
		case error:
			err := resolution.(error)
			s.mutex.Unlock()
			s.Reject(err)
		default:
			s.result = resolution
			cur := s.ring
			curStreams := cur.Value.(*Streams)
			if curStreams.Accept(s.result) {
				curStreams.Resolve(s.result)
				cur = cur.Next()
				s.mutex.Unlock()
				return
			}
			for next := cur.Next(); next != cur; next = next.Next() {
				nextStreams := next.Value.(*Streams)
				if nextStreams.Accept(s.result) {
					nextStreams.Resolve(s.result)
					s.mutex.Unlock()
					return
				}
			}
		}
	}
	s.mutex.Unlock()
	s.unHandle(s.result)
}
