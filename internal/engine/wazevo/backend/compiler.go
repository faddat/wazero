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
		nextVRegID:     0,
	}
	mach.SetCompilationContext(c)
	return c
}

// Compiler is the backend of wazevo which lowers the state stored in ssa.Builder
// into the ISA-specific machine code.
type Compiler interface {
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
	// vRegToRegType maps VRegID to its register type.
	vRegToRegType []RegType
	// returnVRegs is the list of virtual registers that store the return values.
	returnVRegs []VReg

	alreadyLowered map[*ssa.Instruction]struct{}
}

// Compile implements Compiler.Compile.
func (c *compiler) Compile() ([]byte, error) {
	c.assignVirtualRegisters()
	c.mach.StartFunction(c.ssaBuilder.Blocks())
	c.lowerBlocks()
	c.mach.EndFunction()
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
	cur := blk.Tail()

	// First gather the branching instructions at the end of the blocks.
	var br0, br1 *ssa.Instruction
	if cur.IsBranching() {
		br0 = cur
		cur = cur.Prev()
		if cur != nil && cur.IsBranching() {
			br1 = cur
			cur = cur.Prev()
		}
	}

	if br0 != nil {
		mach.LowerBranches(br0, br1)
	}

	if br1 != nil && br0 == nil {
		panic("BUG? when a block has conditional branch but doesn't end with an unconditional branch?")
	}

	// Now start lowering the non-branching instructions.
	for ; cur != nil; cur = cur.Prev() {
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

	need := len(refCounts)
	if need >= len(c.ssaValuesToVRegs) {
		c.ssaValuesToVRegs = append(c.ssaValuesToVRegs, make([]VReg, need)...)
	}
	if need >= len(c.ssaValueDefinitions) {
		c.ssaValueDefinitions = append(c.ssaValueDefinitions, make([]SSAValueDefinition, need)...)
	}

	for blk := builder.BlockIteratorReversePostOrderBegin(); blk != nil; blk = builder.BlockIteratorReversePostOrderNext() {
		// First we assign a virtual register to each parameter.
		for i := 0; i < blk.Params(); i++ {
			p := blk.Param(i)
			pid := p.ID()
			vreg := c.AllocateVReg(RegTypeOf(p.Type()))
			c.ssaValuesToVRegs[pid] = vreg
			c.ssaValueDefinitions[pid] = SSAValueDefinition{BlkParamVReg: vreg}
		}

		// Assigns each value to a virtual register produced by instructions.
		for cur := blk.Root(); cur != nil; cur = cur.Next() {
			r, rs := cur.Returns()
			if r.Valid() {
				id := r.ID()
				c.ssaValuesToVRegs[id] = c.AllocateVReg(RegTypeOf(r.Type()))
				c.ssaValueDefinitions[id] = SSAValueDefinition{
					Instr:    cur,
					N:        0,
					RefCount: refCounts[id],
				}
			}
			for i, r := range rs {
				id := r.ID()
				c.ssaValuesToVRegs[id] = c.AllocateVReg(RegTypeOf(r.Type()))
				c.ssaValueDefinitions[id] = SSAValueDefinition{
					Instr:    cur,
					N:        i,
					RefCount: refCounts[id],
				}
			}
		}
	}

	for i, retBlk := 0, builder.ReturnBlock(); i < retBlk.Params(); i++ {
		typ := retBlk.Param(i).Type()
		c.returnVRegs = append(c.returnVRegs, c.AllocateVReg(RegTypeOf(typ)))
	}
}

// AllocateVReg implements CompilationContext.AllocateVReg.
func (c *compiler) AllocateVReg(regType RegType) VReg {
	r := VReg(c.nextVRegID)
	if ir := int(r); len(c.vRegToRegType) <= ir {
		// Eagerly allocate the slice to reduce reallocation in the future iterations.
		c.vRegToRegType = append(c.vRegToRegType, make([]RegType, ir+1)...)
	}
	c.vRegToRegType[r.ID()] = regType
	c.nextVRegID++
	return r
}

// Reset implements Compiler.Reset.
func (c *compiler) Reset() {
	for i := VRegID(0); i < c.nextVRegID; i++ {
		c.ssaValuesToVRegs[i] = vRegInvalid
		c.vRegToRegType[i] = RegTypeInvalid
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

// VRegOf implements CompilationContext.VRegOf.
func (c *compiler) VRegOf(value ssa.Value) VReg {
	return c.ssaValuesToVRegs[value.ID()]
}
