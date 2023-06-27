package backend

import "github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"

type (
	// Machine is a backend for a specific machine.
	Machine interface {
		StartBlock(blk ssa.BasicBlock)

		LowerInstr(*ssa.Instruction)

		EndBlock()

		// Reset resets the machine state for the next compilation.
		Reset()
	}

	// CompilationContext is a context for a function-scoped context passed to MachineBackend
	// to perform the lowering in the machine specific backend for the given function.
	CompilationContext interface{}
)

// Ensures that compiler[T] implements CompilationContext.
var _ CompilationContext = (*compiler[nopMachineBackend])(nil)

// nopMachineBackend is a MachineBackend that does nothing.
// Defined here to do the type assertion below.
type nopMachineBackend struct{}

// StartBlock implements MachineBackend.StartBlock.
func (b nopMachineBackend) StartBlock(ssa.BasicBlock) {}

// LowerInstr implements MachineBackend.LowerInstr.
func (b nopMachineBackend) LowerInstr(*ssa.Instruction) {}

// EndBlock implements MachineBackend.EndBlock.
func (b nopMachineBackend) EndBlock() {}

// Reset implements MachineBackend.Reset.
func (b nopMachineBackend) Reset() {}
