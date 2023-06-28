package arm64

import (
	"fmt"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

// LowerBranches implements backend.Machine.
func (m *machine) LowerBranches(br0, br1 *ssa.Instruction) {
	m.lowerSingleBranch(br0)
	if br1 != nil {
		m.lowerConditionalBranch(br1)
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
		m.insertAtHead(b)
	case ssa.OpcodeBrTable:
		panic("TODO: support OpcodeBrTable")
	}
}

func (m *machine) lowerConditionalBranch(b *ssa.Instruction) {
	cval, args, targetBlk := b.BranchData()
	if len(args) > 0 {
		panic("conditional branch shouldn't have args; likely a bug in critical edge splitting")
	}
	targetLabel := m.getOrAllocateSSABlockLabel(targetBlk)
	targetLabel.asBranchTarget()

	cvalDef := m.ctx.ValueDefinition(cval)
	if instr, n, condValInstr := cvalDef.Instr(); condValInstr {
		gid := b.GroupID()
		switch {
		case m.matchInstr(instr, gid, cvalDef, ssa.OpcodeIcmp):
			// cbr := m.allocateInstr()
			// cbr.asCondBr()
			// Recursively lower the conditional value.
			m.LowerInstr(instr)
		case m.matchInstr(instr, gid, cvalDef, ssa.OpcodeFcmp):
			panic("TODO")
		default:
			fmt.Println(n)
			panic("TODO")
		}
	}
	return
}

func (m *machine) matchInstr(instr *ssa.Instruction, gid ssa.InstructionGroupID, vdef *backend.SSAValueDefinition, opcode ssa.Opcode) (ok bool) {
	return instr.Opcode() == opcode && instr.GroupID() == gid && vdef.RefCount() > 1
}

// LowerInstr implements backend.Machine.
func (m *machine) LowerInstr(instr *ssa.Instruction) {
	// TODO
	m.ctx.MarkLowered(instr)
}
