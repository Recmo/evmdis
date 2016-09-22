package evmdis

type Stack struct {
	values   []string
}

func CreateStack(arguments int) *Stack {
	stack := &Stack{
		values: make([]string, 0),
	}
	for i := 0; i < arguments; i++ {
		stack.Push("abcdefghijklmnopqrstuvw"[i:i+1])
	}
	return stack
}

func (stack *Stack) Size() int {
	return len(stack.values)
}

func (stack *Stack) Push(value string) {
	stack.values = append(stack.values, value)
}

func (stack *Stack) Pop() string {
	n := len(stack.values) - 1
	value := stack.values[n]
	stack.values = stack.values[:n]
	return value
}

func (stack *Stack) Dup(num int) {
	n := len(stack.values) - 1
	stack.Push(stack.values[n - num + 1])
}

func (stack *Stack) Swap(num int) {
	n := len(stack.values) - 1
	tmp := stack.values[n]
	stack.values[n] = stack.values[n - num]
	stack.values[n - num] = tmp
}
