package backend

import "github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"

type (
	// Machine is a backend for a specific machine.
	Machine interface {
		// SetCompilationContext sets the compilation context used for the lifetime of Machine.
		// This is only called once per Machine, i.e. before the first compilation.
		SetCompilationContext(CompilationContext)

		// StartBlock is called when the compilation of the given block is started.
		StartBlock(ssa.BasicBlock)

		// LowerInstr is called for each instruction in the given block except for the ones marked as already lowered
		// via CompilationContext.MarkLowered. The order is reverse, i.e. from the last instruction to the first one.
		LowerInstr(*ssa.Instruction)

		// EndBlock is called when the compilation of the current block is finished.
		EndBlock()

		// Reset resets the machine state for the next compilation.
		Reset()
	}

	// CompilationContext is passed to MachineBackend to perform the lowering in the machine specific backend by
	// leveraging the information held by *compiler.
	CompilationContext interface {
		// MarkLowered is used to mark the given instruction as already lowered
		// which tells the compiler to skip it when traversing.
		MarkLowered(inst *ssa.Instruction)
	}
)
