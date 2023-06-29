package arm64

// This file contains the logic to "find operands" for instructions.

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

type (
	// operand represents an operand of an instruction whose type is determined by the kind.
	operand struct {
		kind operandKind
		data uint64
	}
	operandKind byte
)

// Here's the list of operand kinds. We use the abbreviation of the kind name not only for these consts,
// but also names of functions which return the operand of the kind.
const (
	// operandKindNR represents "NormalRegister" (NR). This is literally the register without any special operation unlike others.
	operandKindNR operandKind = iota
	// operandKindSR represents "Shifted Register" (SR). This is a register which is shifted by a constant.
	// Some of the arm64 instructions can take this kind of operand.
	operandKindSR
	// operandKindER represents "Extended Register (ER). This is a register which is sign/zero-extended to a larger size.
	// Some of the arm64 instructions can take this kind of operand.
	operandKindER
	// operandKindImm12 represents "Immediate 12" (Imm12). This is a 12-bit immediate value which can be either shifted or not.
	// See asImm12 function for detail.
	operandKindImm12
)

func operandNR(r backend.VReg) operand {
	return operand{kind: operandKindNR, data: uint64(r)}
}

func (o operand) NR() backend.VReg {
	return backend.VReg(o.data)
}

func operandImm12(imm12 uint16, shiftBit byte) operand {
	return operand{kind: operandKindImm12, data: uint64(imm12) | uint64(shiftBit)<<32}
}

func (o operand) imm12NeedShift() bool {
	return o.data>>32 != 0
}

func (m *machine) getOperand_Imm12_ER_SR_NR(def *backend.SSAValueDefinition, mode extMode) (op operand) {
	if def.IsFromBlockParam() {
		return operandNR(def.BlkParamVReg)
	}

	instr := def.Instr
	if instr.Opcode() == ssa.OpcodeIconst {
		if imm12, shift, ok := asImm12(instr.ConstantVal()); ok {
			return operandImm12(imm12, shift)
		}
	}
	return m.getOperand_ER_SR_NR(def, mode)
}

func (m *machine) getOperand_ER_SR_NR(def *backend.SSAValueDefinition, mode extMode) (op operand) {
	if def.IsFromInstr() {
		return operandNR(def.BlkParamVReg)
	}

	switch {
	case m.matchInstr(def, ssa.OpcodeSextend):
		panic("TODO")
	case m.matchInstr(def, ssa.OpcodeUextend):
		panic("TODO")
	}

	return m.getOperand_SR_NR(def, mode)
}

func (m *machine) getOperand_SR_NR(def *backend.SSAValueDefinition, mode extMode) (op operand) {
	if def.IsFromBlockParam() {
		return operandNR(def.BlkParamVReg)
	}

	if m.matchInstr(def, ssa.OpcodeIshl) {
		// TODO:
		return
	}
	return m.getOperand_NR(def, mode)
}

// ensureValueNR ensures that the given value is a normal register.
//
// This doesn't merge any instruction, just check if it is a constant instruction, and inline it if so.
// Otherwise, use the default backend.VReg.
func (m *machine) getOperand_NR(def *backend.SSAValueDefinition, mode extMode) (op operand) {
	var v backend.VReg

	if def.IsFromBlockParam() {
		v = def.BlkParamVReg
	} else {
		instr := def.Instr
		if instr.Constant() {
			// We inline all the constant instructions so that we could reduce the register usage.
			v = m.emitConstant(instr)
		} else {
			r1, rs := instr.Returns()
			if n := def.N; n == 0 {
				v = m.ctx.VRegOf(r1)
			} else {
				v = m.ctx.VRegOf(rs[n-1])
			}
		}
	}

	switch mode {
	case extModeNone:
	case extModeZeroExtend64:
		panic("TODO")
	case extModeSignExtend64:
		panic("TODO")
	}
	return operandNR(v)
}

func (m *machine) emitConstant(instr *ssa.Instruction) (v backend.VReg) {
	panic("TODO")
}

func asImm12(val uint64) (v uint16, shiftBit byte, ok bool) {
	if val < 0xfff {
		return uint16(v), 1, true
	} else if val < 0xfff_000 && (val&0xfff == 0) {
		return uint16(v >> 12), 1, true
	} else {
		return 0, 0, false
	}
}
