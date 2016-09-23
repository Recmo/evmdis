package evmdis

import (
	"fmt"
)

type Stack struct {
	Values   []Expression
}

func (stack Stack) String() string {
	return fmt.Sprintf("%v", stack.Values)
}

func (stack *Stack) Size() int {
	return len(stack.Values)
}

func (stack *Stack) Push(value Expression) {
	stack.Values = append(stack.Values, value)
}

func (stack *Stack) Pop() Expression {
	n := len(stack.Values) - 1
	value := stack.Values[n]
	stack.Values = stack.Values[:n]
	return value
}

func (stack *Stack) Dup(num int) {
	n := len(stack.Values) - 1
	stack.Push(stack.Values[n - num + 1])
}

func (stack *Stack) Swap(num int) {
	n := len(stack.Values) - 1
	tmp := stack.Values[n]
	stack.Values[n] = stack.Values[n - num]
	stack.Values[n - num] = tmp
}
