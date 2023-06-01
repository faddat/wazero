package ssa

import "github.com/tetratelabs/wazero/internal/wasm"

// NewBuilder returns a new Builder implementation.
func NewBuilder(m *wasm.Module) Builder {
	return &builder{
		m: m, currentFnIndex: 0,
	}
}

// Builder is used to builds SSA consisting of Basic Blocks per function.
type Builder interface {
	// Next must be called to reuse this builder for the next function.
	Next()

	// TODO.....
}

// builder implements Builder interface.
//
// We use the algorithm described in the paper:
// "Simple and Efficient Construction of Static Single Assignment Form" https://link.springer.com/content/pdf/10.1007/978-3-642-37051-9_6.pdf
//
// with the stricter assumption that our input is always a "complete" CFG.
type builder struct {
	m              *wasm.Module
	currentFnIndex wasm.Index
}

// Next implements Builder.
func (b *builder) Next() {
	b.currentFnIndex++
}

// BasicBlock is an identifier of a basic block in a SSA-transformed function.
type BasicBlock uint32

// Opcode represents a SSA instruction.
type Opcode uint32
