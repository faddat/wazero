package arm64

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestMachine_lowerConstant(t *testing.T) {
	t.Run("TypeF32", func(t *testing.T) {
		ssaB, m := newSetup()
		ssaConstInstr := ssaB.AllocateInstruction()
		ssaConstInstr.AsF32const(1.1234)
		ssaB.InsertInstruction(ssaConstInstr)

		vr := m.lowerConstant(ssaConstInstr)
		machInstr := getPendingInstr(m)
		require.Equal(t, backend.VRegID(0), vr.ID())
		require.Equal(t, loadFpuConst32, machInstr.kind)
		require.Equal(t, uint64(math.Float32bits(1.1234)), machInstr.u1)

		require.Equal(t, "ldr r0?, pc+8; b 8; data.f32 1.123400", formatEmittedInstructions(m))
	})

	t.Run("TypeF64", func(t *testing.T) {
		ssaB, m := newSetup()
		ssaConstInstr := ssaB.AllocateInstruction()
		ssaConstInstr.AsF64const(-9471.2)
		ssaB.InsertInstruction(ssaConstInstr)

		vr := m.lowerConstant(ssaConstInstr)
		machInstr := getPendingInstr(m)
		require.Equal(t, backend.VRegID(0), vr.ID())
		require.Equal(t, loadFpuConst64, machInstr.kind)
		require.Equal(t, math.Float64bits(-9471.2), machInstr.u1)

		require.Equal(t, "ldr r0?, pc+8; b 16; data.f64 -9471.200000", formatEmittedInstructions(m))
	})
}

func TestMachine_lowerConstantI32(t *testing.T) {
	for _, tc := range []struct {
		val uint32
		exp []string
	}{
		{val: 0, exp: []string{"movz w0, #0x0, LSL 0"}},
		{val: 0xffff, exp: []string{"movz w0, #0xffff, LSL 0"}},
		{val: 0xffff_0000, exp: []string{"movz w0, #0xffff, LSL 16"}},
		{val: 0xffff_fffe, exp: []string{"movn w0, #0x1, LSL 0"}},
		{val: 0x2, exp: []string{"orr w0 wzr, r0?, #0x2"}},
		{val: 0x80000001, exp: []string{"orr w0 wzr, r0?, #0x80000001"}},
		{val: 0xf00000f, exp: []string{
			"movz w0, #0xf, LSL 0",
			"movk w0, #0xf00, LSL 16",
		}},
	} {
		tc := tc
		t.Run(fmt.Sprintf("%#x", tc.val), func(t *testing.T) {
			_, m := newSetup()
			m.lowerConstantI32(regToVReg(x0), int32(tc.val))
			exp := strings.Join(tc.exp, "\n")
			require.Equal(t, exp, formatEmittedInstructions(m))
		})
	}
}
