package backend

import "github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"

// SSAValueDefinition represents a definition of an SSA value.
type SSAValueDefinition struct {
	Kind SSAValueDefinitionKind

	// blk is valid if Kind == SSAValueDefinitionKindBlockParam.
	BlkParamVReg VReg

	// Instr is not nil if Kind == SSAValueDefinitionKindInstr.
	Instr *ssa.Instruction
	// N is the index of the return value in the instr's return values list.
	N int
	// RefCount is the number of references to the result.
	RefCount int
}

// SSAValueDefinitionKind represents the kind of SSA value definition.
type SSAValueDefinitionKind byte

const (
	SSAValueDefinitionKindBlockParam SSAValueDefinitionKind = iota
	SSAValueDefinitionKindInstr
)
