package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	".."
)

func main() {
	hexdata, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
	    log.Fatalf("Could not read from stdin: %v", err)
	}
	
	fmt.Printf("# hex.Decode\n");
	bytecode := make([]byte, hex.DecodedLen(len(hexdata)))
	hex.Decode(bytecode, hexdata)
	
	fmt.Printf("# NewProgram\n");
	program := evmdis.NewProgram(bytecode)
	program.ParseCreation()
	// program.PrintAssembler()
	
	fmt.Printf("# StackLabel\n");
	ssa := evmdis.CompileSSA(program)
	ssa.ComputeJumpTargets()
	ssa.ComputeIncoming()
	ssa.CollapseJumps()
	for _, block := range ssa.Blocks {
		
		fmt.Printf("0x%X %v: %v → %v\n", block.Offset, block.Label, block.Inputs,
			block.Outputs)
		for _, source := range block.Incoming {
			fmt.Printf("\tfrom %v\n", source.Label)
		}
		condCounter := 0
		for _, statement := range block.Statements {
			switch statement.Op {
			case evmdis.JUMPDEST:
			case evmdis.JUMP:
				if block.NextBlock != nil {
					fmt.Printf("\tJUMP(%v)\n", block.NextBlock.Label)
				} else {
					fmt.Printf("\t%v\n", statement)
				}
			case evmdis.JUMPI:
				fmt.Printf("\tJUMPI %v %v\n", statement.Inputs[1],
					block.CondBlocks[condCounter].Label)
				condCounter++
			default:
				fmt.Printf("\t%v\n", statement)
			}
		}
		fmt.Printf("\n")
	}
}
