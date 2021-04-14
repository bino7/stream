package stream

import (
	"context"
	"fmt"
	"testing"
)

func TestOnce(t *testing.T) {
	ctx := context.Background()
	s, err := Once("OnceTest", ctx, f1, f3, f1, f3)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	c := 0
	s.Then(func(v interface{}) (interface{}, error) {
		if c < 10 {
			c++
			return Goto(1), nil
		}
		return nil, nil

	})
	s.Resolve("wtf")
}

func TestStreams(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	s, err := NewStreams("OnceTest", ctx, 100, f1, f3)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	c := 0
	s.Then(func(v interface{}) (interface{}, error) {
		if c < 10 {
			c++
			return Goto(1), nil
		}
		return nil, nil
	})
	s.Input() <- "wtf"
	<-s.Done()
	t.Log("success!")
}

func f1(v interface{}) (interface{}, error) {
	v = fmt.Sprintf("foo-%s", v)
	fmt.Println(v)
	return v, nil
}
func f3(v interface{}) (interface{}, error) {
	v = fmt.Sprintf("foo3-%s", v)
	fmt.Println(v)
	return v, nil
}

func f2(v interface{}) (interface{}, error) {
	fmt.Println("END")
	return nil, nil
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
