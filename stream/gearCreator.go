package stream

import (
	"container/ring"
	"context"
	"sync"
	"time"
)

type gearCreator struct {
	ctx          context.Context
	first        Gear
	last         Gear
	setFirstGear func(Gear)
}

func newGearCreator(ctx context.Context, gear Gear, setFirstGear func(Gear)) *gearCreator {
	return &gearCreator{ctx, gear, nil, setFirstGear}
}

func (c *gearCreator) add(g Gear) {
	if c.first == nil {
		c.first = g
		c.last = g
		if c.setFirstGear != nil {
			c.setFirstGear(g)
		}
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
		cur.Value.(Stream) <- v
		cur = cur.Next()
		return ignore
	})
	var wg sync.WaitGroup
	wg.Add(n)
	init := func(cur *ring.Ring) {
		go func(g Gear) {
			cur.Value = New(100)
			wg.Done()
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
	init(first)
	for cur := first.Next(); cur != first; cur = cur.Next() {
		init(cur)
	}
	wg.Wait()
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

func (c *gearCreator) Every(duration time.Duration, do func()) *gearCreator {
	started := false
	g := NewGear(c.ctx, func(v interface{}) interface{} {
		if !started {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				started = true
				wg.Done()
				ti := time.NewTicker(duration)
				select {
				case <-c.ctx.Done():
					ti.Stop()
					break
				case <-ti.C:
					do()
				}
			}()
			wg.Wait()
		}
		return v
	})
	c.add(g)
	return c
}
