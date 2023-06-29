package arm64

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"strings"
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

func regToVReg(reg backend.RealReg) backend.VReg {
	return backend.VReg(0).SetRealReg(reg)
}
