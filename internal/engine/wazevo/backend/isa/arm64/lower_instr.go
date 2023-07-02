package arm64

// Files prefixed as lower** do the instruction selections, meaning that lowering SSA level instructions
// into machine specific instructions.
//
// Importantly, what the lower** functions does includes tree-matching; find the pattern from the given instruction tree,
// and merge the multiple instructions if possible. It can be considered as "N:1" instruction selection.

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

// LowerBranches implements backend.Machine.
func (m *machine) LowerBranches(br0, br1 *ssa.Instruction) {
	m.setCurrentInstructionGroupID(br0.GroupID())
	m.lowerSingleBranch(br0)
	m.flushPendingInstructions()
	if br1 != nil {
		m.setCurrentInstructionGroupID(br1.GroupID())
		m.lowerConditionalBranch(br1)
		m.flushPendingInstructions()
	}
}

func (m *machine) lowerSingleBranch(br *ssa.Instruction) {
	_, args, targetBlk := br.BranchData()
	if len(args) > 0 {
		panic("TODO: support block args: insert phi moves")
	}

	switch br.Opcode() {
	case ssa.OpcodeJump:
		if br.IsFallthroughJump() {
			return
		}
		b := m.allocateInstr()
		targetLabel := m.getOrAllocateSSABlockLabel(targetBlk)
		b.asBr(targetLabel.asBranchTarget())
		m.insert(b)
	case ssa.OpcodeBrTable:
		panic("TODO: support OpcodeBrTable")
	}
}

func (m *machine) lowerConditionalBranch(b *ssa.Instruction) {
	cval, args, targetBlk := b.BranchData()
	if len(args) > 0 {
		panic("conditional branch shouldn't have args; likely a bug in critical edge splitting")
	}

	target := m.getOrAllocateSSABlockLabel(targetBlk).asBranchTarget()
	cvalDef := m.ctx.ValueDefinition(cval)

	switch {
	case m.matchInstr(cvalDef, ssa.OpcodeIcmp): // This case, we can use the ALU flag set by SUBS instruction.
		cvalInstr := cvalDef.Instr
		x, y, c := cvalInstr.IcmpData()
		cc, signed := condFlagFromSSAIntegerCmpCond(c), c.Signed()
		if b.Opcode() == ssa.OpcodeBrz {
			cc = cc.invert()
		}

		if x.Type() != y.Type() {
			panic("TODO(maybe): support icmp with different types")
		}

		extMod := extModeOf(x.Type(), signed)
		bits := x.Type().Bits()

		cbr := m.allocateInstr()
		cbr.asCondBr(cc.asCond(), target)

		// First operand must be in pure register form.
		rn := m.getOperand_NR(m.ctx.ValueDefinition(x), extMod)
		// Second operand can be in any of Imm12, ER, SR, or NR form supported by the SUBS instructions.
		rm := m.getOperand_Imm12_ER_SR_NR(m.ctx.ValueDefinition(y), extMod)

		alu := m.allocateInstr()
		// subs zr, rn, rm
		alu.asALU(
			aluOpSubS,
			// We don't need the result, just need to set flags.
			operandNR(xzrVReg),
			rn,
			rm,
			bits == 64,
		)
		m.insert2(alu, cbr)
		m.ctx.MarkLowered(cvalDef.Instr)
	case m.matchInstr(cvalDef, ssa.OpcodeFcmp): // This case we can use the Fpu flag directly.
		cvalInstr := cvalDef.Instr
		x, y, c := cvalInstr.FcmpData()
		cc := condFlagFromSSAFloatCmpCond(c)
		if b.Opcode() == ssa.OpcodeBrz {
			cc = cc.invert()
		}

		if x.Type() != y.Type() {
			panic("TODO(maybe): support icmp with different types")
		}

		rn := m.getOperand_NR(m.ctx.ValueDefinition(x), extModeNone)
		rm := m.getOperand_NR(m.ctx.ValueDefinition(y), extModeNone)
		cmp := m.allocateInstr()
		cmp.asFpuCmp(rn, rm, x.Type().Bits() == 64)
		cbr := m.allocateInstr()
		cbr.asCondBr(cc.asCond(), target)
		m.insert2(cmp, cbr)
		m.ctx.MarkLowered(cvalDef.Instr)
	default:
		rn := m.getOperand_NR(cvalDef, extModeNone)
		var c cond
		if b.Opcode() == ssa.OpcodeBrz {
			c = registerAsRegZeroCond(rn.nr())
		} else {
			c = registerAsRegNonZeroCond(rn.nr())
		}
		cbr := m.allocateInstr()
		cbr.asCondBr(c, target)
		m.insert(cbr)
	}
}

// LowerInstr implements backend.Machine.
func (m *machine) LowerInstr(instr *ssa.Instruction) {
	op := instr.Opcode()
	switch op {
	case ssa.OpcodeBrz, ssa.OpcodeBrnz, ssa.OpcodeJump, ssa.OpcodeBrTable:
		return
	}

	m.setCurrentInstructionGroupID(instr.GroupID())

	switch instr.Opcode() {
	case ssa.OpcodeBrz, ssa.OpcodeBrnz, ssa.OpcodeJump, ssa.OpcodeBrTable:
		panic("BUG: branching instructions are handled by LowerBranches")
	}

	m.flushPendingInstructions()
}

// matchInstr returns true if the given definition is from the given opcode and group ID, and has a refcount of 1.
// That means, the instruction can be merged/swapped within the current instruction group.
func (m *machine) matchInstr(def *backend.SSAValueDefinition, opcode ssa.Opcode) bool {
	instr := def.Instr
	return def.IsFromInstr() &&
		instr.Opcode() == opcode &&
		instr.GroupID() == m.currentGID &&
		def.RefCount < 2
}
