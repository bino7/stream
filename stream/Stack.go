package stream

import "container/list"

type Stack struct {
	stack *list.List
}

func NewStack() *Stack {
	var stack = &Stack{
		stack: list.New(),
	}
	return stack
}

func (s *Stack) Push(v interface{}) {
	s.stack.PushFront(v)
}

func (s *Stack) Pop(v interface{}) interface{} {
	front := s.stack.Front()
	if front != nil {
		s.stack.Remove(front)
		return front.Value
	}
	return nil
}

func (s *Stack) Len() int {
	return s.stack.Len()
}

func (s *Stack) CheckCirculation(n int, compareFunc CompareFunc) bool {
	if s.Len()/n < 2 {
		return false
	}
	//todo
	return true
}
