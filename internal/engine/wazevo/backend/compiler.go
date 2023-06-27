package backend

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

// NewBackendCompiler returns a new Compiler that can generate a machine code.
//
// The type parameter T must be a type that implements MachineBackend.
func NewBackendCompiler(mach Machine, builder ssa.Builder) Compiler {
	c := &compiler{
		mach: mach, ssaBuilder: builder,
		alreadyLowered: make(map[*ssa.Instruction]struct{}),
	}
	mach.SetCompilationContext(c)
	return c
}

// Compiler is the backend of wazevo which lowers the state stored in ssa.Builder
// into the ISA-specific machine code.
type Compiler interface {
	CompilationContext

	// Compile lowers the state stored in ssa.Builder into the ISA-specific machine code.
	Compile() ([]byte, error)

	// MarkLowered is used to mark the given instruction as already lowered
	// which tells the compiler to skip it when traversing.
	MarkLowered(inst *ssa.Instruction)

	// Reset should be called to allow this Compiler to use for the next function.
	Reset()
}

// Compiler is the backend of wazevo which takes ssa.Builder and
// use the information there to emit the final machine code.
type compiler struct {
	mach       Machine
	ssaBuilder ssa.Builder
	// nextVRegID is the next virtual register ID to be allocated.
	nextVRegID VRegID
	// ssaValuesToVRegs maps ssa.ValueID to VReg.
	ssaValuesToVRegs []VReg
	// ssaValueDefinitions maps ssa.ValueID to its definition.
	ssaValueDefinitions []SSAValueDefinition
	// returnVRegs is the list of virtual registers that store the return values.
	returnVRegs []VReg

	alreadyLowered map[*ssa.Instruction]struct{}
}

// Compile implements Compiler.Compile.
func (c *compiler) Compile() ([]byte, error) {
	c.assignVirtualRegisters()
	c.lowerBlocks()
	return nil, nil
}

// lowerBlocks lowers each block in the ssa.Builder.
func (c *compiler) lowerBlocks() {
	builder := c.ssaBuilder
	for blk := builder.BlockIteratorReversePostOrderBegin(); blk != nil; blk = builder.BlockIteratorReversePostOrderNext() {
		c.lowerBlock(blk)
	}
}

func (c *compiler) lowerBlock(blk ssa.BasicBlock) {
	mach := c.mach
	mach.StartBlock(blk)
	// We traverse the instructions in reverse order because we might want to lower multiple
	// instructions together.
	for cur := blk.Tail(); cur != nil; cur = cur.Prev() {
		if _, ok := c.alreadyLowered[cur]; ok {
			continue
		}
		mach.LowerInstr(cur)
	}
	mach.EndBlock()
}

// assignVirtualRegisters assigns a virtual register to each ssa.ValueID Valid in the ssa.Builder.
func (c *compiler) assignVirtualRegisters() {
	builder := c.ssaBuilder
	refCounts := builder.ValueRefCountMap()

	if len(refCounts) >= len(c.ssaValuesToVRegs) {
		c.ssaValuesToVRegs = append(c.ssaValuesToVRegs,
			make([]VReg, len(refCounts))...)
	}
	if len(refCounts) >= len(c.ssaValueDefinitions) {
		c.ssaValueDefinitions = append(c.ssaValueDefinitions,
			make([]SSAValueDefinition, len(refCounts))...)
	}

	for blk := builder.BlockIteratorReversePostOrderBegin(); blk != nil; blk = builder.BlockIteratorReversePostOrderNext() {
		// First we assign a virtual register to each parameter.
		for i := 0; i < blk.Params(); i++ {
			p := blk.Param(i).ID()
			c.ssaValuesToVRegs[p] = c.allocateVReg()
			c.ssaValueDefinitions[p] = SSAValueDefinition{
				isBlockParam: true,
				blk:          blk,
				n:            i,
				refCount:     refCounts[p],
			}
		}

		// Assigns each value to a virtual register produced by instructions.
		for cur := blk.Root(); cur != nil; cur = cur.Next() {
			r, rs := cur.Returns()
			if r.Valid() {
				id := r.ID()
				c.ssaValuesToVRegs[id] = c.allocateVReg()
				c.ssaValueDefinitions[id] = SSAValueDefinition{
					isBlockParam: false,
					instr:        cur,
					n:            0,
					refCount:     refCounts[id],
				}
			}
			for i, r := range rs {
				id := r.ID()
				c.ssaValuesToVRegs[id] = c.allocateVReg()
				c.ssaValueDefinitions[id] = SSAValueDefinition{
					isBlockParam: false,
					instr:        cur,
					n:            i,
					refCount:     refCounts[id],
				}
			}
		}
	}

	for i := 0; i < builder.ReturnBlock().Params(); i++ {
		c.returnVRegs = append(c.returnVRegs, c.allocateVReg())
	}
}

// allocateVReg allocates a new virtual register.
func (c *compiler) allocateVReg() VReg {
	ret := VReg(c.nextVRegID)
	c.nextVRegID++
	return ret
}

// Reset implements Compiler.Reset.
func (c *compiler) Reset() {
	for i := VRegID(0); i < c.nextVRegID; i++ {
		c.ssaValuesToVRegs[i] = vRegInvalid
	}
	c.nextVRegID = 0
	c.returnVRegs = c.returnVRegs[:0]
	c.mach.Reset()
}

// MarkLowered implements CompilationContext.MarkLowered.
func (c *compiler) MarkLowered(inst *ssa.Instruction) {
	c.alreadyLowered[inst] = struct{}{}
}

// ValueDefinition implements CompilationContext.ValueDefinition.
func (c *compiler) ValueDefinition(value ssa.Value) *SSAValueDefinition {
	return &c.ssaValueDefinitions[value.ID()]
}
