package deprecated

import (
	"container/list"
	"container/ring"
	"context"
	"fmt"
	"github.com/bino7/stream/stream"
	"log"
	"sync"
)

type Resolve struct {
	Val interface{}
}

type Streams interface {
	Input() stream.Stream
	Put(interface{})
	Cancel()
	Then(apply stream.HandleFunc) Streams
	Catch(apply stream.ErrorHandleFunc) Streams
	Filter(apply stream.FilterableFunc) Streams
	When(test stream.FilterableFunc, apply stream.HandleFunc) Streams
	Max(receiver interface{}, apply stream.CompareFunc) Streams
	Min(receiver interface{}, apply stream.CompareFunc) Streams
	Resolve(resolution interface{})
	Reject(err error)
	Bind(bounder stream.Stream)
	Close()
}

type streams struct {
	stream.Stream
	state                int
	ctx                  context.Context
	cancelFunc           context.CancelFunc
	then                 []stream.HandleFunc
	catch                []stream.ErrorHandleFunc
	outbounds            []stream.Stream
	result               interface{}
	err                  error
	stack                *list.List
	mutex                *sync.Mutex
	broadcastDeadMessage bool
}

func NewStreams(ctx context.Context, buffSize int, broadcastDeadMessage bool) Streams {
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
	return newStreams(ctx, buffSize, broadcastDeadMessage, loop)
}

func newStreams(ctx context.Context, n int, broadcastDeadMessage bool, loop func(*streams)) Streams {
	ctx, cancelFunc := context.WithCancel(ctx)
	var s = &streams{
		Stream:               stream.New(n),
		ctx:                  ctx,
		cancelFunc:           cancelFunc,
		then:                 make([]stream.HandleFunc, 0),
		catch:                make([]stream.ErrorHandleFunc, 0),
		outbounds:            make([]stream.Stream, 0),
		result:               nil,
		err:                  nil,
		stack:                list.New(),
		mutex:                &sync.Mutex{},
		broadcastDeadMessage: broadcastDeadMessage,
	}
	go loop(s)
	return s
}

func (s *streams) Input() stream.Stream {
	return s.Stream
}

func (s *streams) Put(v interface{}) {
	s.Stream <- v
}

func (s *streams) Cancel() {
	s.cancelFunc()
}

func (s *streams) Then(apply stream.HandleFunc) Streams {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.then = append(s.then, apply)
	return s
}

func (s *streams) Catch(apply stream.ErrorHandleFunc) Streams {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.catch = append(s.catch, apply)
	return s
}

func (s *streams) Filter(apply stream.FilterableFunc) Streams {
	s.Then(func(v interface{}) (interface{}, error) {
		if apply(v) {
			return v, nil
		}
		return nil, nil
	})
	return s
}

func (s *streams) When(test stream.FilterableFunc, apply stream.HandleFunc) Streams {
	s.Then(func(v interface{}) (interface{}, error) {
		if test(v) {
			return apply(v)
		}
		return v, nil
	})
	return s
}

func (s *streams) On(test stream.FilterableFunc, apply stream.HandleFunc) Streams {
	s.Then(func(v interface{}) (interface{}, error) {
		if test(v) {
			apply(v)
		}
		return v, nil
	})
	return s
}

func (s *streams) Max(receiver interface{}, apply stream.CompareFunc) Streams {
	s.Then(func(v interface{}) (interface{}, error) {
		if receiver == nil || apply(receiver, v) < 0 {
			receiver = &v
		}
		return v, nil
	})
	return s
}

func (s *streams) Min(receiver interface{}, apply stream.CompareFunc) Streams {
	s.Then(func(v interface{}) (interface{}, error) {
		if receiver == nil || apply(receiver, v) > 0 {
			receiver = &v
		}
		return v, nil
	})
	return s
}

func (s *streams) Resolve(resolution interface{}) {
	if resolution == nil {
		return
	}
	s.mutex.Lock()
	s.stack.PushBack(resolution)
	s.resolve(resolution)
	s.mutex.Unlock()
}

func (s *streams) resolve(resolution interface{}) {
	s.result = resolution

	for _, fn := range s.then {
		s.result, s.err = fn(s.result)
		if s.err != nil {
			s.mutex.Unlock()
			s.Reject(s.err)
			break
		}
		switch s.result.(type) {

		case error:
			err := s.result.(error)
			s.mutex.Unlock()
			s.Reject(err)
			break
		}
	}

	/*if s.result!=nil && s.broadcast(0) == false {
		s.unHandle(s.result)
	}*/
}

func (s *streams) broadcast(i int) bool {
	log.Println("broadcast")
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

func (s *streams) Reject(err error) {
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

func (s *streams) Bind(bounder stream.Stream) {
	s.mutex.Lock()
	s.outbounds = append(s.outbounds, bounder)
	s.mutex.Unlock()
}

func (s *streams) Close() {
	if !s.IsClosed() {
		s.cancelFunc()
		s.Stream.Close()
	}
}

func (s *streams) unHandle(v interface{}) {
	fmt.Println("unHandle ", v)
}

type RoundRobin struct {
	*streams
	ring        *ring.Ring
	creatWorker func(context.Context, int) *streams
}

func NewRoundRobin(ctx context.Context, buffSize, n int, creatWorker func(context.Context, int) *streams) *RoundRobin {
	var roundRobin *RoundRobin
	loop := func(s *streams) {
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

	ctx, cancelFunc := context.WithCancel(ctx)
	var s = &streams{
		Stream:               stream.New(n),
		ctx:                  ctx,
		cancelFunc:           cancelFunc,
		then:                 make([]stream.HandleFunc, 0),
		catch:                make([]stream.ErrorHandleFunc, 0),
		outbounds:            make([]stream.Stream, 0),
		result:               nil,
		err:                  nil,
		mutex:                &sync.Mutex{},
		broadcastDeadMessage: false,
	}
	go loop(s)

	roundRobin = &RoundRobin{
		streams:     s,
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
			curStreams := cur.Value.(*streams)
			if curStreams.Accept(s.result) {
				curStreams.Resolve(s.result)
				cur = cur.Next()
				s.mutex.Unlock()
				return
			}
			for next := cur.Next(); next != cur; next = next.Next() {
				nextStreams := next.Value.(*streams)
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
