// Package ssa is used to construct SSA function. By nature this is free of Wasm specific thing
// and ISA.
package ssa

type (

	// Builder is used to builds SSA consisting of Basic Blocks per function.
	Builder interface {
		// Reset must be called to reuse this builder for the next function.
		Reset()

		// AllocateBasicBlock creates a basic block in SSA function.
		AllocateBasicBlock() BasicBlock

		// TODO.....
	}

	// BasicBlock represents the Basic Block of an SSA function.
	BasicBlock interface {
		// AddParam adds the parameter to the block whose type specified by `t`.
		AddParam(t Type)
	}
)

// NewBuilder returns a new Builder implementation.
func NewBuilder() Builder {
	return &builder{}
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
}

// Reset implements Builder.
func (b *builder) Reset() {
	for i := 0; i < b.nextBasicBlock; i++ {
		b.basicBlocks[i].reset()
	}
}

// AllocateBasicBlock implements Builder.
func (b *builder) AllocateBasicBlock() BasicBlock {
	ret := &b.basicBlocks[b.nextBasicBlock]
	b.nextBasicBlock++
	return ret
}

// BasicBlock is an identifier of a basic block in a SSA-transformed function.
type basicBlock struct {
	args []Type
}

// AddParam implements BasicBlock.
func (bb *basicBlock) AddParam(t Type) {
	bb.args = append(bb.args, t)
}

func (bb *basicBlock) reset() {
	bb.args = bb.args[:0]
}
