// Package ssa is used to construct SSA function. By nature this is free of Wasm specific thing
// and ISA.
package ssa

import (
	"fmt"
	"strings"
)

type

// Builder is used to builds SSA consisting of Basic Blocks per function.
Builder interface {
	fmt.Stringer

	// Reset must be called to reuse this builder for the next function.
	Reset()

	// AllocateBasicBlock creates a basic block in SSA function.
	AllocateBasicBlock() BasicBlock

	// Blocks return the valid BasicBlock(s).
	Blocks() []BasicBlock

	// CurrentBlock returns the currently handled BasicBlock which is set by the latest call to SetCurrentBlock.
	CurrentBlock() BasicBlock

	// SetCurrentBlock sets the instruction insertion target to the BasicBlock `b`.
	SetCurrentBlock(b BasicBlock)

	// DeclareVariable declares a Variable of the given Type.
	DeclareVariable(Type) Variable

	// DefineVariable defines a variable in the `block` with value.
	// The defining instruction will be inserted into the `block`.
	DefineVariable(variable Variable, value Value, block BasicBlock)

	// AllocateInstruction returns a new Instruction.
	AllocateInstruction() *Instruction

	// InsertInstruction executes BasicBlock.InsertInstruction for the currently handled basic block.
	InsertInstruction(raw *Instruction)

	// AllocateValue allocates an unused Value.
	AllocateValue() Value

	// FindValue searches the latest definition of the given Variable and returns the result.
	FindValue(variable Variable) Value
}

// NewBuilder returns a new Builder implementation.
func NewBuilder() Builder {
	return &builder{
		instructionsPool: newInstructionsPool(),
		basicBlocksPool:  newBasicBlocksPool(),
	}
}

// builder implements Builder interface.
//
// We use the algorithm described in the paper:
// "Simple and Efficient Construction of Static Single Assignment Form" https://link.springer.com/content/pdf/10.1007/978-3-642-37051-9_6.pdf
//
// with the stricter assumption that our input is always a "complete" CFG.
type builder struct {
	nextVariable     Variable
	basicBlocksPool  basicBlocksPool
	instructionsPool instructionsPool

	basicBlocksView []BasicBlock
	currentBB       *basicBlock

	// variables track the types for Variable with the index regarded Variable.
	variables []Type

	nextValue Value
}

// Reset implements Builder.
func (b *builder) Reset() {
	b.instructionsPool.reset()

	for i := 0; i < b.basicBlocksPool.allocated; i++ {
		b.basicBlocksPool.view(i).reset()
	}
	b.basicBlocksPool.reset()

	for i := Variable(0); i < b.nextVariable; i++ {
		b.variables[i] = TypeInvalid
	}

	b.nextValue = valueInvalid + 1
}

// AllocateInstruction implements Builder.
func (b *builder) AllocateInstruction() *Instruction {
	return b.instructionsPool.allocate()
}

// AllocateBasicBlock implements Builder.
func (b *builder) AllocateBasicBlock() BasicBlock {
	id := b.basicBlocksPool.allocated
	blk := b.basicBlocksPool.allocate()
	blk.id = id
	return blk
}

// InsertInstruction implements Builder.
func (b *builder) InsertInstruction(instr *Instruction) {
	b.currentBB.InsertInstruction(instr)
	num, unknown := instr.opcode.numReturns()
	if unknown {
		panic("TODO: unknown returns")
	}

	if num == 0 {
		return
	}

	r1 := b.AllocateValue()
	instr.rValue = r1
	num--

	if num == 0 {
		return
	}

	// TODO: reuse slices, though this seems not to be common.
	instr.rValues = make([]Value, num)
	for i := 0; i < num; i++ {
		instr.rValues[i] = b.AllocateValue()
	}
}

// Blocks implements Builder.
func (b *builder) Blocks() []BasicBlock {
	blkNum := b.basicBlocksPool.allocated
	if blkNum >= len(b.basicBlocksView) {
		b.basicBlocksView = append(b.basicBlocksView, make([]BasicBlock, blkNum)...)
	}
	for i := 0; i < blkNum; i++ {
		b.basicBlocksView[i] = b.basicBlocksPool.view(i)
	}
	return b.basicBlocksView[:blkNum]
}

// DefineVariable implements Builder.
func (b *builder) DefineVariable(variable Variable, value Value, block BasicBlock) {
	if b.variables[variable] == TypeInvalid {
		panic("BUG: trying to define variable " + variable.String() + " but is not declared yet")
	}

	bb := block.(*basicBlock)
	if bb.lastDefinitions == nil {
		bb.lastDefinitions = make(map[Variable]Value, 1)
	}
	bb.lastDefinitions[variable] = value
}

// SetCurrentBlock implements Builder.
func (b *builder) SetCurrentBlock(bb BasicBlock) {
	b.currentBB = bb.(*basicBlock)
}

// CurrentBlock implements Builder.
func (b *builder) CurrentBlock() BasicBlock {
	return b.currentBB
}

// DeclareVariable implements Builder.
func (b *builder) DeclareVariable(typ Type) Variable {
	v := b.AllocateVariable()
	iv := int(v)
	if l := len(b.variables); l <= iv {
		b.variables = append(b.variables, make([]Type, 2*(l+1))...)
	}
	b.variables[v] = typ
	return v
}

// AllocateVariable implements Builder.
func (b *builder) AllocateVariable() (ret Variable) {
	ret = b.nextVariable
	b.nextVariable++
	return
}

// String implements fmt.Stringer.
func (b *builder) String() string {
	str := strings.Builder{}
	for _, blk := range b.Blocks() {
		header := blk.String()
		str.WriteByte('\n')
		str.WriteString(header)
		str.WriteByte('\n')
		for cur := blk.Root(); cur != nil; cur = cur.Next() {
			str.WriteByte('\t')
			str.WriteString(cur.String())
			str.WriteByte('\n')
		}
	}
	return str.String()
}

// AllocateValue implements Builder.
func (b *builder) AllocateValue() (v Value) {
	v = b.nextValue
	b.nextValue++
	return
}

// FindValue implements Builder.
func (b *builder) FindValue(variable Variable) Value {
	currentDefs := b.currentBB.lastDefinitions
	if currentDefs != nil {
		if val, ok := currentDefs[variable]; ok {
			return val
		}
	}

	panic("TODO")
}
