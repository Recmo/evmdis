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
	for _, block := range ssa.Blocks {
		
		fmt.Printf("%v: %v → %v\n", block.Label, block.Inputs, block.Outputs)
		for _, statement := range block.Statements {
			fmt.Printf("\t%v\n", statement)
		}
		fmt.Printf("\n")
	}
}
