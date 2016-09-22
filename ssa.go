package evmdis

import (
)

type Variable struct {
	Label      string
}

type Statement struct {
	Op         OpCode
	Inputs     []Variable
	Outputs    *Variable // Statements can have max one output on the stack.
}

func (block *BasicBlock) CompileSSA() {
}
