package stream

import (
	"fmt"
	"reflect"
)

type _goto struct {
	step int
}

func toHandlers(steps ...interface{}) ([]*handler, error) {
	handlers := make([]*handler, 0)
	step := 0
	for _, s := range steps {
		switch s.(type) {
		case *handler:
			h := s.(*handler)
			h.step = step
			handlers = append(handlers, h)
		case HandleFunc:
			handlers = appendToHandlers(handlers, s.(HandleFunc), step)
		case func(interface{}) (interface{}, error):
			fn := s.(func(interface{}) (interface{}, error))
			handlers = appendToHandlers(handlers, HandleFunc(fn), step)
		case Stream:
			handlers = appendToHandlers(handlers, streamHandlerFunc(s.(Stream)), step)
		default:
			return nil, fmt.Errorf("only support types:HandleFunc,func(interface{}) (interface{}, error),Stream,has %v", reflect.TypeOf(s))
		}
		step++
	}

	return handlers, nil
}

func appendToHandlers(handlers []*handler, h HandleFunc, step int) []*handler {
	handlers = append(handlers, &handler{
		apply: h,
		name:  "",
		step:  step,
	})
	return handlers
}
