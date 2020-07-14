package deprecated

import (
	"context"
	"fmt"
	"github.com/bino7/stream/stream"
	"testing"
	"time"
)

func TestEngine(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	s := stream.New(100)
	eg := NewEngine(ctx, s, nil)
	getHandler := func(i int) stream.HandleFunc {
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
