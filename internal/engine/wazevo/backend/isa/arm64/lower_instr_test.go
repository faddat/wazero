package arm64

import (
	"strings"
	"testing"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestMachine_lowerConditionalBranch(t *testing.T) {
	cmpInSameGroupFromParams := func(
		brz bool, intCond ssa.IntegerCmpCond, floatCond ssa.FloatCmpCond,
		ctx *mockCompilationContext, builder ssa.Builder, m *machine,
	) (instr *ssa.Instruction, verify func(t *testing.T)) {
		m.StartFunction(10)
		entry := builder.CurrentBlock()
		isInt := intCond != ssa.IntegerCmpCondInvalid

		var val1, val2 ssa.Value
		if isInt {
			val1 = entry.AddParam(builder, ssa.TypeI64)
			val2 = entry.AddParam(builder, ssa.TypeI64)
			ctx.vRegMap[val1], ctx.vRegMap[val2] = regToVReg(x1), regToVReg(x2)
		} else {
			val1 = entry.AddParam(builder, ssa.TypeF64)
			val2 = entry.AddParam(builder, ssa.TypeF64)
			ctx.vRegMap[val1], ctx.vRegMap[val2] = regToVReg(v1), regToVReg(v2)
		}

		var cmpInstr *ssa.Instruction
		if isInt {
			cmpInstr = builder.AllocateInstruction()
			cmpInstr.AsIcmp(val1, val2, intCond)
			builder.InsertInstruction(cmpInstr)
		} else {
			cmpInstr = builder.AllocateInstruction()
			cmpInstr.AsFcmp(val1, val2, floatCond)
			builder.InsertInstruction(cmpInstr)
		}

		cmpVal := cmpInstr.Return()
		ctx.vRegMap[cmpVal] = 3

		ctx.definitions[val1] = &backend.SSAValueDefinition{BlkParamVReg: ctx.vRegMap[val1], BlockParamValue: val1}
		ctx.definitions[val2] = &backend.SSAValueDefinition{BlkParamVReg: ctx.vRegMap[val2], BlockParamValue: val2}
		ctx.definitions[cmpVal] = &backend.SSAValueDefinition{Instr: cmpInstr}
		b := builder.AllocateInstruction()
		if brz {
			b.AsBrz(cmpVal, nil, builder.AllocateBasicBlock())
		} else {
			b.AsBrnz(cmpVal, nil, builder.AllocateBasicBlock())
		}
		builder.InsertInstruction(b)
		return b, func(t *testing.T) {
			_, ok := ctx.lowered[cmpInstr]
			require.True(t, ok)
		}
	}

	icmpInSameGroupFromParamAndImm12 := func(brz bool, ctx *mockCompilationContext, builder ssa.Builder, m *machine) (instr *ssa.Instruction, verify func(t *testing.T)) {
		m.StartFunction(10)
		entry := builder.CurrentBlock()
		v1 := entry.AddParam(builder, ssa.TypeI32)

		iconst := builder.AllocateInstruction()
		iconst.AsIconst32(0x4d2)
		builder.InsertInstruction(iconst)
		v2 := iconst.Return()

		// Constant can be referenced from different groups because we inline it.
		builder.SetCurrentBlock(builder.AllocateBasicBlock())

		icmp := builder.AllocateInstruction()
		icmp.AsIcmp(v1, v2, ssa.IntegerCmpCondEqual)
		builder.InsertInstruction(icmp)
		icmpVal := icmp.Return()
		ctx.definitions[v1] = &backend.SSAValueDefinition{BlkParamVReg: 1, BlockParamValue: v1}
		ctx.definitions[v2] = &backend.SSAValueDefinition{Instr: iconst}
		ctx.definitions[icmpVal] = &backend.SSAValueDefinition{Instr: icmp}
		ctx.vRegMap[v1], ctx.vRegMap[v2], ctx.vRegMap[icmpVal] = 1, 2, 3
		b := builder.AllocateInstruction()
		if brz {
			b.AsBrz(icmpVal, nil, builder.AllocateBasicBlock())
		} else {
			b.AsBrnz(icmpVal, nil, builder.AllocateBasicBlock())
		}
		builder.InsertInstruction(b)
		return b, func(t *testing.T) {
			_, ok := ctx.lowered[icmp]
			require.True(t, ok)
		}
	}

	for _, tc := range []struct {
		name         string
		setup        func(*mockCompilationContext, ssa.Builder, *machine) (instr *ssa.Instruction, verify func(t *testing.T))
		instructions []string
	}{
		{
			name: "icmp in different group",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (instr *ssa.Instruction, verify func(t *testing.T)) {
				m.StartFunction(10)
				entry := builder.CurrentBlock()
				v1, v2 := entry.AddParam(builder, ssa.TypeI64), entry.AddParam(builder, ssa.TypeI64)

				icmp := builder.AllocateInstruction()
				icmp.AsIcmp(v1, v2, ssa.IntegerCmpCondEqual)
				builder.InsertInstruction(icmp)
				icmpVal := icmp.Return()
				ctx.definitions[icmpVal] = &backend.SSAValueDefinition{Instr: icmp}
				ctx.vRegMap[v1], ctx.vRegMap[v2], ctx.vRegMap[icmpVal] = 1, 2, 3

				brz := builder.AllocateInstruction()
				brz.AsBrz(icmpVal, nil, builder.AllocateBasicBlock())
				builder.InsertInstruction(brz)

				// Indicate that currently compiling in the different group.
				m.setCurrentInstructionGroupID(1000)
				return brz, func(t *testing.T) {
					_, ok := ctx.lowered[icmp]
					require.False(t, ok)
				}
			},
			instructions: []string{"cbz r3?, L1"},
		},
		{
			name: "brz / icmp in the same group / params",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (instr *ssa.Instruction, verify func(t *testing.T)) {
				return cmpInSameGroupFromParams(true, ssa.IntegerCmpCondUnsignedGreaterThan, ssa.FloatCmpCondInvalid, ctx, builder, m)
			},
			instructions: []string{
				"subs xzr, x1, x2",
				"b.ls L1",
			},
		},
		{
			name: "brnz / icmp in the same group / params",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (instr *ssa.Instruction, verify func(t *testing.T)) {
				return cmpInSameGroupFromParams(false, ssa.IntegerCmpCondEqual, ssa.FloatCmpCondInvalid, ctx, builder, m)
			},
			instructions: []string{
				"subs xzr, x1, x2",
				"b.eq L1",
			},
		},
		{
			name: "brz / fcmp in the same group / params",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (instr *ssa.Instruction, verify func(t *testing.T)) {
				return cmpInSameGroupFromParams(true, ssa.IntegerCmpCondInvalid, ssa.FloatCmpCondEqual, ctx, builder, m)
			},
			instructions: []string{
				"fcmp w1, w2",
				"b.ne L1",
			},
		},
		{
			name: "brnz / fcmp in the same group / params",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (instr *ssa.Instruction, verify func(t *testing.T)) {
				return cmpInSameGroupFromParams(false, ssa.IntegerCmpCondInvalid, ssa.FloatCmpCondGreaterThan, ctx, builder, m)
			},
			instructions: []string{
				"fcmp w1, w2",
				"b.gt L1",
			},
		},
		{
			name: "brz / icmp in the same group / params",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (instr *ssa.Instruction, verify func(t *testing.T)) {
				return icmpInSameGroupFromParamAndImm12(true, ctx, builder, m)
			},
			instructions: []string{
				"subs wzr, r1?, #0x4d2",
				"b.ne L1",
			},
		},
		{
			name: "brz / icmp in the same group / params",
			setup: func(ctx *mockCompilationContext, builder ssa.Builder, m *machine) (instr *ssa.Instruction, verify func(t *testing.T)) {
				return icmpInSameGroupFromParamAndImm12(false, ctx, builder, m)
			},
			instructions: []string{
				"subs wzr, r1?, #0x4d2",
				"b.eq L1",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, b, m := newSetupWithMockContext()
			instr, verify := tc.setup(ctx, b, m)
			m.lowerConditionalBranch(instr)
			verify(t)
			require.Equal(t, strings.Join(tc.instructions, "\n"),
				formatEmittedInstructions(m))
		})
	}
}
