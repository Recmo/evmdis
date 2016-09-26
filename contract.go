package evmdis

import (
	"fmt"
)

func (ssa *SSAProgram) LabelFunctions() {
	
	// The 'entry' block looks like this:
	//
	// [… 3 instructions …]
	//  xn = EQ(0x29E99F07, x2)
    //  JUMPI xn block_m
	// [… above pattern repeated for every public function …]
	// [… 2 instructions …]
	
	// TODO: Brute force the ABI hash
	
	entry := ssa.Blocks[1]
	for i := 3; i < len(entry.Statements) - 2; i += 2 {
		hash   := entry.Statements[i + 0].Inputs[0].(Constant).Value
		offset := entry.Statements[i + 1].Inputs[0].(Constant).Value
		
		for _, block := range ssa.Blocks {
			if block.Offset == int(offset.Int64()) {
				block.Label = fmt.Sprintf("func_%x", hash)
				ssa.Unboilerplate(block)
			}
		}
	}
}

func (ssa *SSAProgram) Unboilerplate(block *StatementBlock) {
	
	// A function has boilerplate:
	//
	// JUMPDEST()
	// x14 = CALLVALUE()
	// JUMPI(0x2, x14)
	// x15 = CALLDATALOAD(0x4)
	// x16 = ADD(0x20, 0x4)
	// x17 = CALLDATALOAD(x16)  // Repeated for every
	// x18 = ADD(0x20, x16)     // input argument
	// [… body …]
	// x21 = MLOAD(0x40)
	// MSTORE(x21, x32)
	// x22 = ADD(0x20, x21)     // Repeated for every
	// MSTORE(x22, x33)         // return value
	// x23 = ADD(0x20, x22)
	// x24 = MLOAD(0x40)
	// x25 = SUB(x23, x24)
	// RETURN(x24, x25)
	
	// Turn header into block inputs
	headerLength := 3
	for ; block.Statements[headerLength].Op == CALLDATALOAD; headerLength += 2 {
		arg := block.Statements[headerLength].Output
		block.Inputs = append(block.Inputs, arg)
	}
	block.Statements = block.Statements[headerLength:]
	
	// Turn trailer into block outputs
	n := len(block.Statements)
	trailerLength := 3
	for ; block.Statements[n - trailerLength - 1].Op == ADD; trailerLength += 2 {
		arg := block.Statements[n - trailerLength - 2].Inputs[1]
		block.Outputs = append(block.Outputs, arg)
	}
	trailerLength += 1
    for i, j := 0, len(block.Outputs)-1; i < j; i, j = i+1, j-1 {
		block.Outputs[i], block.Outputs[j] = block.Outputs[j], block.Outputs[i]
	}
	block.Statements = block.Statements[:n - trailerLength]
}

func (ssa *SSAProgram) Function(block *StatementBlock) string {
	
	// Write the function declaration
	str := fmt.Sprintf("\tfunction %v(", block.Label)
	for i := 0; i < len(block.Inputs); i++ {
		if i > 0 {
			str += ", "
		}
		str += fmt.Sprintf("uint %v", block.Inputs[i])
	}
	str += ") "
	if len(block.Outputs) > 0 {
		str += "return ("
		for i := 0; i < len(block.Inputs); i++ {
			if i > 0 {
				str += ", "
			}
			str += "uint"
		}
		str += ") "
	}
	str += "{\n"
	
	// Write the function body
	for _, statement := range block.Statements {
		str += fmt.Sprintf("\t\t%v\n", statement)
	}
	
	// Write the return statement
	if len(block.Outputs) > 0 {
		str += "\t\treturn("
		for i := 0; i < len(block.Outputs); i++ {
			if i > 0 {
				str += ", "
			}
			str += fmt.Sprintf("%v", block.Outputs[i])
		}
		str += ");\n"
	}
	
	str += "\t}\n"
	return str
}

func (ssa *SSAProgram) Contract() string {
	str := "pragma solidity ^0.4.2;\n\ncontract Decompiled {\n"
	for _, block := range ssa.Blocks {
		if block.Label[:5] == "func_" {
			str += ssa.Function(block)
		}
	}
	str+= "}\n"
	return str
}
