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

type opCodeConvention int
const (
	NULLARY opCodeConvention = iota
	UNARY
	BINARY
	FUNCTION
	FIELD
	MEMBER
)

type opCodeInfoRecord struct {
	Convention    opCodeConvention
	Solidity      string
}

var opCodeInfo = map[OpCode]opCodeInfoRecord{
	ADD:          {BINARY,   "+"},
	MUL:          {BINARY,   "*"},
	SUB:          {BINARY,   "-"},
	DIV:          {BINARY,   "/"},
	SDIV:         {BINARY,   "/"}, // signed
	MOD:          {BINARY,   "%"},
	SMOD:         {BINARY,   "%"}, // signed
	ADDMOD:       {FUNCTION, "addmod"},
	MULMOD:       {FUNCTION, "mulmod"},
	EXP:          {BINARY,   "**"},
	NOT:          {UNARY,    "!"},
	LT:           {BINARY,   "<"},
	GT:           {BINARY,   ">"},
	SLT:          {BINARY,   "<"}, // signed
	SGT:          {BINARY,   ">"}, // signed
	EQ:           {BINARY,   "=="},
	ISZERO:       {BINARY,   "0 =="},
	AND:          {BINARY,   "&"},
	OR:           {BINARY,   "|"},
	XOR:          {BINARY,   "^"},
	SHA3:         {FUNCTION, "sha3"},
	ADDRESS:      {NULLARY,  "this"},
	BALANCE:      {FIELD,    "balance"},
	ORIGIN:       {NULLARY,  "tx.origin"},
	CALLER:       {NULLARY,  "msg.sender"},
	CALLVALUE:    {NULLARY,  "msg.value"},
	CALLDATALOAD: {FUNCTION, "CALLDATALOAD"},
	CALLDATASIZE: {FUNCTION, "CALLDATASIZE"},
	CALLDATACOPY: {FUNCTION, "CALLDATACOPY"},
	CODESIZE:     {FUNCTION, "CODESIZE"},
	CODECOPY:     {FUNCTION, "CODECOPY"},
	GASPRICE:     {NULLARY,  "tx.gasprice"},
	BLOCKHASH:    {FUNCTION, "block.blockhash"},
	COINBASE:     {NULLARY,  "block.coinbase"},
	TIMESTAMP:    {NULLARY,  "block.timestamp"},
	NUMBER:       {NULLARY,  "block.number"},
	DIFFICULTY:   {NULLARY,  "block.difficulty"},
	GASLIMIT:     {NULLARY,  "block.gaslimit"},
	EXTCODESIZE:  {FUNCTION, "EXTCODESIZE"},
	EXTCODECOPY:  {FUNCTION, "EXTCODECOPY"},
	MLOAD:        {FUNCTION, "MLOAD"},
	MSTORE:       {FUNCTION, "MSTORE"},
	MSTORE8:      {FUNCTION, "MSTORE8"},
	SLOAD:        {FUNCTION, "SLOAD"},
	SSTORE:       {FUNCTION, "SSTORE"},
	PC:           {FUNCTION, "PC"},
	MSIZE:        {FUNCTION, "MSIZE"},
	GAS:          {NULLARY,  "msg.gas"},
	LOG0:         {FUNCTION, "LOG0"},
	LOG1:         {FUNCTION, "LOG1"},
	LOG2:         {FUNCTION, "LOG2"},
	LOG3:         {FUNCTION, "LOG3"},
	LOG4:         {FUNCTION, "LOG4"},
	CREATE:       {FUNCTION, "CREATE"},
	CALL:         {FUNCTION, "CALL"},
	RETURN:       {FUNCTION, "RETURN"},
	CALLCODE:     {FUNCTION, "CALLCODE"},
	DELEGATECALL: {FUNCTION, "DELEGATECALL"},
	SELFDESTRUCT: {FUNCTION, "selfdestruct"},
}

func (statement Statement) Convention() opCodeConvention {
	return opCodeInfo[statement.Op].Convention
}

func (statement Statement) Solidity() string {
	return opCodeInfo[statement.Op].Solidity
}

func (statement *Statement) Replace(from Expression, to Expression) {
	oldInputs := statement.Inputs
	statement.Inputs = make([]Expression, 0)
	for _, input := range oldInputs {
		if input == from {
			input = to
		}
		statement.Inputs = append(statement.Inputs, input)
	}
	if statement.Output != nil && statement.Output == from {
		statement.Output = &Variable{
			Label: to.(Variable).Label,
		}
	}
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
		str += fmt.Sprintf("var %v = ", statement.Output)
	}
	
	switch statement.Convention() {
	case NULLARY:
		str += fmt.Sprintf("%v", statement.Solidity())
	case UNARY:
		str += fmt.Sprintf("%v %v", statement.Solidity(), statement.Inputs[0])
	case BINARY:
		str += fmt.Sprintf("%v %v %v", statement.Inputs[0],
			statement.Solidity(), statement.Inputs[1])
	case FIELD:
		str += fmt.Sprintf("%v.%v", statement.Inputs[0], statement.Solidity())
	case MEMBER, FUNCTION:
		start := 0
		if statement.Convention() == MEMBER {
			str += fmt.Sprintf("%v.", statement.Inputs[0])
			start = 1
		}
		str += fmt.Sprintf("%v(", statement.Solidity())
		for i := start; i < len(statement.Inputs); i++ {
			if i > start {
				str += ", "
			}
			str += fmt.Sprintf("%v", statement.Inputs[i])
		}
		str += ")"
	}
	str += ";"
	return str
}

type StatementBlock struct {
	Offset          int
	Statements      []*Statement
	Label           string
	Inputs          []Expression
	Outputs         []Expression
	Incoming        []*StatementBlock
	CondBlocks      []*StatementBlock
	NextBlock       *StatementBlock
}

func (block StatementBlock) String() string {
	condCounter := 0
	
	// Block header
	str := fmt.Sprintf("0x%X %v: %v → %v\n",
		block.Offset, block.Label, block.Inputs, block.Outputs)
	
	// Origins
	for _, source := range block.Incoming {
		str += fmt.Sprintf("\tfrom %v\n", source.Label)
	}
	
	// Statements
	for _, statement := range block.Statements {
		switch statement.Op {
		case JUMPDEST:
		case JUMP:
			if block.NextBlock != nil {
				str += fmt.Sprintf("\tJUMP(%v)\n", block.NextBlock.Label)
			} else {
				str += fmt.Sprintf("\t%v\n", statement)
			}
		case JUMPI:
			str += fmt.Sprintf("\tJUMPI %v %v\n", statement.Inputs[1],
				block.CondBlocks[condCounter].Label)
			condCounter++
		default:
			str += fmt.Sprintf("\t%v\n", statement)
		}
	}
	
	// Targets
	if block.NextBlock != nil {
		str += fmt.Sprintf("\tto %v\n", block.NextBlock.Label)
	}
	
	return str
}

func (block *StatementBlock) Replace(from Expression, to Expression) {
	for _, statement := range block.Statements {
		statement.Replace(from, to)
	}
	newOutputs := make([]Expression, 0)
	for _, output := range block.Outputs {
		if output == from {
			output = to
		}
		newOutputs = append(newOutputs, output)
	}
	block.Outputs = newOutputs
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

type SSAProgram struct {
	Blocks          []*StatementBlock
}

func (ssa SSAProgram) PrintSSA() {
	for _, block := range ssa.Blocks {
		fmt.Printf("%v\n", block)
	}
}

// A counter for generating unique identifiers
var ssaCount int

func CompileSSABlock(block *BasicBlock) *StatementBlock {
	
	// Create the StatementBlock
	statements := &StatementBlock{
		Offset:     block.Offset,
		Label:      block.Label,
		Statements: make([]*Statement, 0),
		Inputs:     make([]Expression, 0),
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
		ssaCount++
		variable := Variable{
			Label: fmt.Sprintf("a%v", ssaCount),
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
	statements.Outputs = stack.Values
	
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
		Inputs:     make([]Expression, 0),
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

func Prefix(list *[]Expression, prefix []Expression) {
	newList := make([]Expression, 0)
	for _, expression := range prefix {
		newList = append(newList, expression)
	}
	for _, expression := range *list {
		newList = append(newList, expression)
	}
	*list = newList
}

func (ssa *SSAProgram) MergeBlocks(first *StatementBlock, second *StatementBlock) {
	
	// Connect outputs to inputs
	out := len(first.Outputs)
	in := len(second.Inputs)
	if out > in {
		// The extra outputs are prepended to second's inputs and outputs
		// making them pass through the second block without being touched.
		extra := first.Outputs[:out - in]
		Prefix(&second.Inputs, extra)
		Prefix(&second.Outputs, extra)
		in = out
	}
	if in > out {
		// The extra inputs are prepended to the first's inputs and outputs
		// making them pass through the first block without being touched.
		extra := second.Inputs[:in - out]
		Prefix(&first.Inputs, extra)
		Prefix(&first.Outputs, extra)
		out = in
	}
	
	// Replace all occurences of seconds inputs with the firsts outputs
	for i := 0; i < in; i++ {
		second.Replace(second.Inputs[i], first.Outputs[i])
	}
	
	// Seconds outputs are the merged blocks outputs
	first.Outputs = second.Outputs
	
	// TODO: Remove uncoditional JUMP statement?
	
	// Append statements from second block to first
	for _, statement := range second.Statements {
		if statement.Op == JUMPDEST {
			continue
		}
		first.Statements = append(first.Statements, statement)
	}
	
	// Remove the second block
	newBlocks := make([]*StatementBlock, 0)
	for _, block := range ssa.Blocks {
		if block != second {
			newBlocks = append(newBlocks, block)
		}
	}
	ssa.Blocks = newBlocks
	
	// Rewire the connections and re-analyse the control flow
	first.NextBlock = second.NextBlock
	ssa.UpdateJumpTargets(first)
	ssa.ComputeIncoming()
}

func (ssa *SSAProgram) TryCollapseOneJump() bool {
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
		
		// If the last statement of source is a JUMP, we can drop it
		n := len(source.Statements)
		if  n > 0  && source.Statements[n - 1].Op == JUMP {
			source.Statements = source.Statements[:n - 1]
		}
		
		// Okay, merge the blocks!
		ssa.MergeBlocks(source, block)
		
		// Return because the itterator may be invalid now
		return true
	}
	return false
}

func (ssa *SSAProgram) CollapseJumps() {
	for ssa.TryCollapseOneJump() { }
}
