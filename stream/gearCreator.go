package stream

import (
	"container/ring"
	"context"
)

type gearCreator struct {
	ctx   context.Context
	first Gear
	last  Gear
}

func newGearCreator(ctx context.Context, gear Gear) *gearCreator {
	return &gearCreator{ctx, gear, nil}
}

func (c *gearCreator) add(g Gear) {
	if c.first == nil {
		c.first = g
		c.last = g
	} else {
		c.last.Link(g)
		c.last = g
	}
}
func (c *gearCreator) Do(do func(interface{}) interface{}) *gearCreator {
	c.add(NewGear(c.ctx, do))
	return c
}
func (c *gearCreator) Filter(apply FilterableFunc) *gearCreator {
	g := NewGear(c.ctx, func(v interface{}) interface{} {
		if apply(v) {
			return v
		}
		return nil
	})

	c.add(g)
	return c
}
func (c *gearCreator) GroupBy(keySelectorFunc KeySelectorFunc, handlers GroupHandleFunc, defaultHandler func(k, v interface{}) interface{}) *gearCreator {
	c.add(newGearGroup(c.ctx, keySelectorFunc, handlers, defaultHandler))
	return c
}

type result int

const (
	ignore = result(iota)
)

func (c *gearCreator) RoundRobin(n int, do HandleFunc) *gearCreator {
	first := ring.New(n)
	cur := first
	g := NewGear(c.ctx, func(v interface{}) interface{} {
		if cur.Value == nil {
			cur.Value = make(chan interface{}, 1000)
		}
		cur.Value.(Stream) <- v
		cur = cur.Next()
		return ignore
	})
	for cur := first; cur.Next() != first; cur = cur.Next() {
		go func(g Gear) {
			select {
			case v := <-cur.Value.(Stream):
				nv := do(v)
				if g.Next() != nil {
					g.Next().Do(nv)
				}
			case <-g.Context().Done():
				break
			}
		}(g)
	}
	c.add(g)
	return c
}

func (c *gearCreator) Max(receiver interface{}, apply CompareFunc) *gearCreator {
	g := NewGear(c.ctx, func(v interface{}) interface{} {
		if receiver == nil || apply(receiver, v) < 0 {
			receiver = &v
		}
		return v
	})

	c.add(g)
	return c
}

func (c *gearCreator) Min(receiver interface{}, apply CompareFunc) *gearCreator {
	g := NewGear(c.ctx, func(v interface{}) interface{} {
		if receiver == nil || apply(receiver, v) > 0 {
			receiver = &v
		}
		return v
	})

	c.add(g)
	return c
}
