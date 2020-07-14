package stream

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestF(t *testing.T) {
	var v []string
	fmt.Println(reflect.TypeOf(v).Kind())

}
func TestWith(t *testing.T) {
	done := make(chan struct{})
	ctx := context.Background()
	ctx = context.WithValue(ctx, "Before-1st", HandleFunc(func(v interface{}) (interface{}, error) {
		fmt.Println("Before foo")
		return v, nil
	}))
	s := Static("test", ctx,
		&Handler{
			Name:  "1st",
			Apply: foo,
			Pipes: []Stream{
				With(ctx, 100,
					&Handler{
						Name:  "3st",
						Apply: foo3,
						Pipes: nil,
					}).Input(),
			},
		},
		&Handler{
			Name:  "2nd",
			Apply: HandleFunc(foo),
			Pipes: []Stream{},
		},
	)
	s.Then(func(v interface{}) interface{} {
		fmt.Println(v)
		return v
	})
	s.Resolve("wtf")
	go func() {
		for range time.Tick(1 * time.Second) {
		}
	}()
	<-done
}

func foo(v interface{}) (interface{}, error) {
	v = fmt.Sprintf("foo-%s", v)
	fmt.Println(v)
	return v, nil
}
func foo3(v interface{}) (interface{}, error) {
	v = fmt.Sprintf("foo3-%s", v)
	fmt.Println(v)
	return v, nil
}

func TestFoo(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "Before-1st", HandleFunc(func(v interface{}) (interface{}, error) {
		fmt.Println("Before foo")
		return v, nil
	}))
	h1 := &Handler{
		Name:  "1st",
		Apply: foo,
		Pipes: []Stream{New(10), New(10), New(10)},
	}
	fmt.Println(h1.Before == nil, f(h1.Before, ctx.Value("Before-1st")), ctx.Value("Before-1st"))
}

func f(vs ...interface{}) interface{} {
	for _, v := range vs {
		fmt.Println(v)
		if v != nil {
			fmt.Println("return", v, v != nil, v == nil)
			return v
		}
	}
	return nil
}
