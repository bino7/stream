package stream

import (
	"container/list"
	"fmt"
	"strings"
	"sync"
)

/*type Handler interface {
	Apply(interface{}) (interface{}, error)
	Name() string
	Step() int
	Path() string
}*/

type handler struct {
	apply      HandleFunc
	name       string
	step       int
	prefixFunc func() string
}

func (h *handler) Apply(v interface{}) (interface{}, error) {
	return h.apply(v)
}
func (h *handler) Name() string {
	if h.name == "" {
		return h.Path()
	}
	return h.name
}
func (h *handler) Path() string {
	prefix := ""
	if h.prefixFunc != nil {
		prefix = h.prefixFunc()
	}
	name := h.name
	if name == "" {
		name = fmt.Sprintf("%d", h.step)
	}
	if strings.HasPrefix(prefix, "s.") {
		return fmt.Sprintf("%s.%s", prefix, name)
	} else {
		return fmt.Sprintf("s.%s.%s", prefix, name)
	}
}
func (h *handler) Step() int {
	return h.step
}
func (h *handler) SetPrefix(prefixFunc func() string) {
	h.prefixFunc = prefixFunc
}

type handlerList struct {
	*handler
	handlers *list.List
	mu       sync.Mutex
}

func newHandlerList(name string, step int) *handlerList {
	var mu sync.Mutex
	return &handlerList{
		handler: &handler{
			name: name,
			step: step,
		},
		handlers: list.New(),
		mu:       mu,
	}
}

func (hl *handlerList) forEach(apply func(i int, h *handler) bool) {
	e := hl.handlers.Front()
	i := 0
	for e != nil {
		h := e.Value.(*handler)
		if apply(i, h) == false {
			return
		}
	}
}

func (hl *handlerList) Apply(v interface{}) (interface{}, error) {
	result := v
	var err error
	hl.forEach(func(i int, h *handler) bool {
		result, err = h.Apply(result)
		return result != nil && err == nil
	})
	return result, err
}

func (hl *handlerList) Name() string {
	if hl.name == "" {
		return fmt.Sprintf("s.%d", hl.step)
	}
	return hl.name
}

func (hl *handlerList) Step() int {
	return hl.step
}

func (hl *handlerList) Len() int {
	return hl.handlers.Len()
}

func (hl *handlerList) PushBack(h *handler) {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	h.step = hl.Len()
	h.SetPrefix(func() string {
		return h.Path()
	})
	hl.handlers.PushBack(h)
}

func (hl *handlerList) Remove(step int) bool {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	e := hl.handlers.Front()
	i := 0
	for e != nil && i < step {
		e = e.Next()
		i++
	}
	if e != nil && i == step {
		hl.handlers.Remove(e)
		return true
	}
	return false
}

func (hl *handlerList) Insert(step int, h *handler) bool {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	e := hl.handlers.Front()
	i := 0
	for e != nil && i < step {
		e = e.Next()
		i++
	}
	if e != nil && i == step {
		h.step = hl.Len()
		h.SetPrefix(func() string {
			return h.Path()
		})
		hl.handlers.InsertBefore(h, e)
		return true
	}
	return false
}

func Goto(step int) *_goto {
	return &_goto{
		step,
	}
}

func streamHandlerFunc(s Stream) HandleFunc {
	return func(v interface{}) (interface{}, error) {
		if !s.IsClosed() {
			s <- v
		}
		return v, nil
	}
}
