package stream

import (
	logger "github.com/inconshreveable/log15"
	"sync"
	"time"
)

var log = logger.New("module", "stream")

type Stream chan interface{}

func (s Stream) To(another Stream) {
	go func() {
		defer func() {
			r := recover()
			if r != nil {
				err := r.(error)
				if err.Error() != "send on closed channel" {
					panic(err)
				}
			}

		}()
		for {
			select {
			case v, ok := <-s:
				if ok {
					another <- v
				} else {
					break
				}
			}
		}
	}()
}
func (s Stream) Handle(apply HandleFunc) Stream {
	var wg sync.WaitGroup
	wg.Add(1)
	out := make(chan interface{})
	go func() {
		wg.Done()
		for v := range s {
			out <- apply(v)
		}
		close(out)
	}()
	wg.Wait()
	return Stream(out)
}

func (s Stream) Consume(apply ConsumeFunc) {
	go func() {
		for v := range s {
			apply(v)
		}
	}()
}

func (s Stream) Classify(apply ClassifyFunc, handlers map[string]Stream, deadHandler Stream) {
	go func() {
		for v := range s {
			class := apply(v)
			if handler, ok := handlers[class]; ok {
				handler <- v
			} else {
				deadHandler <- v
			}
		}
	}()
}

func (s Stream) Map(apply MappableFunc) Stream {
	out := make(chan interface{})
	go func() {
		for v := range s {
			out <- apply(v)
		}
		close(out)
	}()
	return Stream(out)
}

func (s Stream) Take(nth uint) Stream {
	out := make(chan interface{})
	go func() {
		takeCount := 0
		for item := range s {
			if takeCount < int(nth) {
				takeCount += 1
				out <- item
				continue
			}
			break
		}
		close(out)
	}()
	return Stream(out)
}

func (s Stream) TakeLast(nth uint) Stream {
	out := make(chan interface{})
	go func() {
		buf := make([]interface{}, nth)
		for item := range s {
			if len(buf) >= int(nth) {
				buf = buf[1:]
			}
			buf = append(buf, item)
		}
		for _, takenItem := range buf {
			out <- takenItem
		}
		close(out)
	}()
	return Stream(out)
}

func Just(item interface{}, items ...interface{}) Stream {
	source := make(chan interface{})
	if len(items) > 0 {
		items = append([]interface{}{item}, items...)
	} else {
		items = []interface{}{item}
	}

	go func() {
		for _, item := range items {
			source <- item
		}
		close(source)
	}()

	return Stream(source)
}

func Range(start, end int) Stream {
	source := make(chan interface{})
	go func() {
		i := start
		for i < end {
			source <- i
			i++
		}
		close(source)
	}()
	return Stream(source)
}

func Repeat(item interface{}, nTimes ...int) Stream {
	source := make(chan interface{})

	// this is the infinity case no ntime parameter is given
	if len(nTimes) == 0 {
		go func() {
			for {
				source <- item
			}
			close(source)
		}()
		return Stream(source)
	}

	// this repeat the item ntime
	if len(nTimes) > 0 {
		count := nTimes[0]
		if count <= 0 {
			return Empty()
		}
		go func() {
			for i := 0; i < count; i++ {
				source <- item
			}
			close(source)
		}()
		return Stream(source)
	}

	return Empty()
}

func Interval(term chan struct{}, interval time.Duration) Stream {
	source := make(chan interface{})
	go func(term chan struct{}) {
		i := 0
	OuterLoop:
		for {
			select {
			case <-term:
				break OuterLoop
			case <-time.After(interval):
				source <- i
			}
			i++
		}
		close(source)
	}(term)
	return Stream(source)
}

func Empty() Stream {
	source := make(chan interface{})
	go func() {
		close(source)
	}()
	return Stream(source)
}

func (s Stream) Scan(apply ScannableFunc) Stream {
	out := make(chan interface{})

	go func() {
		var current interface{}
		for item := range s {
			out <- apply(current, item)
			current = apply(current, item)
		}
		close(out)
	}()
	return Stream(out)
}

func (s Stream) SkipLast(nth uint) Stream {
	out := make(chan interface{})
	go func() {
		buf := make(chan interface{}, nth)
		for item := range s {
			select {
			case buf <- item:
			default:
				out <- <-buf
				buf <- item
			}
		}
		close(buf)
		close(out)
	}()
	return Stream(out)
}

func (s Stream) Skip(nth uint) Stream {
	out := make(chan interface{})
	go func() {
		skipCount := 0
		for item := range s {
			if skipCount < int(nth) {
				skipCount += 1
				continue
			}
			out <- item
		}
		close(out)
	}()
	return Stream(out)
}

/*func (s Stream) Sort(apply lib.CompareFunc) Stream{
	heap:=lib.NewHeap(apply)
	for v:=range s {
		heap.Push(v)
	}
}*/

func (s Stream) DistinctUntilChanged(apply KeySelectorFunc) Stream {
	out := make(chan interface{})
	go func() {
		var current interface{}
		for item := range s {
			key := apply(item)
			if current != key {
				out <- item
				current = key
			}
		}
		close(out)
	}()
	return Stream(out)
}

func (s Stream) Distinct(apply KeySelectorFunc) Stream {
	out := make(chan interface{})
	go func() {
		keysets := make(map[interface{}]struct{})
		for item := range s {
			key := apply(item)
			_, ok := keysets[key]
			if !ok {
				out <- item
			}
			keysets[key] = struct{}{}
		}
		close(out)
	}()
	return Stream(out)
}

func (s Stream) First() Stream {
	out := make(chan interface{})
	go func() {
		for item := range s {
			out <- item
			break
		}
		close(out)
	}()
	return Stream(out)
}

func (s Stream) Filter(apply FilterableFunc) Stream {
	out := make(chan interface{})
	go func() {
		for item := range s {
			if apply(item) {
				out <- item
			}
		}
		close(out)
	}()
	return Stream(out)
}

func Join(a, b Stream) Stream {
	out := make(chan interface{})
	go func() {
		for a != nil || b != nil {
			select {
			case v, ok := <-a:
				if !ok {
					a = nil
					continue
				}
				out <- v
			case v, ok := <-b:
				if !ok {
					b = nil
					continue
				}
				out <- v
			}
		}
		close(out)
	}()
	return Stream(out)
}

func (s Stream) AsChan() chan interface{} {
	return (chan interface{})(s)
}

func FromOutbound(outbound <-chan interface{}) Stream {
	out := make(chan interface{})
	go func() {
		for item := range outbound {
			out <- item
			break
		}
		close(out)
	}()
	return Stream(out)
}

func Source(apply EmittableFunc) Stream {
	out := make(chan interface{})
	go func() {
		for {
			for item := apply(); item != nil; item = apply() {
				out <- item
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()
	return Stream(out)
}
