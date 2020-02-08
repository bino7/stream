package stream

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestEngine(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	s := New(100)
	eg := NewEngine(ctx, s, nil)
	getHandler := func(i int) HandleFunc {
		n := i
		return func(v interface{}) interface{} {
			fmt.Println(n, ":", v)
			return nil
		}
	}
	i := 0
	eg.Create().RoundRobin(10, getHandler(i))
	eg.Start()
	go func() {
		i := 0
		for range time.Tick(1 * time.Second) {
			s <- i
			i++
		}
	}()
	<-ctx.Done()
}
