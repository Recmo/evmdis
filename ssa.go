package evmdis

import (
	"log"
	"fmt"
	"math/big"
)

type Expression interface {
}

type Constant struct {
	Expression
	Value      *big.Int
}

type PhiNode struct {
	Expression
}

type Variable struct {
	Expression
	Label      string
}

type Statement struct {
	Op         OpCode
	Inputs     []Expression
	Output     *Variable // Statements can have max one output on the stack.
}

func (constant Constant) String() string {
	return fmt.Sprintf("0x%X", constant.Value)
}

func (variable Variable) String() string {
	return variable.Label
}

func (statement Statement) String() string {
	str := ""
	if statement.Output != nil {
		str += fmt.Sprintf("%v = ", statement.Output)
	}
	str += fmt.Sprintf("%v(", statement.Op)
	for i, exp := range statement.Inputs {
		str += fmt.Sprintf("%v", exp)
		if i != len(statement.Inputs) - 1 {
			str += ", "
		}
	}
	str += ")"
	return str
}

type StatementBlock struct {
	Statements      []*Statement
	Label           string
	Inputs          []Variable
	Outputs         []Expression
}

type SSAProgram struct {
	Blocks          []*StatementBlock
}

// A counter for generating unique identifiers in the block's scope
var ssaCount int

func CompileSSABlock(block *BasicBlock) *StatementBlock {
	
	// Create the StatementBlock
	statements := &StatementBlock{
		Statements: make([]*Statement, 0),
		Label:      block.Label,
		Inputs:     make([]Variable, 0),
		Outputs:    nil,
	}
	
	// Create an abstract stack and load it with input variables
	stack := &Stack{
		Values:     make([]Expression, 0),
	}
	for i := 0; i < block.Reads; i++ {
		variable := Variable{
			Label: "abcdefghijklmnopqrstuvw"[i:i+1],
		}
		statements.Inputs = append(statements.Inputs, variable)
		stack.Push(variable)
	}
	
	// Label the block
	for _, instruction := range block.Instructions {
		
		// Stack management
		if instruction.Op.IsPush() {
			stack.Push(Constant{
				Value: instruction.Arg,
			})
			continue
		}
		if instruction.Op.IsSwap() {
			stack.Swap(instruction.Op.OperandSuffix())
			continue
		}
		if instruction.Op.IsDup() {
			stack.Dup(instruction.Op.OperandSuffix())
			continue
		}
		if instruction.Op == POP {
			stack.Pop()
			continue
		}
		
		// Create a new statement
		statement := &Statement{
			Op:       instruction.Op,
			Inputs:   make([]Expression, 0),
		}
		statements.Statements = append(statements.Statements, statement)
		
		// Pop the instruction inputs of the stack
		for i := 0; i < instruction.Op.StackReads(); i++ {
			statement.Inputs = append(statement.Inputs, stack.Pop())
		}
		
		// At this point, instructions either write one or zero outputs to
		// the stack. The only exceptions (DUP, SWAP) are handled above.
		if instruction.Op.StackWrites() > 1 {
			log.Fatalf("Instruction returning more than one: %v", instruction.Op)
		}
		if instruction.Op.StackWrites() == 1 {
			ssaCount++
			variable := Variable{
				Label: fmt.Sprintf("x%v", ssaCount),
			}
			stack.Push(variable)
			statement.Output = &variable
		}
		
		// Add offset as argument to JUMPDEST.
		if instruction.Op == JUMPDEST {
			statement.Inputs = append(statement.Inputs, Constant{
				Value: big.NewInt(int64(block.Offset)),
			})
		}
	}
	
	// Check if the stack is empty at the end of the StatementBlock
	for stack.Size() > 0 {
		statements.Outputs = append(statements.Outputs, stack.Pop())
	}
	
	return statements
}

func CompileSSA(program *Program) *SSAProgram {
	ssaCount = 0
	ssaProgram := &SSAProgram{
		Blocks: make([]*StatementBlock, 0),
	}
	for _, block := range program.Blocks {
		ssaProgram.Blocks = append(ssaProgram.Blocks, CompileSSABlock(block))
	}
	return ssaProgram
}

