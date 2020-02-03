package stream

import (
	"context"
	"fmt"
)

type (
	Gear interface {
		Context() context.Context
		Do(interface{})
		Link(Gear) Gear
		Next() Gear
		At(int) Gear
		Group(key interface{}) (Gear, error)
		Create() *gearCreator
	}

	gear struct {
		parent context.Context
		do     HandleFunc
		next   Gear
	}
)

func NewGear(parent context.Context, do func(interface{}) interface{}) Gear {
	return &gear{parent, do, nil}
}
func (g *gear) Context() context.Context {
	return g.parent
}
func (g *gear) Do(v interface{}) {
	nv := g.do(v)
	if nv != nil && nv != ignore {
		g.next.Do(v)
	}
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

func (g *gear) Create() *gearCreator {
	return newGearCreator(g.parent, g, nil)
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
