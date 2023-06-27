package backend

import "github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"

// SSAValueDefinition represents a definition of an SSA value.
type SSAValueDefinition struct {
	refCount int

	// isBlockParam indicates whether this value is a block parameter.
	isBlockParam bool

	// blk is not nil if this value is a block parameter, i.e. isBlockParam is true.
	blk ssa.BasicBlock
	// instr is not nil if this value is defined by an instruction, i.e. isBlockParam is false.
	instr *ssa.Instruction
	// n is the index of the parameter in the blk's parameter list if isBlockParam is true.
	// Otherwise, n is the index of the return value in the instr's return values list.
	n int
}

// Param returns the block and the index of the parameter if this value is a block parameter.
func (s *SSAValueDefinition) Param() (blk ssa.BasicBlock, n int, ok bool) {
	if s.isBlockParam {
		blk = s.blk
		n = s.n
		ok = true
	}
	return
}

// Instr returns the instruction and the index of the return value if this value is defined by an instruction.
func (s *SSAValueDefinition) Instr() (instr *ssa.Instruction, n int, ok bool) {
	if !s.isBlockParam {
		instr = s.instr
		n = s.n
		ok = true
	}
	return
}

// RefCount returns the reference count of the ssa.Value.
func (s *SSAValueDefinition) RefCount() int {
	return s.refCount
}

// reset resets this SSAValueDefinition so that it can be reused in the next compilation.
func (s *SSAValueDefinition) reset() {
	s.blk = nil
	s.instr = nil
	s.n = -1
}
