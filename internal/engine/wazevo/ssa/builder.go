// Package ssa is used to construct SSA function. By nature this is free of Wasm specific thing
// and ISA.
package ssa

import (
	"fmt"
	"strconv"
	"strings"
)

type (

	// Builder is used to builds SSA consisting of Basic Blocks per function.
	Builder interface {
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
		InsertInstruction(raw Value)
	}

	// BasicBlock represents the Basic Block of an SSA function.
	// In traditional SSA terminology, the block "params" here are called phi values,
	// and there does not exist "params". However, for simplicity, we handle them as parameters to a BB.
	BasicBlock interface {
		fmt.Stringer

		// AddParam adds the parameter to the block whose type specified by `t`.
		AddParam(b Builder, t Type) Variable

		// Params returns the number of parameters to this block.
		Params() int

		// Param returns (Variable, Value) which corresponds to the i-th parameter of this block.
		// The returned Variable can be used to add the definition of it in predecessors,
		// and the returned Value is the phi definition of a variable in this block.
		Param(i int) (Variable, Value)

		// InsertInstruction inserts an instruction that implements Value into the tail of this block.
		InsertInstruction(raw Value)

		// AddPred appends `block` as a predecessor to this BB.
		AddPred(block BasicBlock)

		// Root returns the root instruction of this block.
		Root() *Instruction
	}
)

// NewBuilder returns a new Builder implementation.
func NewBuilder() Builder {
	return &builder{
		instructionsPool: instructionsPool{index: instructionsPoolPageSize},
	}
}

// builder implements Builder interface.
//
// We use the algorithm described in the paper:
// "Simple and Efficient Construction of Static Single Assignment Form" https://link.springer.com/content/pdf/10.1007/978-3-642-37051-9_6.pdf
//
// with the stricter assumption that our input is always a "complete" CFG.
type builder struct {
	nextBasicBlock  int
	nextVariable    Variable
	basicBlocks     []basicBlock
	basicBlocksView []BasicBlock
	currentBB       *basicBlock

	// variables track the types for Variable with the index regarded Variable.
	variables []Type

	// lastDefinitions track last definitions of a variable in each block.
	lastDefinitions          []map[Variable]Value
	lastDefinitionsResetTemp []Variable
	instructions             []Instruction

	instructionsPool instructionsPool
}

// Reset implements Builder.
func (b *builder) Reset() {
	b.instructionsPool.reset()

	for i := 0; i < b.nextBasicBlock; i++ {
		b.basicBlocks[i].reset()
	}

	for i := Variable(0); i < b.nextVariable; i++ {
		b.variables[i] = TypeInvalid
	}

	for _, defs := range b.lastDefinitions {
		b.lastDefinitionsResetTemp = b.lastDefinitionsResetTemp[:0]
		for key := range defs {
			b.lastDefinitionsResetTemp = append(b.lastDefinitionsResetTemp, key)
		}
		for _, key := range b.lastDefinitionsResetTemp {
			delete(defs, key)
		}
	}
}

func (b *builder) AllocateInstruction() *Instruction {
	return b.instructionsPool.allocateInstruction()
}

// AllocateBasicBlock implements Builder.
func (b *builder) AllocateBasicBlock() BasicBlock {
	if l := len(b.basicBlocks); l <= b.nextBasicBlock {
		b.basicBlocks = append(b.basicBlocks, make([]basicBlock, 2*(l+1))...)
	}

	ret := &b.basicBlocks[b.nextBasicBlock]
	ret.id = b.nextBasicBlock
	b.nextBasicBlock++
	return ret
}

// InsertInstruction implements Builder.
func (b *builder) InsertInstruction(raw Value) {
	b.currentBB.InsertInstruction(raw)
}

// Blocks implements Builder.
func (b *builder) Blocks() []BasicBlock {
	if b.nextBasicBlock >= len(b.basicBlocksView) {
		b.basicBlocksView = append(b.basicBlocksView, make([]BasicBlock, b.nextBasicBlock)...)
	}
	for i := 0; i < b.nextBasicBlock; i++ {
		b.basicBlocksView[i] = &b.basicBlocks[i]
	}
	return b.basicBlocksView[:b.nextBasicBlock]
}

// DefineVariable implements Builder.
func (b *builder) DefineVariable(variable Variable, value Value, block BasicBlock) {
	if b.variables[variable] == TypeInvalid {
		panic("BUG: trying to define variable " + variable.String() + " but is not declared yet")
	}

	blockID := block.(*basicBlock).id
	if l := len(b.lastDefinitions); l <= blockID {
		maps := make([]map[Variable]Value, 2*(l+1))
		for i := range maps {
			maps[i] = make(map[Variable]Value)
		}
		b.lastDefinitions = append(b.lastDefinitions, maps...)
	}

	defs := b.lastDefinitions[blockID]
	defs[variable] = value

	block.InsertInstruction(value)
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

// BasicBlock is an identifier of a basic block in a SSA-transformed function.
type basicBlock struct {
	id                      int
	params                  []blockParam
	rootInstr, currentInstr *Instruction
	preds                   []*basicBlock
}

// AddParam implements BasicBlock.
func (bb *basicBlock) AddParam(b Builder, typ Type) Variable {
	variable := b.DeclareVariable(typ)
	n := len(bb.params)
	bb.params = append(bb.params, blockParam{bb: bb, typ: typ, n: n, variable: variable})
	return variable
}

// Params implements BasicBlock.
func (bb *basicBlock) Params() int {
	return len(bb.params)
}

// Param implements BasicBlock.
func (bb *basicBlock) Param(i int) (Variable, Value) {
	p := &bb.params[i]
	return p.variable, p
}

// InsertInstruction implements BasicBlock.
func (bb *basicBlock) InsertInstruction(raw Value) {
	next := raw.(*Instruction)
	current := bb.currentInstr
	if current != nil {
		current.next = next
		next.prev = current
	} else {
		bb.rootInstr = current
	}
	bb.currentInstr = next
}

// Root implements BasicBlock.
func (bb *basicBlock) Root() *Instruction {
	return bb.rootInstr
}

func (bb *basicBlock) reset() {
	bb.params = bb.params[:0]
	bb.rootInstr, bb.currentInstr = nil, nil
	bb.preds = bb.preds[:0]
}

// AddPred implements BasicBlock.
func (bb *basicBlock) AddPred(blk BasicBlock) {
	pred := blk.(*basicBlock)
	bb.preds = append(bb.preds, pred)
}

// String implements fmt.Stringer.
func (bb *basicBlock) String() string {
	ps := make([]string, len(bb.params))
	for i := range ps {
		ps[i] = bb.params[i].String()
	}
	preds := make([]string, len(bb.preds))
	for i, pred := range bb.preds {
		preds[i] = strconv.Itoa(pred.id)
	}
	return fmt.Sprintf("block[%d] (%s) <--(%s)",
		bb.id, strings.Join(ps, ","), strings.Join(preds, ","))
}

const instructionsPoolPageSize = 128

type (
	instructionsPoolPage = [instructionsPoolPageSize]Instruction
	instructionsPool     struct {
		pages []*instructionsPoolPage
		index int
	}
)

func (n *instructionsPool) allocateInstruction() *Instruction {
	if n.index == instructionsPoolPageSize {
		if len(n.pages) == cap(n.pages) {
			n.pages = append(n.pages, new(instructionsPoolPage))
		} else {
			i := len(n.pages)
			n.pages = n.pages[:i+1]
			if n.pages[i] == nil {
				n.pages[i] = new(instructionsPoolPage)
			}
		}
		n.index = 0
	}
	ret := &n.pages[len(n.pages)-1][n.index]
	n.index++
	return ret
}

func (n *instructionsPool) reset() {
	for _, ns := range n.pages {
		pages := ns[:]
		for i := range pages {
			pages[i] = Instruction{}
		}
	}
	n.pages = n.pages[:0]
	n.index = instructionsPoolPageSize
}
