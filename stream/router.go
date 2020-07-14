package stream

import (
	"context"
	"github.com/bino7/stream/stream/deprecated"
	"sync"
)

type Key interface{}

type Router struct {
	*deprecated.gear
	mu              sync.Mutex
	keySelectorFunc KeySelectorFunc
	defaultHandler  func(k, v interface{}) interface{}
	gears           map[Key]deprecated.Gear
}

func NewRouter(ctx context.Context, keySelectorFunc KeySelectorFunc, gears map[Key]deprecated.Gear,
	defaultHandler func(k, v interface{}) interface{}) *Router {
	var mu sync.Mutex
	if gears == nil {
		gears = make(map[Key]deprecated.Gear)
	}
	return &Router{&deprecated.gear{ctx: ctx}, mu, keySelectorFunc, defaultHandler,
		gears}
}

func (r *Router) Do(v interface{}) interface{} {
	key := r.keySelectorFunc(v)
	for k, g := range r.gears {
		if k == key && v != nil {
			g := g.(deprecated.Gear)
			result := g.Do(v)
			return r.gear.Do(result)
		}
	}
	return r.defaultHandler(key, v)
}

func (r *Router) Set(k Key, g deprecated.Gear) *Router {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gears[k] = g
	return r
}
