package arm64

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func Test_asImm12(t *testing.T) {
	v, shift, ok := asImm12(0xfff)
	require.True(t, ok)
	require.True(t, shift == 0)
	require.Equal(t, uint16(0xfff), v)

	_, _, ok = asImm12(0xfff << 1)
	require.False(t, ok)

	v, shift, ok = asImm12(0xabc << 12)
	require.True(t, ok)
	require.True(t, shift == 1)
	require.Equal(t, uint16(0xabc), v)

	_, _, ok = asImm12(0xabc<<12 | 0b1)
	require.False(t, ok)
}

func TestMachine_getOperand_NR(t *testing.T) {
	for _, tc := range []struct {
		name         string
		setup        func(*mockCompilationContext, ssa.Builder, *machine) (def *backend.SSAValueDefinition, mode extMode)
		exp          operand
		instructions []string
	}{
		{
			name: "block param - no extend",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
				blk := builder.CurrentBlock()
				v := blk.AddParam(builder, ssa.TypeI64)
				def = &backend.SSAValueDefinition{BlkParamVReg: regToVReg(x4), BlockParamValue: v}
				return def, extModeZeroExtend64
			},
			exp: operandNR(regToVReg(x4)),
		},
		{
			name: "block param - zero extend",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
				blk := builder.CurrentBlock()
				v := blk.AddParam(builder, ssa.TypeI32)
				def = &backend.SSAValueDefinition{BlkParamVReg: regToVReg(x4), BlockParamValue: v}
				return def, extModeZeroExtend64
			},
			exp:          operandNR(regToVReg(x4)),
			instructions: []string{"mov w4, w4"},
		},
		{
			name: "block param - sign extend",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
				blk := builder.CurrentBlock()
				v := blk.AddParam(builder, ssa.TypeI32)
				def = &backend.SSAValueDefinition{BlkParamVReg: regToVReg(x4), BlockParamValue: v}
				return def, extModeSignExtend64
			},
			exp:          operandNR(regToVReg(x4)),
			instructions: []string{"sxtw x4, w4"},
		},
		{
			name: "const instr",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
				instr := builder.AllocateInstruction()
				instr.AsIconst32(0xf00000f)
				builder.InsertInstruction(instr)
				def = &backend.SSAValueDefinition{Instr: instr, N: 0}
				ctx.vRegCounter = 99
				return
			},
			exp: operandNR(backend.VReg(100)),
			instructions: []string{
				"movz r100?, #0xf, LSL 0",
				"movk r100?, #0xf00, LSL 16",
			},
		},
		{
			name: "non const instr",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
				c := builder.AllocateInstruction()
				sig := &ssa.Signature{Results: []ssa.Type{ssa.TypeI64, ssa.TypeF64, ssa.TypeF64}}
				builder.DeclareSignature(sig)
				c.AsCall(ssa.FuncRef(0), sig, nil)
				builder.InsertInstruction(c)
				_, rs := c.Returns()
				ctx.vRegMap[rs[1]] = backend.VReg(50)
				def = &backend.SSAValueDefinition{Instr: c, N: 2}
				return
			},
			exp: operandNR(backend.VReg(50)),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, b, m := newSetupWithMockContext()
			def, mode := tc.setup(ctx, b, m)
			actual := m.getOperand_NR(def, mode)
			require.Equal(t, tc.exp, actual)
			require.Equal(t, strings.Join(tc.instructions, "\n"), formatEmittedInstructions(m))
		})
	}
}
