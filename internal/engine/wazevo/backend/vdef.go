package backend

import "github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"

// SSAValueDefinition represents a definition of an SSA value.
type SSAValueDefinition struct {
	// blk is valid if Instr == nil
	BlkParamVReg VReg

	// Instr is not nil if Kind == SSAValueDefinitionKindInstr.
	Instr *ssa.Instruction
	// N is the index of the return value in the instr's return values list.
	N int
	// RefCount is the number of references to the result.
	RefCount int
}

func (d *SSAValueDefinition) IsFromInstr() bool {
	return d.Instr != nil
}

func (d *SSAValueDefinition) IsFromBlockParam() bool {
	return d.Instr == nil
}
