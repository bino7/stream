package stream

import (
	"errors"
	"reflect"
)

type handler interface {
	Eval(interface{}) (interface{}, error)
}

type Handler struct {
	Apply  HandleFunc
	Name   string
	Before HandleFunc
	After  HandleFunc
	Pipes  []Stream
}

func (h *Handler) Eval(v interface{}) (interface{}, error) {
	return h.Apply(v)
}

type SliceHandler struct {
	Apply  HandleFunc
	Name   string
	Before HandleFunc
	After  HandleFunc
	Pipes  []Stream
}

var (
	InputNotASliceError = errors.New("input is not a slice")
)

func (h *SliceHandler) Eval(v interface{}) (interface{}, error) {
	t := reflect.TypeOf(v)
	if k := t.Kind(); k == reflect.Slice || k == reflect.Array {
		m := reflect.ValueOf(v)
		result := make([]interface{}, m.Len())
		for i := 0; i < m.Len(); i++ {
			ele := m.Index(i)
			r, err := h.Apply(ele)
			if err != nil {
				return nil, err
			}
			result = append(result, r)
		}
		return result, nil
	}
	return nil, InputNotASliceError
}
