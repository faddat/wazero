package ssa

import (
	"fmt"
	"strings"
)

// BasicBlock represents the Basic Block of an SSA function.
//
// Note: we use the "block argument" variant of SSA, instead of PHI functions. See the package level doc comments.
//
// Note: we use "parameter/param" as a placeholder which represents a variant of PHI, and "argument/arg" as an actual
// Value passed to that "parameter/param".
type BasicBlock interface {
	// Name returns the unique string ID of this block. e.g. blk0, blk1, ...
	Name() string

	// AddParam adds the parameter to the block whose type specified by `t`.
	AddParam(b Builder, t Type) Value

	// Params returns the number of parameters to this block.
	Params() int

	// Param returns (Variable, Value) which corresponds to the i-th parameter of this block.
	// The returned Value is the definition of the param in this block.
	Param(i int) Value

	// InsertInstruction inserts an instruction that implements Value into the tail of this block.
	InsertInstruction(raw *Instruction)

	// Root returns the root instruction of this block.
	Root() *Instruction

	// ReturnBlock returns ture if this block represents the function return.
	ReturnBlock() bool

	// FormatHeader returns the debug string of this block, not including instruction.
	FormatHeader(b Builder) string

	// Valid is true if this block is still valid even after optimizations.
	Valid() bool
}

type (
	// basicBlock is a basic block in a SSA-transformed function.
	basicBlock struct {
		id                      basicBlockID
		rootInstr, currentInstr *Instruction
		params                  []blockParam
		preds                   []basicBlockPredecessorInfo
		success                 []*basicBlock
		// singlePred is the alias to preds[0] for fast lookup, and only set after Seal is called.
		singlePred *basicBlock
		// lastDefinitions maps Variable to its last definition in this block.
		lastDefinitions map[Variable]Value
		// sealed is true if this is sealed (all the predecessors are known).
		sealed bool
		// unknownsValues are used in builder.findValue. The usage is well-described in the paper.
		unknownValues map[Variable]Value
		// invalid is true if this block is made invalid during optimizations.
		invalid bool
	}
	basicBlockID uint32
)

const basicBlockIDReturnBlock = 0xffffffff

// BasicBlockReturn is a special BasicBlock which represents a function return which
// can be a virtual target of branch instructions.
var BasicBlockReturn BasicBlock = &basicBlock{id: basicBlockIDReturnBlock}

// Name implements BasicBlock.Name.
func (bb *basicBlock) Name() string {
	if bb.id == basicBlockIDReturnBlock {
		return "blk_ret"
	} else {
		return fmt.Sprintf("blk%d", bb.id)
	}
}

type basicBlockPredecessorInfo struct {
	blk    *basicBlock
	branch *Instruction
}

// ReturnBlock implements BasicBlock.ReturnBlock.
func (bb *basicBlock) ReturnBlock() bool {
	return bb.id == basicBlockIDReturnBlock
}

// AddParam implements BasicBlock.AddParam.
func (bb *basicBlock) AddParam(b Builder, typ Type) Value {
	paramValue := b.allocateValue(typ)
	bb.params = append(bb.params, blockParam{typ: typ, value: paramValue})
	return paramValue
}

// addParamOn adds a parameter to this block whose value is already allocated.
func (bb *basicBlock) addParamOn(typ Type, value Value) {
	bb.params = append(bb.params, blockParam{typ: typ, value: value})
}

// Params implements BasicBlock.Params.
func (bb *basicBlock) Params() int {
	return len(bb.params)
}

// Param implements BasicBlock.Param.
func (bb *basicBlock) Param(i int) Value {
	p := &bb.params[i]
	return p.value
}

// Valid implements BasicBlock.Valid.
func (bb *basicBlock) Valid() bool {
	return !bb.invalid
}

// InsertInstruction implements BasicBlock.InsertInstruction.
func (bb *basicBlock) InsertInstruction(next *Instruction) {
	current := bb.currentInstr
	if current != nil {
		current.next = next
		next.prev = current
	} else {
		bb.rootInstr = next
	}
	bb.currentInstr = next

	switch next.opcode {
	case OpcodeJump, OpcodeBrz, OpcodeBrnz:
		target := next.blk.(*basicBlock)
		target.addPred(bb, next)
	case OpcodeBrTable:
		panic(OpcodeBrTable)
	}
}

// Root implements BasicBlock.Root.
func (bb *basicBlock) Root() *Instruction {
	return bb.rootInstr
}

func (bb *basicBlock) reset() {
	bb.params = bb.params[:0]
	bb.rootInstr, bb.currentInstr = nil, nil
	bb.preds = bb.preds[:0]
	bb.success = bb.success[:0]
	bb.invalid, bb.sealed = false, false
	bb.singlePred = nil
	// TODO: reuse the map!
	bb.unknownValues = make(map[Variable]Value)
	bb.lastDefinitions = make(map[Variable]Value)
}

func (bb *basicBlock) addPred(blk BasicBlock, branch *Instruction) {
	if blk.ReturnBlock() {
		// Return Block does not need to know the predecessors.
		return
	}
	if bb.sealed {
		panic("BUG: trying to add predecessor to a sealed block: " + bb.Name())
	}
	pred := blk.(*basicBlock)
	bb.preds = append(bb.preds, basicBlockPredecessorInfo{
		blk:    pred,
		branch: branch,
	})

	pred.success = append(pred.success, bb)
}

// FormatHeader implements BasicBlock.FormatHeader.
func (bb *basicBlock) FormatHeader(b Builder) string {
	ps := make([]string, len(bb.params))
	for i, p := range bb.params {
		ps[i] = p.value.formatWithType(b)
	}

	if len(bb.preds) > 0 {
		preds := make([]string, 0, len(bb.preds))
		for _, pred := range bb.preds {
			if len(pred.branch.vs) != len(bb.params) {
				panic(fmt.Sprintf("BUG: len(argument) != len(params): %d != %d",
					len(pred.branch.vs), len(bb.params)))
			}
			if pred.blk.invalid {
				continue
			}
			preds = append(preds, fmt.Sprintf("blk%d", pred.blk.id))

		}
		return fmt.Sprintf("blk%d: (%s) <-- (%s)",
			bb.id, strings.Join(ps, ","), strings.Join(preds, ","))
	} else {
		return fmt.Sprintf("blk%d: (%s)", bb.id, strings.Join(ps, ", "))
	}
}

// blockParam implements Value and represents a parameter to a basicBlock.
type blockParam struct {
	// value is the Value that corresponds to the parameter in this block,
	// and can be considered as an output of PHI instruction in traditional SSA.
	value Value
	typ   Type
}
