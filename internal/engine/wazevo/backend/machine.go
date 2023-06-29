package backend

import "github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"

type (
	// Machine is a backend for a specific machine.
	Machine interface {
		// SetCompilationContext sets the compilation context used for the lifetime of Machine.
		// This is only called once per Machine, i.e. before the first compilation.
		SetCompilationContext(CompilationContext)

		// StartFunction is called when the compilation of the given function is started.
		// n is the number of ssa.BasicBlock(s) existing in the function.
		StartFunction(n int)

		// EndFunction is called when the compilation of the current function is finished.
		EndFunction()

		// StartBlock is called when the compilation of the given block is started.
		StartBlock(ssa.BasicBlock)

		// EndBlock is called when the compilation of the current block is finished.
		EndBlock()

		// LowerBranches is called right after StartBlock and before LowerInstr if
		// there are branches to the given block. br0 is the very end of the block and b1 is the before the br0 if it exists.
		// At least br0 is not nil, but br1 can be nil if there's no branching before br0.
		//
		// See ssa.Instruction IsBranching, and the comment on ssa.BasicBlock.
		LowerBranches(br0, br1 *ssa.Instruction)

		// LowerInstr is called for each instruction in the given block except for the ones marked as already lowered
		// via CompilationContext.MarkLowered. The order is reverse, i.e. from the last instruction to the first one.
		//
		// Note: this can lower multiple instructions (which produce the inputs) at once whenever it's possible
		// for optimization.
		LowerInstr(*ssa.Instruction)

		// Reset resets the machine state for the next compilation.
		Reset()
	}

	// CompilationContext is passed to MachineBackend to perform the lowering in the machine specific backend by
	// leveraging the information held by *compiler.
	CompilationContext interface {
		// AllocateVReg allocates a new virtual register of the given type.
		AllocateVReg(regType RegType) VReg

		// MarkLowered is used to mark the given instruction as already lowered
		// which tells the compiler to skip it when traversing.
		MarkLowered(inst *ssa.Instruction)

		// ValueDefinition returns the definition of the given value.
		ValueDefinition(ssa.Value) *SSAValueDefinition

		// VRegOf returns the virtual register of the given ssa.Value.
		VRegOf(value ssa.Value) VReg
	}
)

var _ CompilationContext = (*compiler)(nil)
