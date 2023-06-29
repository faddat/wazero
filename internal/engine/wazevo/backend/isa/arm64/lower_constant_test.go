package arm64

import (
	"math"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

func getPendingInstr(m *machine) *instruction {
	return m.pendingInstructions[0]
}

func formatEmittedInstructions(m *machine) string {
	m.flushPendingInstructions()
	builder := strings.Builder{}
	for cur := m.tail; cur != nil; cur = cur.next {
		builder.WriteString(cur.String())
	}
	return builder.String()
}

func newSetup() (ssa.Builder, *machine) {
	m := NewBackend().(*machine)
	ssaB := ssa.NewBuilder()
	backend.NewBackendCompiler(m, ssaB)
	blk := ssaB.AllocateBasicBlock()
	ssaB.SetCurrentBlock(blk)
	return ssaB, m
}

func TestMachine_lowerConstant(t *testing.T) {
	t.Run("TypeF32", func(t *testing.T) {
		ssaB, m := newSetup()
		ssaConstInstr := ssaB.AllocateInstruction()
		ssaConstInstr.AsF32const(1.1234)
		ssaB.InsertInstruction(ssaConstInstr)

		vr := m.lowerConstant(ssaConstInstr)
		machInstr := getPendingInstr(m)
		require.Equal(t, backend.VRegIDUnreservedBegin, vr.ID())
		require.Equal(t, loadFpuConst32, machInstr.kind)
		require.Equal(t, uint64(math.Float32bits(1.1234)), machInstr.u1)

		require.Equal(t, "ldr v?0, pc+8; b 8; data.f32 1.123400", formatEmittedInstructions(m))
	})

	t.Run("TypeF64", func(t *testing.T) {
		ssaB, m := newSetup()
		ssaConstInstr := ssaB.AllocateInstruction()
		ssaConstInstr.AsF64const(-9471.2)
		ssaB.InsertInstruction(ssaConstInstr)

		vr := m.lowerConstant(ssaConstInstr)
		machInstr := getPendingInstr(m)
		require.Equal(t, backend.VRegIDUnreservedBegin, vr.ID())
		require.Equal(t, loadFpuConst64, machInstr.kind)
		require.Equal(t, math.Float64bits(-9471.2), machInstr.u1)

		require.Equal(t, "ldr v?0, pc+8; b 16; data.f64 -9471.200000", formatEmittedInstructions(m))
	})
}
