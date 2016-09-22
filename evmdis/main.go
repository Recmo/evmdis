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
	for _, block := range program.Blocks {
		offset := block.Offset
		
		// Print label
		var label *evmdis.JumpLabel
		block.Annotations.Get(&label)
		if label != nil {
			fmt.Printf("%v\n", label)
		} else {
			fmt.Print("block:\n")
		}
		for _, instruction := range block.Instructions {
			fmt.Printf("0x%X\t%v", offset, instruction.Op)
			if instruction.Arg != nil {
				fmt.Printf("\t 0x%X", instruction.Arg)
			}
			fmt.Printf("\n")			
			offset += instruction.Op.OperandSize() + 1
		}
		fmt.Printf("\n")
		
		// Update offset
	}
	
	return
	
	fmt.Printf("# PerformReachingAnalysis\n");
	if err := evmdis.PerformReachingAnalysis(program); err != nil {
		log.Fatalf("Error performing reaching analysis: %v", err)
	}
	fmt.Printf("# PerformReachesAnalysis\n");
	evmdis.PerformReachesAnalysis(program)
	fmt.Printf("# CreateLabels\n");
	evmdis.CreateLabels(program)
	fmt.Printf("# BuildExpressions\n");
	evmdis.BuildExpressions(program)

	for _, block := range program.Blocks {
		offset := block.Offset

		var label *evmdis.JumpLabel
		block.Annotations.Get(&label)
		if label != nil {
			fmt.Printf("%v\n", label)
		}

		for _, instruction := range block.Instructions {
			var reaching evmdis.ReachingDefinition
			instruction.Annotations.Get(&reaching)

			var reaches evmdis.ReachesDefinition
			instruction.Annotations.Get(&reaches)

			var expression evmdis.Expression
			instruction.Annotations.Get(&expression)

			if expression != nil {
				fmt.Printf("0x%X\t%v\t%v\t%v\n", offset, expression, reaching, reaches)
				if instruction.Op.StackWrites() == 1 && !instruction.Op.IsDup() {
					fmt.Printf("0x%X\tPUSH(%v)\n", offset, expression)
				} else {
					fmt.Printf("0x%X\t%v\n", offset, expression)
				}
			}
			offset += instruction.Op.OperandSize() + 1
		}
		fmt.Printf("\n")
	}
}
