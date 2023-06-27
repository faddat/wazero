package arm64

import "github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"

// LowerBranches implements backend.Machine.
func (m *machine) LowerBranches(br0, br1 *ssa.Instruction) {
	if br1 == nil {

	}
}

// LowerInstr implements backend.Machine.
func (m *machine) LowerInstr(instr *ssa.Instruction) {
	switch instr.Opcode() {
	default:
		panic("not implemented")
	}
}
