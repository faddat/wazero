package arm64

// This file contains the logic to "find and determine operands" for instructions.
// In order to finalize the form of an operand, we might end up merging
// the source instructions into one whenever possible.

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

// operandNR encodes the given VReg as an operand of operandKindNR.
func operandNR(r backend.VReg) operand {
	return operand{kind: operandKindNR, data: uint64(r)}
}

// nr decodes the underlying VReg assuming the operand is of operandKindNR.
func (o operand) nr() backend.VReg {
	return backend.VReg(o.data)
}

// operandImm12 encodes the given imm12 as an operand of operandKindImm12.
func operandImm12(imm12 uint16, shiftBit byte) operand {
	return operand{kind: operandKindImm12, data: uint64(imm12) | uint64(shiftBit)<<32}
}

// imm12 decodes the underlying imm12 data assuming the operand is of operandKindImm12.
func (o operand) imm12() (v uint16, shiftBit byte) {
	return uint16(o.data), byte(o.data >> 32 & 0b1)
}

// ensureValueNR returns an operand of either operandKindER, operandKindSR, or operandKindNR from the given value (defined by `def).
//
// `mode` is used to extend the operand if the bit length is smaller than mode.bits().
// If the operand can be expressed as operandKindImm12, `mode` is ignored.
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

// ensureValueNR returns an operand of either operandKindER, operandKindSR, or operandKindNR from the given value (defined by `def).
//
// `mode` is used to extend the operand if the bit length is smaller than mode.bits().
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

// ensureValueNR returns an operand of either operandKindSR or operandKindNR from the given value (defined by `def).
//
// `mode` is used to extend the operand if the bit length is smaller than mode.bits().
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

// ensureValueNR returns an operand of operandKindNR from the given value (defined by `def).
//
// `mode` is used to extend the operand if the bit length is smaller than mode.bits().
func (m *machine) getOperand_NR(def *backend.SSAValueDefinition, mode extMode) (op operand) {
	var v backend.VReg
	if def.IsFromBlockParam() {
		v = def.BlkParamVReg
	} else {
		instr := def.Instr
		if instr.Constant() {
			// We inline all the constant instructions so that we could reduce the register usage.
			v = m.lowerConstant(instr)
		} else {
			r1, rs := instr.Returns()
			if n := def.N; n == 0 {
				v = m.ctx.VRegOf(r1)
			} else {
				v = m.ctx.VRegOf(rs[n-1])
			}
		}
	}

	switch inBits := def.SSAValue().Type().Bits(); {
	case mode == extModeNone:
	case inBits == 32 && (mode == extModeZeroExtend32 || mode == extModeSignExtend32):
	case inBits == 32 && mode == extModeZeroExtend64:
		ext := m.allocateInstr()
		ext.asExtend(v, v, 32, 64, false)
		m.insert(ext)
	case inBits == 32 && mode == extModeSignExtend64:
		ext := m.allocateInstr()
		ext.asExtend(v, v, 32, 64, true)
		m.insert(ext)
	case inBits == 64 && (mode == extModeZeroExtend64 || mode == extModeSignExtend64):
	}
	return operandNR(v)
}

func asImm12(val uint64) (v uint16, shiftBit byte, ok bool) {
	const mask1, mask2 uint64 = 0xfff, 0xfff_000
	if val&^mask1 == 0 {
		return uint16(val), 0, true
	} else if val&^mask2 == 0 {
		return uint16(val >> 12), 1, true
	} else {
		return 0, 0, false
	}
}
