package arm64

import (
	"strings"
	"testing"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
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

func TestMachine_getOperand_SR_NR(t *testing.T) {
	ishlWithConstAmount := func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
		blk := builder.CurrentBlock()
		// (p1+p2) << amount
		p1 := blk.AddParam(builder, ssa.TypeI64)
		p2 := blk.AddParam(builder, ssa.TypeI64)
		add := builder.AllocateInstruction()
		add.AsIadd(p1, p2)
		builder.InsertInstruction(add)
		addResult := add.Return()

		amount := builder.AllocateInstruction()
		amount.AsIconst32(14)
		builder.InsertInstruction(amount)

		amountVal := amount.Return()

		ishl := builder.AllocateInstruction()
		ishl.AsIshl(addResult, amountVal)
		builder.InsertInstruction(ishl)

		ctx.definitions[p1] = &backend.SSAValueDefinition{BlkParamVReg: backend.VReg(1), BlockParamValue: p1}
		ctx.definitions[p2] = &backend.SSAValueDefinition{BlkParamVReg: backend.VReg(2), BlockParamValue: p2}
		ctx.definitions[addResult] = &backend.SSAValueDefinition{Instr: add, N: 0}
		ctx.definitions[amountVal] = &backend.SSAValueDefinition{Instr: amount, N: 0}

		ctx.vRegMap[addResult] = backend.VReg(1234)
		ctx.vRegMap[ishl.Return()] = backend.VReg(10)
		def = &backend.SSAValueDefinition{Instr: ishl, N: 0}
		mode = extModeNone
		return
	}

	for _, tc := range []struct {
		name         string
		setup        func(*mockCompilationContext, ssa.Builder, *machine) (def *backend.SSAValueDefinition, mode extMode)
		exp          operand
		instructions []string
	}{
		{
			name: "block param",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
				blk := builder.CurrentBlock()
				v := blk.AddParam(builder, ssa.TypeI64)
				def = &backend.SSAValueDefinition{BlkParamVReg: regToVReg(x4), BlockParamValue: v}
				return def, extModeNone
			},
			exp: operandNR(regToVReg(x4)),
		},
		{
			name: "ishl but not const amount",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
				blk := builder.CurrentBlock()
				// (p1+p2) << p3
				p1 := blk.AddParam(builder, ssa.TypeI64)
				p2 := blk.AddParam(builder, ssa.TypeI64)
				p3 := blk.AddParam(builder, ssa.TypeI64)
				add := builder.AllocateInstruction()
				add.AsIadd(p1, p2)
				builder.InsertInstruction(add)
				addResult := add.Return()

				ishl := builder.AllocateInstruction()
				ishl.AsIshl(addResult, p3)
				builder.InsertInstruction(ishl)

				ctx.definitions[p1] = &backend.SSAValueDefinition{BlkParamVReg: backend.VReg(1), BlockParamValue: p1}
				ctx.definitions[p2] = &backend.SSAValueDefinition{BlkParamVReg: backend.VReg(2), BlockParamValue: p2}
				ctx.definitions[p3] = &backend.SSAValueDefinition{BlkParamVReg: backend.VReg(3), BlockParamValue: p3}
				ctx.definitions[addResult] = &backend.SSAValueDefinition{Instr: add, N: 0}

				ctx.vRegMap[ishl.Return()] = backend.VReg(10)
				def = &backend.SSAValueDefinition{Instr: ishl, N: 0}
				return def, extModeNone
			},
			exp: operandNR(backend.VReg(10)),
		},
		{
			name:  "ishl with const amount",
			setup: ishlWithConstAmount,
			exp:   operandSR(backend.VReg(1234), 14, shiftOpLSL),
		},
		{
			name: "ishl with const amount but group id is different",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
				def, mode = ishlWithConstAmount(ctx, builder, m)
				m.currentGID = 1230
				return
			},
			exp: operandNR(backend.VReg(10)),
		},
		{
			name: "ishl with const amount but ref count is larger than 1",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
				def, mode = ishlWithConstAmount(ctx, builder, m)
				def.RefCount = 10
				return
			},
			exp: operandNR(backend.VReg(10)),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, b, m := newSetupWithMockContext()
			def, mode := tc.setup(ctx, b, m)
			actual := m.getOperand_SR_NR(def, mode)
			require.Equal(t, tc.exp, actual)
			require.Equal(t, strings.Join(tc.instructions, "\n"), formatEmittedInstructions(m))
		})
	}
}

func TestMachine_getOperand_Imm12_ER_SR_NR(t *testing.T) {
	for _, tc := range []struct {
		name         string
		setup        func(*mockCompilationContext, ssa.Builder, *machine) (def *backend.SSAValueDefinition, mode extMode)
		exp          operand
		instructions []string
	}{
		{
			name: "block param",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
				blk := builder.CurrentBlock()
				v := blk.AddParam(builder, ssa.TypeI64)
				def = &backend.SSAValueDefinition{BlkParamVReg: regToVReg(x4), BlockParamValue: v}
				return def, extModeZeroExtend64
			},
			exp: operandNR(regToVReg(x4)),
		},
		{
			name: "const imm 12 no shift",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
				cinst := builder.AllocateInstruction()
				cinst.AsIconst32(0xfff)
				builder.InsertInstruction(cinst)
				def = &backend.SSAValueDefinition{Instr: cinst, N: 0}
				m.currentGID = 0xff // const can be merged anytime, regardless of the group id.
				return
			},
			exp: operandImm12(0xfff, 0),
		},
		{
			name: "const imm 12 with shift bit",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (def *backend.SSAValueDefinition, mode extMode) {
				cinst := builder.AllocateInstruction()
				cinst.AsIconst32(0xabc_000)
				builder.InsertInstruction(cinst)
				def = &backend.SSAValueDefinition{Instr: cinst, N: 0}
				m.currentGID = 0xff // const can be merged anytime, regardless of the group id.
				return
			},
			exp: operandImm12(0xabc, 1),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, b, m := newSetupWithMockContext()
			def, mode := tc.setup(ctx, b, m)
			actual := m.getOperand_Imm12_ER_SR_NR(def, mode)
			require.Equal(t, tc.exp, actual)
			require.Equal(t, strings.Join(tc.instructions, "\n"), formatEmittedInstructions(m))
		})
	}
}
