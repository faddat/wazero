package arm64

import (
	"fmt"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
)

type (
	// machine implements backend.Machine.
	machine struct {
		ctx           backend.CompilationContext
		currentSSABlk ssa.BasicBlock
		instrPool     wazevoapi.Pool[instruction]
		head, tail    *instruction
		nextLabel     label

		// ssaBlockIDToLabels maps a SSA block ID to the labels of the generated code.
		ssaBlockIDToLabels []label
	}

	// label represents a position in the generated code which is either
	// a real instruction or the constant pool (e.g. jump tables).
	//
	// This is exactly the same as the traditional "label" in assembly code.
	label uint32
)

// NewBackend returns a new backend for arm64.
func NewBackend() backend.Machine {
	return &machine{
		instrPool: wazevoapi.NewPool[instruction](),
	}
}

// Reset implements backend.Machine.
func (m *machine) Reset() {
	m.instrPool.Reset()
	m.ctx = nil
	m.currentSSABlk = nil
}

// allocateLabel allocates an unused label.
func (m *machine) allocateLabel() label {
	m.nextLabel++
	return m.nextLabel
}

// SetCompilationContext implements backend.Machine.
func (m *machine) SetCompilationContext(ctx backend.CompilationContext) {
	m.ctx = ctx
}

// StartFunction implements backend.Machine.
func (m *machine) StartFunction(n int) {
	if len(m.ssaBlockIDToLabels) <= n {
		// Eagerly allocate labels for the blocks since the underlying slice will be used for the next iteration.
		m.ssaBlockIDToLabels = append(m.ssaBlockIDToLabels, make([]label, n)...)
	}
}

// StartBlock implements backend.Machine.
func (m *machine) StartBlock(blk ssa.BasicBlock) {
	m.currentSSABlk = blk
	l := m.allocateLabel()
	m.ssaBlockIDToLabels[blk.ID()] = l
}

// EndBlock implements backend.Machine.
func (m *machine) EndBlock() {}

// String implements backend.Machine.
func (l label) String() string {
	return fmt.Sprintf("L%d", l)
}
