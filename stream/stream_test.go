package stream

import (
	"testing"
	"fmt"
	"time"
)

type Foo struct {
}

func (f *Foo) Handle(in interface{}) (interface{}, error) {
	fmt.Println(in)
	switch in {
	case "a":
		return "1", nil
	case "b":
		return "2", nil
	case "c":
		return "3", nil
	default:
		return nil, nil
	}

}
func Test(t *testing.T) {
	//var s In
	//s = Just("a", "b", "c")
	/*for r := range s.Handle(new(Foo)).Map(func(interface{}) interface{}{
		return "ok"
	}){
		fmt.Println(r)
	}*/
}

func Test_to_ch(t *testing.T) {
	s := Empty()
	log.Info("", "test", (chan interface{})(s))
}

func Test_join(t *testing.T) {
	s1 := Stream(make(chan interface{}))
	s2 := Stream(make(chan interface{}))
	Join(s1,s2)

	go func() {
		vs := []string{"1", "2", "3", "4", "5"}
		for _, v := range vs {
			s1 <- v
			time.Sleep(2 * time.Second)
		}
	}()

	go func() {
		vs := []string{"a", "b", "c", "d", "e"}
		for _, v := range vs {
			s1 <- v
			time.Sleep(3 * time.Second)
		}
	}()
	for v := range s1 {
		log.Info("test", "v", v)
	}
}

func Test_stream_vs_channel(t *testing.T) {
	foo := func(v interface{}) interface{} {
		num := v.(int)
		num++
		return num
	}
	start := time.Now()
	for range Range(0,1000) {
		Just(0).Map(foo).Map(foo).Map(foo).Map(foo).Map(foo).Map(foo).Map(foo).Map(foo).Map(foo).Map(foo)
	}
	elapsed := time.Since(start)
	fmt.Println(elapsed)

	foos := func(v interface{}) interface{} {
		return foo(foo(foo(foo(foo(foo(foo(foo(foo(foo(v))))))))))
	}
	start = time.Now()
	for range Range(0,1000) {
		Just(0).Map(foos)
	}
	elapsed = time.Since(start)
	fmt.Println(elapsed)
}

func Test_stream_close(t *testing.T) {
	s:=Stream(make(chan interface{}))
	foo:=func(v interface{})interface{}{return nil}
	out:=s.Map(foo)
	close(s)
	<- out
	select{}
}

func TestStream_Join(t *testing.T) {
	a:=Range(0,100)
	b:=Range(200,300)
	s:=Join(a,b)
	for v:= range s{
		fmt.Println(s,v,a,b)
	}
}

func TestStream_To(t *testing.T) {
	defer func(){
		r:=recover()
		if r!=nil {
			err:=r.(error)
			panic(err)
		}
	}()
	a:=make(chan interface{})
	close(a)
	a <- 100

}
