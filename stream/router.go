package stream

import (
	"context"
	"sync"
)

type Key interface{}

type Router struct {
	*gear
	mu              sync.Mutex
	keySelectorFunc KeySelectorFunc
	defaultHandler  func(k, v interface{}) interface{}
	gears           map[Key]Gear
}

func NewRouter(ctx context.Context, keySelectorFunc KeySelectorFunc, gears map[Key]Gear,
	defaultHandler func(k, v interface{}) interface{}) *Router {
	var mu sync.Mutex
	if gears == nil {
		gears = make(map[Key]Gear)
	}
	return &Router{&gear{ctx: ctx}, mu, keySelectorFunc, defaultHandler,
		gears}
}

func (r *Router) Do(v interface{}) interface{} {
	key := r.keySelectorFunc(v)
	for k, g := range r.gears {
		if k == key && v != nil {
			g := g.(Gear)
			result := g.Do(v)
			return r.gear.Do(result)
		}
	}
	return r.defaultHandler(key, v)
}

func (r *Router) Set(k Key, g Gear) *Router {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gears[k] = g
	return r
}
