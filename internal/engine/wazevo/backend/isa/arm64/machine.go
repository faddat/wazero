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
		ctx                 backend.CompilationContext
		currentSSABlk       ssa.BasicBlock
		currentGID          ssa.InstructionGroupID
		instrPool           wazevoapi.Pool[instruction]
		pendingInstructions []*instruction
		head, tail          *instruction
		nextLabel           label

		// ssaBlockIDToLabels maps an SSA block ID to the label.
		ssaBlockIDToLabels []label
		// labelToInstructions maps a label to the instructions of the region which the label represents.
		labelPositions    map[label]*labelPosition
		labelPositionPool wazevoapi.Pool[labelPosition]
	}

	// label represents a position in the generated code which is either
	// a real instruction or the constant pool (e.g. jump tables).
	//
	// This is exactly the same as the traditional "label" in assembly code.
	label uint32

	// labelPosition represents the regions of the generated code which the label represents.
	labelPosition struct{ begin, end *instruction }
)

const invalidLabel = 0

// NewBackend returns a new backend for arm64.
func NewBackend() backend.Machine {
	return &machine{
		instrPool:      wazevoapi.NewPool[instruction](),
		labelPositions: make(map[label]*labelPosition),
		nextLabel:      invalidLabel,
	}
}

// Reset implements backend.Machine.
func (m *machine) Reset() {
	m.instrPool.Reset()
	m.ctx = nil
	m.currentSSABlk = nil
	m.nextLabel = invalidLabel
	m.pendingInstructions = m.pendingInstructions[:0]
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

// EndFunction implements backend.Machine.
func (m *machine) EndFunction() {}

// StartBlock implements backend.Machine.
func (m *machine) StartBlock(blk ssa.BasicBlock) {
	m.currentSSABlk = blk

	l := m.ssaBlockIDToLabels[m.currentSSABlk.ID()]
	if l == invalidLabel {
		l = m.allocateLabel()
		m.ssaBlockIDToLabels[blk.ID()] = l
	}

	end := m.allocateNop()
	m.insertAtHead(end)
	m.labelPositions[l] = &labelPosition{end, end}
}

func (m *machine) insert(i *instruction) {
	m.pendingInstructions = append(m.pendingInstructions, i)
}

func (m *machine) insert2(i1, i2 *instruction) {
	m.pendingInstructions = append(m.pendingInstructions, i1, i2)
}

func (m *machine) flushPendingInstructions() {
	l := len(m.pendingInstructions)
	if l == 0 {
		return
	}
	for i := l - 1; i > -0; i-- { // reverse because we lower instructions in reverse order.
		m.insertAtHead(m.pendingInstructions[i])
	}
	m.pendingInstructions = m.pendingInstructions[:0]
}

func (m *machine) insertAtHead(i *instruction) {
	if m.head == nil {
		m.head = i
		m.tail = i
		return
	}
	i.next = m.head
	m.head.prev = i
	m.head = i
}

// EndBlock implements backend.Machine.
func (m *machine) EndBlock() {
	l := m.ssaBlockIDToLabels[m.currentSSABlk.ID()]
	m.labelPositions[l].begin = m.head
}

// String implements backend.Machine.
func (l label) String() string {
	return fmt.Sprintf("L%d", l)
}

func (m *machine) allocateInstr() *instruction {
	instr := m.instrPool.Allocate()
	return instr
}

func (m *machine) allocateNop() *instruction {
	instr := m.instrPool.Allocate()
	instr.asNop0()
	return instr
}

func (m *machine) getOrAllocateSSABlockLabel(blk ssa.BasicBlock) label {
	l := m.ssaBlockIDToLabels[blk.ID()]
	if l == invalidLabel {
		l = m.allocateLabel()
		m.ssaBlockIDToLabels[blk.ID()] = l
	}
	return l
}

func (m *machine) setCurrentInstructionGroupID(gid ssa.InstructionGroupID) {
	m.currentGID = gid
}
