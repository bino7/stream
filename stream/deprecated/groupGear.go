package deprecated

import (
	"context"
	"github.com/bino7/stream/stream"
)

type groupGear struct {
	parent          context.Context
	next            Gear
	subGears        map[interface{}]Gear
	keySelectorFunc stream.KeySelectorFunc
	handlers        stream.GroupHandleFunc
	defaultHandler  func(key, v interface{}) interface{}
}

/*func newGearGroup(parent context.Context, keySelectorFunc KeySelectorFunc, handlers GroupHandleFunc,
	defaultHandler func(k, v interface{}) interface{}) Gear {
	return &groupGear{parent, nil, make(map[interface{}]Gear), keySelectorFunc,
		handlers, defaultHandler}
}
func (g *groupGear) Context() context.Context {
	return g.parent
}
func (g *groupGear) Do(v interface{}) {
	key := g.keySelectorFunc(v)
	var sg Gear
	c := func(k interface{}) HandleFunc {
		h, ok := g.handlers[k]
		if !ok {
			h = func(v interface{}) interface{} {
				return g.defaultHandler(k, v)
			}
		}
		return h
	}
	if g1, ok := g.subGears[key]; !ok {
		sg = NewGear(g.Context(), c(key))
		g.subGears[key] = sg
	} else {
		sg = g1
	}
	sg.Do(v)
}

func (g *groupGear) Link(next Gear) Gear {
	for _, sg := range g.subGears {
		sg.Link(next)
	}
	return next
}

func (g *groupGear) Next() Gear {
	return g.next
}

func (g *groupGear) At(n int) Gear {
	return nextNthGear(g, n)
}

func (g *groupGear) Group(key interface{}) (Gear, error) {
	if g0, ok := g.subGears[key]; ok {
		return g0, nil
	}
	return nil, nil
}

func (g *groupGear) Create() *GearCreator {
	return newGearCreator(g.parent, g, nil)
}*/
