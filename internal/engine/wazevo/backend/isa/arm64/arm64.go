package arm64

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

// NewBackend returns a new backend for arm64.
func NewBackend() backend.Machine {
	return &machine{}
}

// backend implements isa.MachineBackend.
type machine struct{}

// Reset implements isa.MachineBackend.
func (b *machine) Reset() {
}

// StartBlock implements backend.Machine.
func (b *machine) StartBlock(blk ssa.BasicBlock) {}

// LowerInstr implements backend.MachineBackend.
func (b *machine) LowerInstr(*ssa.Instruction) {}

// EndBlock implements backend.MachineBackend.
func (b *machine) EndBlock() {}

var _ backend.Machine = (*machine)(nil)
