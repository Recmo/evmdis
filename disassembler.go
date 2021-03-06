package evmdis

import (
	"fmt"
	"math/big"
)

type Instruction struct {
	Op              OpCode
	Arg             *big.Int
	Annotations     *TypeMap
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
	Label           string
	Offset          int
	Reads           int
	Writes          int
}

type Program struct {
	Blocks          []*BasicBlock
}

func NewProgram(bytecode []byte) *Program {
	program := &Program{}
	currentBlock := &BasicBlock{
		Label: fmt.Sprintf("block_%v", len(program.Blocks)),
		Offset: 0,
		Reads: 0,
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
					Label: fmt.Sprintf("block_%v", len(program.Blocks)),
					Offset: i,
					Reads: 0,
				}
				currentBlock = newBlock
			}
			currentStackIndex = 0
	    }
		
		// Add a new instruction to the current block
		instruction := Instruction{
			Op: op,
			Arg: arg,
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
				Label: fmt.Sprintf("block_%v", len(program.Blocks)),
				Offset: i + size + 1,
				Reads: 0,
			}
			currentBlock = newBlock
			currentStackIndex = 0
		}
		
		// Skip operand bytes
		i += op.OperandSize()
	}
	
	if len(currentBlock.Instructions) > 0 {
		program.Blocks = append(program.Blocks, currentBlock)
	}
	
	return program
}

func (program *Program) PrintAssembler() {
	for _, block := range program.Blocks {
		offset := block.Offset
		
		// Label the block
		fmt.Printf("%v: (reads %v, writes %v)\n", block.Label,
			block.Reads, block.Writes)
		for _, instruction := range block.Instructions {
			fmt.Printf("0x%X\t%v", offset, instruction.Op)
			if instruction.Arg != nil {
				fmt.Printf("\t 0x%X", instruction.Arg)
			}
			fmt.Printf("\n")
			offset += instruction.Op.OperandSize() + 1
		}
		fmt.Printf("\n")
	}
}

func (program *Program) ParseCreation() {
	// The program is contract creation code. The entry point is 0x0 and will
	// at some point use CODECOPY calls to set up the contract at the right
	// location.
	// For now we assume the basic behaviour of `solc`. The setup code is in
	// one basic block (MSTORE, CODECOPY, RETURN) and the second basic block
	// is the entry point for the Contract ABI.
	program.Blocks[0].Label = "create"
	program.Blocks[1].Label = "enter"
	
	// Adjust the offsets the blocks.
	enterOffset := program.Blocks[1].Offset
	for i := 1; i < len(program.Blocks); i++ {
		program.Blocks[i].Offset -= enterOffset
	}
}
