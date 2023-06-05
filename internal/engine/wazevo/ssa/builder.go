// Package ssa is used to construct SSA function. By nature this is free of Wasm specific thing
// and ISA.
package ssa

import (
	"fmt"
	"strings"
)

type (

	// Builder is used to builds SSA consisting of Basic Blocks per function.
	Builder interface {
		// Reset must be called to reuse this builder for the next function.
		Reset()

		// AllocateBasicBlock creates a basic block in SSA function.
		AllocateBasicBlock() BasicBlock

		// SetCurrentBlock sets the instruction insertion target to the BasicBlock `b`.
		SetCurrentBlock(b BasicBlock)

		// DeclareVariable declares a Variable of the given Type.
		DeclareVariable(Variable, Type)

		// AnnotateVariable associate the given Variable with `annotation` for debugging purpose.
		AnnotateVariable(variable Variable, annotation string)

		// DefineVariable defines a variable in the `block` with value.
		DefineVariable(variable Variable, value Value, block BasicBlock)
	}

	// BasicBlock represents the Basic Block of an SSA function.
	BasicBlock interface {
		// AddParam adds the parameter to the block whose type specified by `t`.
		AddParam(t Type) Value

		// Param returns Value which corresponds to the i-th parameter of this block.
		Param(i int) Value
	}
)

// NewBuilder returns a new Builder implementation.
func NewBuilder() Builder {
	return &builder{variableAnnotations: make(map[Variable]string)}
}

// builder implements Builder interface.
//
// We use the algorithm described in the paper:
// "Simple and Efficient Construction of Static Single Assignment Form" https://link.springer.com/content/pdf/10.1007/978-3-642-37051-9_6.pdf
//
// with the stricter assumption that our input is always a "complete" CFG.
type builder struct {
	nextBasicBlock int
	basicBlocks    []basicBlock
	currentBB      *basicBlock

	// variables track the types for Variable with the index regarded Variable.
	variables           []Type
	variablesDefined    int
	variableAnnotations map[Variable]string

	// lastDefinitions track last definitions of a variable in each block.
	lastDefinitions          []map[Variable]Value
	lastDefinitionsResetTemp []Variable
}

// Reset implements Builder.
func (b *builder) Reset() {
	for i := 0; i < b.nextBasicBlock; i++ {
		b.basicBlocks[i].reset()
	}

	for i := 0; i < b.variablesDefined; i++ {
		b.variables[i] = TypeInvalid
		delete(b.variableAnnotations, Variable(i))
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

// DefineVariable implements Builder.
func (b *builder) DefineVariable(variable Variable, value Value, block BasicBlock) {
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
}

// SetCurrentBlock implements Builder.
func (b *builder) SetCurrentBlock(bb BasicBlock) {
	b.currentBB = bb.(*basicBlock)
}

// DeclareVariable implements Builder.
func (b *builder) DeclareVariable(v Variable, typ Type) {
	iv := int(v)
	if l := len(b.variables); l <= iv {
		b.variables = append(b.variables, make([]Type, 2*(l+1))...)
	}
	b.variables[v] = typ

	if iv > b.variablesDefined {
		b.variablesDefined = iv
	}
	return
}

// AnnotateVariable implements Builder.
func (b *builder) AnnotateVariable(variable Variable, annotation string) {
	b.variableAnnotations[variable] = annotation
	return
}

// BasicBlock is an identifier of a basic block in a SSA-transformed function.
type basicBlock struct {
	id     int
	params []blockParam
}

// AddParam implements BasicBlock.
func (bb *basicBlock) AddParam(typ Type) Value {
	n := len(bb.params)
	bb.params = append(bb.params, blockParam{bb: bb, typ: typ, n: n})
	return &bb.params[n]
}

func (bb *basicBlock) reset() {
	bb.params = bb.params[:0]
}

// Param implements BasicBlock.
func (bb *basicBlock) Param(i int) Value {
	return &bb.params[i]
}

// String implements fmt.Stringer.
func (bb *basicBlock) String() string {
	ps := make([]string, len(bb.params))
	for i := range ps {
		ps[i] = bb.params[i].String()
	}
	return fmt.Sprintf("block[%d] (%s)", bb.id, strings.Join(ps, ","))
}
