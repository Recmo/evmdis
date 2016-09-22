package evmdis

import (
	"fmt"
	"math/big"
)

type Instruction struct {
	Op          OpCode
	Arg         *big.Int
	Annotations *TypeMap
}

func (self *Instruction) String() string {
	if self.Arg != nil {
		return fmt.Sprintf("%v 0x%x", self.Op, self.Arg)
	} else {
		return self.Op.String()
	}
}

type BasicBlock struct {
	Instructions    []Instruction
	Offset          int
	Reads           int
	Writes          int
	Next            *BasicBlock
	Annotations     *TypeMap
}

type Program struct {
	Blocks              []*BasicBlock
	JumpDestinations    map[int]*BasicBlock
	//Instructions map[int]*Instruction
}

func NewProgram(bytecode []byte) *Program {
	program := &Program{
		JumpDestinations:   make(map[int]*BasicBlock),
	}
	
	currentBlock := &BasicBlock{
		Offset: 0,
		Reads: 0,
		Annotations: NewTypeMap(),
	}
	
	var currentStackIndex = 0
	for i := 0; i < len(bytecode); i++ {
		
		// Read next opcode and optional argument
		op := OpCode(bytecode[i])
		size := op.OperandSize()
		var arg *big.Int
		if size > 0 {
			arg = big.NewInt(0)
			for j := 1; j <= size; j++ {
				arg.Lsh(arg, 8)
				if i + j < len(bytecode) {
					arg.Or(arg, big.NewInt(int64(bytecode[i + j])))
				}
			}
		}
		
		// Start a new basic block on reaching a JUMPDEST
	    if op == JUMPDEST {
			if len(currentBlock.Instructions) > 0 {
				program.Blocks = append(program.Blocks, currentBlock)
				newBlock := &BasicBlock{
					Offset: i,
					Reads: 0,
					Annotations: NewTypeMap(),
				}
				currentBlock.Next = newBlock
				currentBlock = newBlock
			}
			currentBlock.Offset += 1
			currentStackIndex = 0
			
			// Store the jump destination in a program global list
			program.JumpDestinations[i] = currentBlock
			fmt.Printf("Jump destination: %2x\n", i)
	    }
		
		// Add a new instruction to the current block
		instruction := Instruction{
			Op: op,
			Arg: arg,
			Annotations: NewTypeMap(),
		}
		currentBlock.Instructions = append(currentBlock.Instructions, instruction)
		
		// Update the current block's max stack read depth
		currentStackIndex -= op.StackReads()
		if currentStackIndex < 0 && (-currentStackIndex) > currentBlock.Reads {
			currentBlock.Reads = -currentStackIndex
		}
		
		// Update stack index
		currentStackIndex += op.StackWrites()
		currentBlock.Writes = currentStackIndex + currentBlock.Reads
		
		// Start a new basic block after a control flow statement
		if op.IsControlFlow() {
			program.Blocks = append(program.Blocks, currentBlock)
			newBlock := &BasicBlock{
				Offset: i + size + 1,
				Reads: 0,
				Annotations: NewTypeMap(),
			}
			currentBlock.Next = newBlock
			currentBlock = newBlock
			currentStackIndex = 0
		}
		i += size
	}
	
	if len(currentBlock.Instructions) > 0 || program.JumpDestinations[currentBlock.Offset] != nil {
		program.Blocks = append(program.Blocks, currentBlock)
	} else {
		program.Blocks[len(program.Blocks) - 1].Next = nil
	}
	
	fmt.Printf("Found %v basic blocks\n", len(program.Blocks));
	
	return program
}
