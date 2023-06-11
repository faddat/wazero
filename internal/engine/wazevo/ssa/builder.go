// Package ssa is used to construct SSA function. By nature this is free of Wasm specific thing
// and ISA.
package ssa

import (
	"fmt"
)

// Builder is used to builds SSA consisting of Basic Blocks per function.
type Builder interface {
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

	// DefineVariableInCurrentBB is the same as DefineVariable except the definition is
	// inserted into the current BasicBlock. Alias to DefineVariable(x, y, CurrentBlock()).
	DefineVariableInCurrentBB(variable Variable, value Value)

	// AllocateInstruction returns a new Instruction.
	AllocateInstruction() *Instruction

	// InsertInstruction executes BasicBlock.InsertInstruction for the currently handled basic block.
	InsertInstruction(raw *Instruction)

	// allocateValue allocates an unused Value.
	allocateValue(typ Type) Value

	// FindValue searches the latest definition of the given Variable and returns the result.
	FindValue(variable Variable) Value

	// Seal declares that we've known all the predecessors to this block and were added via AddPred.
	// After calling this, AddPred will be forbidden.
	Seal(blk BasicBlock)

	// AnnotateValue is for debugging purpose.
	AnnotateValue(value Value, annotation string)
}

// NewBuilder returns a new Builder implementation.
func NewBuilder() Builder {
	return &builder{
		instructionsPool: newPool[Instruction](),
		basicBlocksPool:  newPool[basicBlock](),
		valueAnnotations: make(map[valueID]string),
	}
}

// builder implements Builder interface.
type builder struct {
	basicBlocksPool  pool[basicBlock]
	instructionsPool pool[Instruction]

	basicBlocksView []BasicBlock
	currentBB       *basicBlock

	// variables track the types for Variable with the index regarded Variable.
	variables []Type
	// nextValueID is used by builder.AllocateValue.
	nextValueID valueID
	// nextVariable is used by builder.AllocateVariable.
	nextVariable Variable

	valueAnnotations map[valueID]string
}

// Reset implements Builder.Reset.
func (b *builder) Reset() {
	b.instructionsPool.reset()

	for i := 0; i < b.basicBlocksPool.allocated; i++ {
		b.basicBlocksPool.view(i).reset()
	}
	b.basicBlocksPool.reset()

	for i := Variable(0); i < b.nextVariable; i++ {
		b.variables[i] = TypeInvalid
	}

	for v := valueID(0); v < b.nextValueID; v++ {
		delete(b.valueAnnotations, v)
	}
	b.nextValueID = valueIDInvalid + 1
}

// AnnotateValue implements Builder.AnnotateValue.
func (b *builder) AnnotateValue(value Value, a string) {
	b.valueAnnotations[value.id()] = a
}

// AllocateInstruction implements Builder.AllocateInstruction.
func (b *builder) AllocateInstruction() *Instruction {
	return b.instructionsPool.allocate()
}

// AllocateBasicBlock implements Builder.AllocateBasicBlock.
func (b *builder) AllocateBasicBlock() BasicBlock {
	id := basicBlockID(b.basicBlocksPool.allocated)
	blk := b.basicBlocksPool.allocate()
	blk.id = id
	blk.lastDefinitions = make(map[Variable]Value)
	blk.unknownValues = make(map[Variable]Value)
	return blk
}

// InsertInstruction implements Builder.InsertInstruction.
func (b *builder) InsertInstruction(instr *Instruction) {
	b.currentBB.InsertInstruction(instr)

	resultTypesFn := instructionReturnTypes[instr.opcode]
	if resultTypesFn == nil {
		panic("TODO: " + instr.Format(b))
	}

	t1, ts := resultTypesFn(b, instr)
	if t1.invalid() {
		return
	}

	r1 := b.allocateValue(t1)
	instr.rValue = r1

	tsl := len(ts)
	if tsl == 0 {
		return
	}

	// TODO: reuse slices, though this seems not to be common.
	instr.rValues = make([]Value, tsl)
	for i := 0; i < tsl; i++ {
		instr.rValues[i] = b.allocateValue(ts[i])
	}
}

// Blocks implements Builder.Blocks.
func (b *builder) Blocks() []BasicBlock {
	b.basicBlocksView = b.basicBlocksView[:0]
	for i := 0; i < b.basicBlocksPool.allocated; i++ {
		blk := b.basicBlocksPool.view(i)
		if blk.ReturnBlock() {
			continue
		}
		b.basicBlocksView = append(b.basicBlocksView, blk)
	}
	return b.basicBlocksView
}

// DefineVariable implements Builder.DefineVariable.
func (b *builder) DefineVariable(variable Variable, value Value, block BasicBlock) {
	if b.variables[variable] == TypeInvalid {
		panic("BUG: trying to define variable " + variable.String() + " but is not declared yet")
	}

	bb := block.(*basicBlock)
	bb.lastDefinitions[variable] = value
}

// DefineVariableInCurrentBB implements Builder.DefineVariableInCurrentBB.
func (b *builder) DefineVariableInCurrentBB(variable Variable, value Value) {
	b.DefineVariable(variable, value, b.currentBB)
}

// SetCurrentBlock implements Builder.SetCurrentBlock.
func (b *builder) SetCurrentBlock(bb BasicBlock) {
	b.currentBB = bb.(*basicBlock)
}

// CurrentBlock implements Builder.CurrentBlock.
func (b *builder) CurrentBlock() BasicBlock {
	return b.currentBB
}

// DeclareVariable implements Builder.DeclareVariable.
func (b *builder) DeclareVariable(typ Type) Variable {
	v := b.allocateVariable()
	iv := int(v)
	if l := len(b.variables); l <= iv {
		b.variables = append(b.variables, make([]Type, 2*(l+1))...)
	}
	b.variables[v] = typ
	return v
}

func (b *builder) allocateVariable() (ret Variable) {
	ret = b.nextVariable
	b.nextVariable++
	return
}

// allocateValue implements Builder.AllocateValue.
func (b *builder) allocateValue(typ Type) (v Value) {
	v = Value(b.nextValueID)
	v.setType(typ)
	b.nextValueID++
	return
}

// FindValue implements Builder.FindValue.
func (b *builder) FindValue(variable Variable) Value {
	typ := b.definedVariableType(variable)
	return b.findValue(typ, variable, b.currentBB)
}

// findValue recursively tries to find the latest definition of a `variable`. The algorithm is described in
// the section 2 of the paper https://link.springer.com/content/pdf/10.1007/978-3-642-37051-9_6.pdf.
//
// TODO: reimplement this in iterative, not recursive, to avoid stack overflow.
func (b *builder) findValue(typ Type, variable Variable, blk *basicBlock) Value {
	if val, ok := blk.lastDefinitions[variable]; ok {
		// The value is already defined in this block!
		return val
	} else if !blk.sealed { // Incomplete CFG as in the paper.
		// If this is not sealed, that means it might have additional unknown predecessor later on.
		// So we temporarily define the placeholder value here (not add as a parameter yet!),
		// and record it as unknown.
		// The unknown values are resolved when we call seal this block via BasicBlock.Seal().
		value := b.allocateValue(typ)
		blk.lastDefinitions[variable] = value
		blk.unknownValues[variable] = value
		return value
	}

	if pred := blk.singlePred; pred != nil {
		// If this block is sealed and have only one predecessor,
		// we can use the value in that block without ambiguity on definition.
		return b.findValue(typ, variable, pred)
	}

	// If this block has multiple predecessors, we have to gather the definitions,
	// and treat them as an argument to this block. So the first thing we do now is
	// define a new parameter to this block which may or may not be redundant, but
	// later we eliminate trivial params in an optimization pass.
	paramValue := b.allocateValue(typ)
	blk.addParamOn(b, variable, paramValue)
	// After the new "phi" param is added, we have to manipulate the original branching instructions
	// in predecessors so that they would pass the definition of `variable` as the argument to
	// the newly added phi.
	for i := range blk.preds {
		pred := &blk.preds[i]
		// Find the definition in the predecessor recursively.
		value := b.findValue(typ, variable, pred.blk)
		pred.branch.addArgument(value)
	}
	return paramValue
}

// Seal implements Builder.Seal.
func (b *builder) Seal(raw BasicBlock) {
	blk := raw.(*basicBlock)
	if len(blk.preds) == 1 {
		blk.singlePred = blk.preds[0].blk
	}
	blk.sealed = true

	for variable, phiValue := range blk.unknownValues {
		typ := b.definedVariableType(variable)
		blk.addParamOn(b, variable, phiValue)
		for i := range blk.preds {
			pred := &blk.preds[i]
			predValue := b.findValue(typ, variable, pred.blk)
			pred.branch.addArgument(predValue)
		}
	}
}

func (b *builder) definedVariableType(variable Variable) Type {
	typ := b.variables[variable]
	if typ == TypeInvalid {
		panic(fmt.Sprintf("%s is not defined yet", variable))
	}
	return typ
}
