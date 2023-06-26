// Package backend must be free of Wasm-specific concept. In other words,
// this package must not import internal/wasm package.
package backend

import "github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"

// NewBackendCompiler returns a new Compiler that can generate a machine code.
func NewBackendCompiler(builder ssa.Builder) *Compiler {
	return &Compiler{ssaBuilder: builder}
}

// Compiler is the backend of wazevo which takes ssa.Builder and
// use the information there to emit the final machine code.
type Compiler struct {
	ssaBuilder ssa.Builder
	// ssaValuesToVRegs maps ssa.ValueID to vReg.
	ssaValuesToVRegs []vReg
	// nextVRegID is the next virtual register ID to be allocated.
	nextVRegID vRegID
}

// Generate generates the machine code.
func (c *Compiler) Generate() ([]byte, error) {
	return nil, nil
}

// Reset should be called to allow this Compiler to use for the next function.
func (c *Compiler) Reset() {
	for i := vRegID(0); i < c.nextVRegID; i++ {
		c.ssaValuesToVRegs[i] = vRegInvalid
	}
	c.nextVRegID = 0
}
