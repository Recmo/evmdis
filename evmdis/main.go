package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"
	"log"
	"os"
	".."
)

func main() {
	hexdata, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
	    log.Fatalf("Could not read from stdin: %v", err)
	}
	
	printSSA := true
	
	fmt.Printf("# hex.Decode\n");
	bytecode := make([]byte, hex.DecodedLen(len(hexdata)))
	hex.Decode(bytecode, hexdata)
	
	fmt.Printf("# NewProgram\n");
	program := evmdis.NewProgram(bytecode)
	program.ParseCreation()
	// program.PrintAssembler()
	
	fmt.Printf("# StackLabel\n");
	if printSSA {
		for _, block := range program.Blocks {
			offset := 0
			nextOffset := block.Offset
			stack := evmdis.CreateStack(block.Reads)
			ssaCount := 0
			
			// Label the block
			if block.Reads > 0 {
				fmt.Printf("%v(%v):\n", block.Label, stack)
			} else {
				fmt.Printf("%v:\n", block.Label)
			}
			for _, instruction := range block.Instructions {
				
				// Update offsets
				offset = nextOffset
				nextOffset = offset + instruction.Op.OperandSize() + 1
				
				// Stack management
				if instruction.Op.IsPush() {
					value := fmt.Sprintf("0x%X", instruction.Arg)
					stack.Push(value)
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
				if instruction.Op == evmdis.POP {
					stack.Pop()
					continue
				}
				arguments := make([]string, 0)
				for i := 0; i < instruction.Op.StackReads(); i++ {
					arguments = append(arguments, stack.Pop())
				}
				results := make([]string, 0)
				for i := 0; i < instruction.Op.StackWrites(); i++ {
					ssaCount++
					variable := fmt.Sprintf("x%v", ssaCount)
					stack.Push(variable)
					results = append(results, variable)
				}
				
				// Print offset
				fmt.Printf("0x%X\t", offset)
				
				// Print result
				if len(results) > 0 {
					fmt.Printf("%v = ", strings.Join(results, ", "))
				}
				
				// Print opcode
				fmt.Printf("%v(%v)\n", instruction.Op, strings.Join(arguments, ", "))
			}
			if stack.Size() > 0 {
				fmt.Printf("\tstack = %v\n", stack)
			}
			fmt.Printf("\n")
		}
	}
}
