package stream

import (
	"context"
	"fmt"
)

type (
	Gear interface {
		Context() context.Context
		Do(interface{}) interface{}
		Link(Gear) Gear
		Next() Gear
		At(int) Gear
		Create() *GearCreator
	}

	gear struct {
		ctx  context.Context
		do   HandleFunc
		next Gear
	}
)

func NewGear(ctx context.Context, do func(interface{}) interface{}) Gear {
	return &gear{ctx, do, nil}
}
func (g *gear) Context() context.Context {
	return g.ctx
}
func (g *gear) Do(v interface{}) interface{} {
	var nv interface{}
	if g.do == nil {
		nv = v
	} else {
		nv = g.do(v)
	}
	if nv != nil && nv != ignore && g.next != nil {
		return g.next.Do(nv)
	}
	return nv
}

func (g *gear) Link(next Gear) Gear {
	if g.next != nil {
		next.Link(g.next)
	}
	g.next = next
	return next
}

func (g *gear) Next() Gear {
	return g.next
}

func (g *gear) At(n int) Gear {
	return nextNthGear(g, n)
}

func (g *gear) Group(key interface{}) (Gear, error) {
	return nil, NotGroupGearErr
}

func (g *gear) Create() *GearCreator {
	return newGearCreator(g.ctx, g, nil)
}

func nextNthGear(g Gear, n int) Gear {
	if g == nil || n < 0 {
		return g
	}
	g0 := g
	for i := 1; i < n; i++ {
		if g0.Next() == nil {
			return nil
		}
		g0 = g0.Next()
	}
	return g0
}

var NotGroupGearErr = fmt.Errorf("current gear not a group gear")
