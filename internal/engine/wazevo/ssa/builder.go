// Package ssa is used to construct SSA function. By nature this is free of Wasm specific thing
// and ISA.
package ssa

// NewBuilder returns a new Builder implementation.
func NewBuilder() Builder {
	return &builder{}
}

// Builder is used to builds SSA consisting of Basic Blocks per function.
type Builder interface {
	// Reset must be called to reuse this builder for the next function.
	Reset()

	// TODO.....
}

// builder implements Builder interface.
//
// We use the algorithm described in the paper:
// "Simple and Efficient Construction of Static Single Assignment Form" https://link.springer.com/content/pdf/10.1007/978-3-642-37051-9_6.pdf
//
// with the stricter assumption that our input is always a "complete" CFG.
type builder struct {
}

// Reset implements Builder.
func (b *builder) Reset() {}

// BasicBlock is an identifier of a basic block in a SSA-transformed function.
type BasicBlock uint32
