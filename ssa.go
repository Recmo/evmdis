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
	Offset          int
	Statements      []*Statement
	Label           string
	Inputs          []Variable
	Outputs         []Expression
	Incoming        []*StatementBlock
	CondBlocks      []*StatementBlock
	NextBlock       *StatementBlock
}

type SSAProgram struct {
	Blocks          []*StatementBlock
}

func (block *StatementBlock) CanGoToNext() bool {
	n := len(block.Statements)
	if n == 0 {
		return false
	}
	last := block.Statements[n - 1]
	switch last.Op {
	case JUMP, RETURN, SELFDESTRUCT, STOP:
		return false
	default:
		return true
	}
}

// A counter for generating unique identifiers in the block's scope
var ssaCount int

func CompileSSABlock(block *BasicBlock) *StatementBlock {
	
	// Create the StatementBlock
	statements := &StatementBlock{
		Offset:     block.Offset,
		Label:      block.Label,
		Statements: make([]*Statement, 0),
		Inputs:     make([]Variable, 0),
		Outputs:    nil,
		Incoming:   make([]*StatementBlock, 0),
		CondBlocks: make([]*StatementBlock, 0),
		NextBlock:  nil,
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
	
	// Add compile assembly blocks to SSA
	for _, block := range program.Blocks {
		ssaProgram.Blocks = append(ssaProgram.Blocks, CompileSSABlock(block))
		n := len(ssaProgram.Blocks) - 1
		
		// Under certain condition the blocks can continue to the next
		if n > 0 && ssaProgram.Blocks[n - 1].CanGoToNext() {
			ssaProgram.Blocks[n - 1].NextBlock = ssaProgram.Blocks[n]
		}
	}
	
	// Add fake error block at offset 2. This is the default
	// jump target for errors in solidity
	ssaProgram.Blocks = append(ssaProgram.Blocks, &StatementBlock{
		Offset:     2,
		Label:      "ErrorTag",
		Statements: make([]*Statement, 0),
		Inputs:     make([]Variable, 0),
		Outputs:    nil,
		Incoming:   make([]*StatementBlock, 0),
		CondBlocks: make([]*StatementBlock, 0),
		NextBlock:  nil,
	})
	
	return ssaProgram
}

func (ssa *SSAProgram) BlockByOffset(offset int) *StatementBlock {
	for _, block := range ssa.Blocks {
		if block.Offset == offset {
			return block
		}
	}
	return nil
}

func (ssa *SSAProgram) UpdateJumpTargets(block *StatementBlock) {
	
	// Clear existing
	block.CondBlocks = make([]*StatementBlock, 0)
	
	// All statements
	for _, statement := range block.Statements {
		
		// Filter JUMPS
		if statement.Op != JUMP && statement.Op != JUMPI {
			continue
		}
		
		// Filter fixed JUMPS
		constant, ok:= statement.Inputs[0].(Constant)
		if !ok {
			continue
		}
		target := int(constant.Value.Int64())
		
		// Find the target block
		targetBlock := ssa.BlockByOffset(target)
		if statement.Op == JUMPI {
			block.CondBlocks = append(block.CondBlocks, targetBlock)
		} else {
			block.NextBlock = targetBlock
		}
	}
}

func (ssa *SSAProgram) ComputeJumpTargets() {
	for _, block := range ssa.Blocks {
		ssa.UpdateJumpTargets(block)
	}
}

func (ssa *SSAProgram) ComputeIncoming() {
	for _, block := range ssa.Blocks {
		block.Incoming = make([]*StatementBlock, 0)
		for _, source := range ssa.Blocks {
			isTarget := false
			if source.NextBlock == block {
				isTarget = true
			}
			for _, sourceCond := range source.CondBlocks {
				if sourceCond == block {
					isTarget = true
				}
			}
			if isTarget {
				block.Incoming = append(block.Incoming, source)
			}
		}
	}
}

func (ssa *SSAProgram) CollapseJumps() {
	// This function tries to simplify the SSA by joining blocks
	// when they will always follow eachother. This is done by
	// looking at the incoming, if block_1 ends in a `JUMP block_2`
	// And no other code can flow or jump into `block_2`, we can
	// put `block_2` at the end of `block_1`.
	//
	// @NOTE: This requires finding all the jump targets. Since
	//        jump targets may be computed, we can never do this
	//        perfectly.
	
	for _, block := range ssa.Blocks {
		if len(block.Incoming) != 1 {
			continue
		}
		
		// We want to be next, not the target of a conditional jump
		source := block.Incoming[0]
		if source.NextBlock != block {
			continue
		}
		
		// Okay, merge the blocks!
		fmt.Printf("MERGING %v %v\n", source.Label, block.Label)
		
		
		for _, statement := range block.Statements {
			fmt.Printf("%v\n", statement)
			source.Statements = append(source.Statements, statement)
		}
		
		// Rewire the connections
		source.NextBlock = block.NextBlock
		ssa.UpdateJumpTargets(source)
		block.Label += " REMOVED"
		
		
	}
}
