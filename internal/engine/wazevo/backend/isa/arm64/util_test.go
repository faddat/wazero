package arm64

import (
	"strings"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

func getPendingInstr(m *machine) *instruction {
	return m.pendingInstructions[0]
}

func formatEmittedInstructions(m *machine) string {
	m.flushPendingInstructions()
	var strs []string
	for cur := m.head; cur != nil; cur = cur.next {
		strs = append(strs, cur.String())
	}
	return strings.Join(strs, "\n")
}

func newSetup() (ssa.Builder, *machine) {
	m := NewBackend().(*machine)
	ssaB := ssa.NewBuilder()
	backend.NewBackendCompiler(m, ssaB)
	blk := ssaB.AllocateBasicBlock()
	ssaB.SetCurrentBlock(blk)
	return ssaB, m
}

func newSetupWithMockContext() (*mockCompilationContext, ssa.Builder, *machine) {
	ctx := newMockCompilationContext()
	m := NewBackend().(*machine)
	m.SetCompilationContext(ctx)
	ssaB := ssa.NewBuilder()
	blk := ssaB.AllocateBasicBlock()
	ssaB.SetCurrentBlock(blk)
	return ctx, ssaB, m
}

func regToVReg(reg backend.RealReg) backend.VReg {
	return backend.VReg(0).SetRealReg(reg)
}

// mockCompilationContext implements backend.CompilationContext for testing.
type mockCompilationContext struct {
	vRegCounter int
	vRegMap     map[ssa.Value]backend.VReg
	definitions map[ssa.Value]*backend.SSAValueDefinition
	lowered     map[*ssa.Instruction]bool
}

func newMockCompilationContext() *mockCompilationContext {
	return &mockCompilationContext{
		vRegCounter: 0,
		vRegMap:     make(map[ssa.Value]backend.VReg),
		definitions: make(map[ssa.Value]*backend.SSAValueDefinition),
		lowered:     make(map[*ssa.Instruction]bool),
	}
}

// AllocateVReg implements backend.CompilationContext.
func (m *mockCompilationContext) AllocateVReg(regType backend.RegType) backend.VReg {
	m.vRegCounter++
	return backend.VReg(m.vRegCounter)
}

// MarkLowered implements backend.CompilationContext.
func (m *mockCompilationContext) MarkLowered(inst *ssa.Instruction) {
	m.lowered[inst] = true
}

// ValueDefinition implements backend.CompilationContext.
func (m *mockCompilationContext) ValueDefinition(value ssa.Value) *backend.SSAValueDefinition {
	definition, exists := m.definitions[value]
	if !exists {
		return nil
	}
	return definition
}

// VRegOf implements backend.CompilationContext.
func (m *mockCompilationContext) VRegOf(value ssa.Value) backend.VReg {
	vReg, exists := m.vRegMap[value]
	if !exists {
		panic("Value does not exist")
	}
	return vReg
}
