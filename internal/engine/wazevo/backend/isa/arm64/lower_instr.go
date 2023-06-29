package arm64

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

	targetLabel := m.getOrAllocateSSABlockLabel(targetBlk)

	switch br.Opcode() {
	case ssa.OpcodeJump:
		if br.IsFallthroughJump() {
			return
		}
		b := m.allocateInstr()
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
	case m.matchInstr(cvalDef, ssa.OpcodeIcmp):
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
		// subs zr, rn, rm!
		alu.asALU(
			pickByBits(bits, subS32, subS64),
			// We don't need the result, just need to set flags.
			operandNR(pickByBits(bits, wzrVReg, xzrVReg)),
			rn,
			rm,
		)
		m.insert2(alu, cbr)
	case m.matchInstr(cvalDef, ssa.OpcodeFcmp):
		// TODO: this should be able to share the code in the above case.
		panic("TODO")
	default:
		panic("TODO")
	}
}

func pickByBits[T any](bits byte, v32, v64 T) T {
	if bits == 32 {
		return v32
	}
	return v64
}

// LowerInstr implements backend.Machine.
func (m *machine) LowerInstr(instr *ssa.Instruction) {
	m.setCurrentInstructionGroupID(instr.GroupID())

	// TODO: lowering logic.

	m.flushPendingInstructions()
}

func (m *machine) matchInstr(def *backend.SSAValueDefinition, opcode ssa.Opcode) bool {
	instr := def.Instr
	return def.IsFromInstr() &&
		instr.Opcode() == opcode &&
		instr.GroupID() == m.currentGID &&
		def.RefCount < 2
}
