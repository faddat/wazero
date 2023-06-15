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
	c.assignVirtualRegisters()
	return nil, nil
}

// Reset should be called to allow this Compiler to use for the next function.
func (c *Compiler) Reset() {
	for i := vRegID(0); i < c.nextVRegID; i++ {
		c.ssaValuesToVRegs[i] = vRegInvalid
	}
	c.nextVRegID = 0
}

// assignVirtualRegisters assigns a virtual register to each ssa.ValueID valid in the ssa.Builder.
func (c *Compiler) assignVirtualRegisters() {
	builder := c.ssaBuilder
	refCounts := builder.ValueRefCountMap()

	if len(refCounts) >= len(c.ssaValuesToVRegs) {
		c.ssaValuesToVRegs = append(c.ssaValuesToVRegs,
			make([]vReg, len(refCounts))...)
	}

	for blk := builder.BlockIteratorBegin(); blk != nil; blk = builder.BlockIteratorNext() {
		// First we assign a virtual register to each parameter.
		for i := 0; i < blk.Params(); i++ {
			p := blk.Param(i)
			c.ssaValuesToVRegs[p.ID()] = c.allocateVReg()
		}

		// Assigns each value to a virtual register produced by instructions.
		for cur := blk.Root(); cur != nil; cur = cur.Next() {
			r, rs := cur.Returns()
			if r.Valid() {
				c.ssaValuesToVRegs[r.ID()] = c.allocateVReg()
			}
			for _, r := range rs {
				c.ssaValuesToVRegs[r.ID()] = c.allocateVReg()
			}
		}
	}

	// After assigned all values produced by instructions, we assign a virtual register to alias destination.
	for blk := builder.BlockIteratorBegin(); blk != nil; blk = builder.BlockIteratorNext() {
		for _, alias := range blk.Aliases() {
			src := c.ssaValuesToVRegs[alias.Src.ID()]
			if !src.valid() {
				panic("alias.Src must have a valid virtual register")
			}
			c.ssaValuesToVRegs[alias.Dst.ID()] = src
		}
	}
}

// allocateVReg allocates a new virtual register.
func (c *Compiler) allocateVReg() vReg {
	ret := vReg(c.nextVRegID)
	c.nextVRegID++
	return ret
}
