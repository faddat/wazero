package arm64

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

// NewBackend returns a new backend for arm64.
func NewBackend() backend.Machine {
	return &machine{}
}

// backend implements backend.Machine.
type machine struct {
	ctx           backend.CompilationContext
	currentSSABlk ssa.BasicBlock
}

var _ backend.Machine = (*machine)(nil)

// SetCompilationContext implements backend.Machine.
func (m *machine) SetCompilationContext(backend.CompilationContext) {
}

// Reset implements backend.Machine.
func (m *machine) Reset() {
}

// StartBlock implements backend.Machine.
func (m *machine) StartBlock(blk ssa.BasicBlock) {
	m.currentSSABlk = blk
}

// LowerInstr implements backend.Machine.
func (m *machine) LowerInstr(*ssa.Instruction) {}

// EndBlock implements backend.Machine.
func (m *machine) EndBlock() {}
