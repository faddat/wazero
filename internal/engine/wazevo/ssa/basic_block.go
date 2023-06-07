package ssa

import (
	"fmt"
	"strings"
)

// BasicBlock represents the Basic Block of an SSA function.
// In traditional SSA terminology, the block "params" here are called phi values,
// and there does not exist "params". However, for simplicity, we handle them as parameters to a BB.
type BasicBlock interface {
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
	InsertInstruction(raw *Instruction)

	// AddPred appends `block` as a predecessor to this BB.
	AddPred(block BasicBlock)

	// Root returns the root instruction of this block.
	Root() *Instruction
}

// BasicBlock is an identifier of a basic block in a SSA-transformed function.
type basicBlock struct {
	id                      int
	rootInstr, currentInstr *Instruction
	params                  []blockParam
	preds                   []*basicBlock
	// lastDefinitions track last definitions of a variable in each block.
	lastDefinitions map[Variable]Value
}

// AddParam implements BasicBlock.
func (bb *basicBlock) AddParam(b Builder, typ Type) Variable {
	variable := b.DeclareVariable(typ)
	n := len(bb.params)

	paramValue := b.AllocateValue()
	bb.params = append(bb.params, blockParam{typ: typ, n: n, variable: variable, value: paramValue})
	b.DefineVariable(variable, paramValue, bb)
	return variable
}

// Params implements BasicBlock.
func (bb *basicBlock) Params() int {
	return len(bb.params)
}

// Param implements BasicBlock.
func (bb *basicBlock) Param(i int) (Variable, Value) {
	p := &bb.params[i]
	return p.variable, p.value
}

// InsertInstruction implements BasicBlock.
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

// String implements fmt.Stringer. Only used for debugging.
func (bb *basicBlock) String() string {
	ps := make([]string, len(bb.params))
	for i, p := range bb.params {
		ps[i] = p.String()
	}

	if len(bb.preds) > 0 {
		preds := make([]string, len(bb.preds))
		for i, pred := range bb.preds {
			preds[i] = fmt.Sprintf("blk%d", pred.id)
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

// String implements Value.
func (p *blockParam) String() (ret string) {
	return fmt.Sprintf("%s: %s", p.value, p.typ)
}
