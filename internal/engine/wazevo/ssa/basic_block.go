package ssa

import (
	"fmt"
	"strings"
)

// BasicBlock represents the Basic Block of an SSA function.
// In traditional SSA terminology, the block "params" here are called phi values,
// and there does not exist "params". However, for simplicity, we handle them as parameters to a BB.
type BasicBlock interface {
	// Name returns the unique string ID of this block. e.g. blk0, blk1, ...
	Name() string

	// AddParam adds the parameter to the block whose type specified by `t`.
	AddParam(b Builder, t Type) (Variable, Value)

	// Params returns the number of parameters to this block.
	Params() int

	// Param returns (Variable, Value) which corresponds to the i-th parameter of this block.
	// The returned Variable can be used to add the definition of it in predecessors,
	// and the returned Value is the phi definition of a variable in this block.
	Param(i int) (Variable, Value)

	// InsertInstruction inserts an instruction that implements Value into the tail of this block.
	InsertInstruction(raw *Instruction)

	// AddPred appends `block` as a predecessor to this BB.
	// `branch` is an *Instruction used to reach to this block which holds the arguments.
	AddPred(block BasicBlock, branch *Instruction)

	// Root returns the root instruction of this block.
	Root() *Instruction

	// ReturnBlock returns ture if this block represents the function return.
	ReturnBlock() bool

	// FormatHeader returns the debug string of this block, not including instruction.
	FormatHeader(b Builder) string
}

type (
	// basicBlock is a basic block in a SSA-transformed function.
	basicBlock struct {
		id                      basicBlockID
		rootInstr, currentInstr *Instruction
		params                  []blockParam
		preds                   []basicBlockPredecessorInfo
		// singlePred is the alias to preds[0] for fast lookup, and only set after Seal is called.
		singlePred *basicBlock
		// lastDefinitions maps Variable to its last definition in this block.
		lastDefinitions map[Variable]Value
		// sealed is true if this is sealed (all the predecessors are known).
		sealed bool
		// unknownsValues are used in builder.findValue. The usage is well-described in the paper.
		unknownValues map[Variable]Value
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
func (bb *basicBlock) AddParam(b Builder, typ Type) (Variable, Value) {
	variable := b.DeclareVariable(typ)
	n := len(bb.params)
	paramValue := b.AllocateValue()
	bb.params = append(bb.params, blockParam{typ: typ, n: n, variable: variable, value: paramValue})
	b.DefineVariable(variable, paramValue, bb)
	return variable, paramValue
}

// addParamOn adds a parameter to this block whose variable is already defined.
// This is only used in the variable resolution.
func (bb *basicBlock) addParamOn(b *builder, variable Variable, value Value) {
	typ := b.variables[variable]
	if typ == TypeInvalid {
		panic("BUG: variable " + variable.String() + " not declared yet")
	}
	n := len(bb.params)
	bb.params = append(bb.params, blockParam{typ: typ, n: n, variable: variable, value: value})
	b.DefineVariable(variable, value, bb)
}

// Params implements BasicBlock.Params.
func (bb *basicBlock) Params() int {
	return len(bb.params)
}

// Param implements BasicBlock.Param.
func (bb *basicBlock) Param(i int) (Variable, Value) {
	p := &bb.params[i]
	return p.variable, p.value
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
}

// Root implements BasicBlock.Root.
func (bb *basicBlock) Root() *Instruction {
	return bb.rootInstr
}

func (bb *basicBlock) reset() {
	bb.params = bb.params[:0]
	bb.rootInstr, bb.currentInstr = nil, nil
	bb.preds = bb.preds[:0]
	// TODO: reuse the map!
	bb.unknownValues = make(map[Variable]Value)
	bb.lastDefinitions = make(map[Variable]Value)
}

// AddPred implements BasicBlock.AddPred.
func (bb *basicBlock) AddPred(blk BasicBlock, branch *Instruction) {
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
}

// FormatHeader implements BasicBlock.FormatHeader.
func (bb *basicBlock) FormatHeader(b Builder) string {
	ps := make([]string, len(bb.params))
	for i, p := range bb.params {
		ps[i] = p.format(b)
	}

	if len(bb.preds) > 0 {
		preds := make([]string, len(bb.preds))
		for i, pred := range bb.preds {
			preds[i] = fmt.Sprintf("blk%d", pred.blk.id)
			if len(pred.branch.vs) != len(bb.params) {
				panic("BUG: len(argument) != len(params)")
			}
		}
		return fmt.Sprintf("blk%d: (%s) <-- (%s)",
			bb.id, strings.Join(ps, ",v"), strings.Join(preds, ","))
	} else {
		return fmt.Sprintf("blk%d: (%s)", bb.id, strings.Join(ps, ", "))
	}
}

// blockParam implements Value and represents a parameter to a basicBlock.
type blockParam struct {
	// variable is a Variable for this parameter. This can be used to associate
	// the origins of this parameter with the defining instruction if .
	variable Variable
	// value represents the very first value that defines .variable in this block,
	// and can be considered as phi instruction.
	value Value
	typ   Type
	// n is the index of this blockParam in the bb.
	n int
}

func (p *blockParam) format(b Builder) (ret string) {
	return fmt.Sprintf("%s: %s", p.value.Format(b), p.typ)
}
