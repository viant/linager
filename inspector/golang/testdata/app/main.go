package main

import (
	"fmt"
	"myapp/stack"
	"myapp/util"
)

func main() {
	// Using a generic stack of strings
	s := stack.New[string]()
	s.Push("first")
	s.Push("second")

	top, _ := s.Pop()
	fmt.Println("Popped:", top)

	// Call utility inspector to reflect on the stack
	util.Inspect(s)
}
