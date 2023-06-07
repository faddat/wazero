package ssa

import (
	"fmt"
	"math"
	"strings"
)

// Opcode represents a SSA instruction.
type Opcode uint32

// Instruction represents an instruction whose opcode is specified by
// Opcode. Since Go doesn't have union type, we use this flattened type
// for all instructions, and therefore each field has different meaning
// depending on Opcode.
type Instruction struct {
	opcode     Opcode
	u64        uint64
	vs         []Value
	typ        Type
	blk1, blk2 BasicBlock
	prev, next *Instruction
	srcPos     uint64

	rValue  Value
	rValues []Value
}

// Returns Value(s) produced by this instruction if any.
// The `first` is the first return value, and `rest` is the rest of the values.
func (i *Instruction) Returns() (first Value, rest []Value) {
	return i.rValue, i.rValues
}

// Next returns the next instruction laid out next to itself.
func (i *Instruction) Next() *Instruction {
	return i.next
}

// Prev returns the previous instruction laid out prior to itself.
func (i *Instruction) Prev() *Instruction {
	return i.prev
}

// SetSourcePos sets the opaque source position of this instruction.
func (i *Instruction) SetSourcePos(p uint64) {
	i.srcPos = p
}

// SourcePos returns the opaque source position of this instruction set by SetSourcePos.
func (i *Instruction) SourcePos() (p uint64) {
	return i.srcPos
}

var instructionFormats = [opcodeEnd]func(instruction *Instruction) string{}

// Followings match the generated code from https://github.com/bytecodealliance/wasmtime/blob/v9.0.3/cranelift/codegen/meta/src/shared/instructions.rs
// TODO: complete opcode comments.
// TODO: there should be unnecessary opcodes.
const (
	// OpcodeJump takes the list of args to the `block` and unconditionally jumps to it.
	OpcodeJump Opcode = 1 + iota

	// OpcodeBrz ...
	// `brz c, block, args`.
	// Type inferred from `c`.
	OpcodeBrz

	// OpcodeBrnz ...
	// `brnz c, block, args`.
	// Type inferred from `c`.
	OpcodeBrnz

	// OpcodeBrTable ...
	// `br_table x, block, JT`.
	OpcodeBrTable

	// OpcodeDebugtrap ...
	// `debugtrap`.
	OpcodeDebugtrap

	// OpcodeTrap ...
	// `trap code`.
	OpcodeTrap

	// OpcodeTrapz ...
	// `trapz c, code`.
	// Type inferred from `c`.
	OpcodeTrapz

	// OpcodeResumableTrap ...
	// `resumable_trap code`.
	OpcodeResumableTrap

	// OpcodeTrapnz ...
	// `trapnz c, code`.
	// Type inferred from `c`.
	OpcodeTrapnz

	// OpcodeResumableTrapnz ...
	// `resumable_trapnz c, code`.
	// Type inferred from `c`.
	OpcodeResumableTrapnz

	// OpcodeReturn ...
	// `return rvals`.
	OpcodeReturn

	// OpcodeCall ...
	// `rvals = call FN, args`.
	OpcodeCall

	// OpcodeCallIndirect ...
	// `rvals = call_indirect SIG, callee, args`.
	// Type inferred from `callee`.
	OpcodeCallIndirect

	// OpcodeFuncAddr ...
	// `addr = func_addr FN`.
	OpcodeFuncAddr

	// OpcodeSplat ...
	// `a = splat x`.
	OpcodeSplat

	// OpcodeSwizzle ...
	// `a = swizzle x, y`.
	OpcodeSwizzle

	// OpcodeInsertlane ...
	// `a = insertlane x, y, Idx`. (TernaryImm8)
	// Type inferred from `x`.
	OpcodeInsertlane

	// OpcodeExtractlane ...
	// `a = extractlane x, Idx`. (BinaryImm8)
	// Type inferred from `x`.
	OpcodeExtractlane

	// OpcodeSmin ...
	// `a = smin x, y`.
	// Type inferred from `x`.
	OpcodeSmin

	// OpcodeUmin ...
	// `a = umin x, y`.
	// Type inferred from `x`.
	OpcodeUmin

	// OpcodeSmax ...
	// `a = smax x, y`.
	// Type inferred from `x`.
	OpcodeSmax

	// OpcodeUmax ...
	// `a = umax x, y`.
	// Type inferred from `x`.
	OpcodeUmax

	// OpcodeAvgRound ...
	// `a = avg_round x, y`.
	// Type inferred from `x`.
	OpcodeAvgRound

	// OpcodeUaddSat ...
	// `a = uadd_sat x, y`.
	// Type inferred from `x`.
	OpcodeUaddSat

	// OpcodeSaddSat ...
	// `a = sadd_sat x, y`.
	// Type inferred from `x`.
	OpcodeSaddSat

	// OpcodeUsubSat ...
	// `a = usub_sat x, y`.
	// Type inferred from `x`.
	OpcodeUsubSat

	// OpcodeSsubSat ...
	// `a = ssub_sat x, y`.
	// Type inferred from `x`.
	OpcodeSsubSat

	// OpcodeLoad ...
	// `a = load MemFlags, p, Offset`.
	OpcodeLoad

	// OpcodeStore ...
	// `store MemFlags, x, p, Offset`.
	// Type inferred from `x`.
	OpcodeStore

	// OpcodeUload8 ...
	// `a = uload8 MemFlags, p, Offset`.
	OpcodeUload8

	// OpcodeSload8 ...
	// `a = sload8 MemFlags, p, Offset`.
	OpcodeSload8

	// OpcodeIstore8 ...
	// `istore8 MemFlags, x, p, Offset`.
	// Type inferred from `x`.
	OpcodeIstore8

	// OpcodeUload16 ...
	// `a = uload16 MemFlags, p, Offset`.
	OpcodeUload16

	// OpcodeSload16 ...
	// `a = sload16 MemFlags, p, Offset`.
	OpcodeSload16

	// OpcodeIstore16 ...
	// `istore16 MemFlags, x, p, Offset`.
	// Type inferred from `x`.
	OpcodeIstore16

	// OpcodeUload32 ...
	// `a = uload32 MemFlags, p, Offset`.
	// Type inferred from `p`.
	OpcodeUload32

	// OpcodeSload32 ...
	// `a = sload32 MemFlags, p, Offset`.
	// Type inferred from `p`.
	OpcodeSload32

	// OpcodeIstore32 ...
	// `istore32 MemFlags, x, p, Offset`.
	// Type inferred from `x`.
	OpcodeIstore32

	// OpcodeUload8x8 ...
	// `a = uload8x8 MemFlags, p, Offset`.
	// Type inferred from `p`.
	OpcodeUload8x8

	// OpcodeSload8x8 ...
	// `a = sload8x8 MemFlags, p, Offset`.
	// Type inferred from `p`.
	OpcodeSload8x8

	// OpcodeUload16x4 ...
	// `a = uload16x4 MemFlags, p, Offset`.
	// Type inferred from `p`.
	OpcodeUload16x4

	// OpcodeSload16x4 ...
	// `a = sload16x4 MemFlags, p, Offset`.
	// Type inferred from `p`.
	OpcodeSload16x4

	// OpcodeUload32x2 ...
	// `a = uload32x2 MemFlags, p, Offset`.
	// Type inferred from `p`.
	OpcodeUload32x2

	// OpcodeSload32x2 ...
	// `a = sload32x2 MemFlags, p, Offset`.
	// Type inferred from `p`.
	OpcodeSload32x2

	// OpcodeStackLoad ...
	// `a = stack_load SS, Offset`.
	OpcodeStackLoad

	// OpcodeStackStore ...
	// `stack_store x, SS, Offset`.
	// Type inferred from `x`.
	OpcodeStackStore

	// OpcodeStackAddr ...
	// `addr = stack_addr SS, Offset`.
	OpcodeStackAddr

	// OpcodeDynamicStackLoad ...
	// `a = dynamic_stack_load DSS`.
	OpcodeDynamicStackLoad

	// OpcodeDynamicStackStore ...
	// `dynamic_stack_store x, DSS`.
	// Type inferred from `x`.
	OpcodeDynamicStackStore

	// OpcodeDynamicStackAddr ...
	// `addr = dynamic_stack_addr DSS`.
	OpcodeDynamicStackAddr

	// OpcodeGlobalValue ...
	// `a = global_value GV`.
	OpcodeGlobalValue

	// OpcodeSymbolValue ...
	// `a = symbol_value GV`.
	OpcodeSymbolValue

	// OpcodeTlsValue ...
	// `a = tls_value GV`.
	OpcodeTlsValue

	// OpcodeHeapAddr ...
	// `addr = heap_addr H, index, Offset, Size`.
	OpcodeHeapAddr

	// OpcodeHeapLoad ...
	// `a = heap_load heap_imm, index`.
	OpcodeHeapLoad

	// OpcodeHeapStore ...
	// `heap_store heap_imm, index, a`.
	// Type inferred from `index`.
	OpcodeHeapStore

	// OpcodeGetPinnedReg ...
	// `addr = get_pinned_reg`.
	OpcodeGetPinnedReg

	// OpcodeSetPinnedReg ...
	// `set_pinned_reg addr`.
	// Type inferred from `addr`.
	OpcodeSetPinnedReg

	// OpcodeGetFramePointer ...
	// `addr = get_frame_pointer`.
	OpcodeGetFramePointer

	// OpcodeGetStackPointer ...
	// `addr = get_stack_pointer`.
	OpcodeGetStackPointer

	// OpcodeGetReturnAddress ...
	// `addr = get_return_address`.
	OpcodeGetReturnAddress

	// OpcodeTableAddr ...
	// `addr = table_addr T, p, Offset`.
	OpcodeTableAddr

	// OpcodeIconst represents the integer const.
	OpcodeIconst

	// OpcodeF32const ...
	// `a = f32const N`. (UnaryIeee32)
	OpcodeF32const

	// OpcodeF64const ...
	// `a = f64const N`. (UnaryIeee64)
	OpcodeF64const

	// OpcodeVconst ...
	// `a = vconst N`.
	OpcodeVconst

	// OpcodeShuffle ...
	// `a = shuffle a, b, mask`.
	OpcodeShuffle

	// OpcodeNull ...
	// `a = null`.
	OpcodeNull

	// OpcodeNop ...
	// `nop`.
	OpcodeNop

	// OpcodeSelect ...
	// `a = select c, x, y`.
	// Type inferred from `x`.
	OpcodeSelect

	// OpcodeSelectSpectreGuard ...
	// `a = select_spectre_guard c, x, y`.
	// Type inferred from `x`.
	OpcodeSelectSpectreGuard

	// OpcodeBitselect ...
	// `a = bitselect c, x, y`.
	// Type inferred from `x`.
	OpcodeBitselect

	// OpcodeVsplit ...
	// `lo, hi = vsplit x`.
	// Type inferred from `x`.
	OpcodeVsplit

	// OpcodeVconcat ...
	// `a = vconcat x, y`.
	// Type inferred from `x`.
	OpcodeVconcat

	// OpcodeVselect ...
	// `a = vselect c, x, y`.
	// Type inferred from `x`.
	OpcodeVselect

	// OpcodeVanyTrue ...
	// `s = vany_true a`.
	// Type inferred from `a`.
	OpcodeVanyTrue

	// OpcodeVallTrue ...
	// `s = vall_true a`.
	// Type inferred from `a`.
	OpcodeVallTrue

	// OpcodeVhighBits ...
	// `x = vhigh_bits a`.
	OpcodeVhighBits

	// OpcodeIcmp ...
	// `a = icmp Cond, x, y`.
	// Type inferred from `x`.
	OpcodeIcmp

	// OpcodeIcmpImm ...
	// `a = icmp_imm Cond, x, Y`.
	// Type inferred from `x`.
	OpcodeIcmpImm

	// OpcodeIfcmp ...
	// `f = ifcmp x, y`.
	// Type inferred from `x`.
	OpcodeIfcmp

	// OpcodeIfcmpImm ...
	// `f = ifcmp_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeIfcmpImm

	// OpcodeIadd ...
	// `a = iadd x, y`.
	// Type inferred from `x`.
	OpcodeIadd

	// OpcodeIsub ...
	// `a = isub x, y`.
	// Type inferred from `x`.
	OpcodeIsub

	// OpcodeIneg ...
	// `a = ineg x`.
	// Type inferred from `x`.
	OpcodeIneg

	// OpcodeIabs ...
	// `a = iabs x`.
	// Type inferred from `x`.
	OpcodeIabs

	// OpcodeImul ...
	// `a = imul x, y`.
	// Type inferred from `x`.
	OpcodeImul

	// OpcodeUmulhi ...
	// `a = umulhi x, y`.
	// Type inferred from `x`.
	OpcodeUmulhi

	// OpcodeSmulhi ...
	// `a = smulhi x, y`.
	// Type inferred from `x`.
	OpcodeSmulhi

	// OpcodeSqmulRoundSat ...
	// `a = sqmul_round_sat x, y`.
	// Type inferred from `x`.
	OpcodeSqmulRoundSat

	// OpcodeUdiv ...
	// `a = udiv x, y`.
	// Type inferred from `x`.
	OpcodeUdiv

	// OpcodeSdiv ...
	// `a = sdiv x, y`.
	// Type inferred from `x`.
	OpcodeSdiv

	// OpcodeUrem ...
	// `a = urem x, y`.
	// Type inferred from `x`.
	OpcodeUrem

	// OpcodeSrem ...
	// `a = srem x, y`.
	// Type inferred from `x`.
	OpcodeSrem

	// OpcodeIaddImm ...
	// `a = iadd_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeIaddImm

	// OpcodeImulImm ...
	// `a = imul_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeImulImm

	// OpcodeUdivImm ...
	// `a = udiv_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeUdivImm

	// OpcodeSdivImm ...
	// `a = sdiv_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeSdivImm

	// OpcodeUremImm ...
	// `a = urem_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeUremImm

	// OpcodeSremImm ...
	// `a = srem_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeSremImm

	// OpcodeIrsubImm ...
	// `a = irsub_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeIrsubImm

	// OpcodeIaddCin ...
	// `a = iadd_cin x, y, c_in`.
	// Type inferred from `y`.
	OpcodeIaddCin

	// OpcodeIaddIfcin ...
	// `a = iadd_ifcin x, y, c_in`.
	// Type inferred from `y`.
	OpcodeIaddIfcin

	// OpcodeIaddCout ...
	// `a, c_out = iadd_cout x, y`.
	// Type inferred from `x`.
	OpcodeIaddCout

	// OpcodeIaddIfcout ...
	// `a, c_out = iadd_ifcout x, y`.
	// Type inferred from `x`.
	OpcodeIaddIfcout

	// OpcodeIaddCarry ...
	// `a, c_out = iadd_carry x, y, c_in`.
	// Type inferred from `y`.
	OpcodeIaddCarry

	// OpcodeIaddIfcarry ...
	// `a, c_out = iadd_ifcarry x, y, c_in`.
	// Type inferred from `y`.
	OpcodeIaddIfcarry

	// OpcodeUaddOverflowTrap ...
	// `a = uadd_overflow_trap x, y, code`.
	// Type inferred from `x`.
	OpcodeUaddOverflowTrap

	// OpcodeIsubBin ...
	// `a = isub_bin x, y, b_in`.
	// Type inferred from `y`.
	OpcodeIsubBin

	// OpcodeIsubIfbin ...
	// `a = isub_ifbin x, y, b_in`.
	// Type inferred from `y`.
	OpcodeIsubIfbin

	// OpcodeIsubBout ...
	// `a, b_out = isub_bout x, y`.
	// Type inferred from `x`.
	OpcodeIsubBout

	// OpcodeIsubIfbout ...
	// `a, b_out = isub_ifbout x, y`.
	// Type inferred from `x`.
	OpcodeIsubIfbout

	// OpcodeIsubBorrow ...
	// `a, b_out = isub_borrow x, y, b_in`.
	// Type inferred from `y`.
	OpcodeIsubBorrow

	// OpcodeIsubIfborrow ...
	// `a, b_out = isub_ifborrow x, y, b_in`.
	// Type inferred from `y`.
	OpcodeIsubIfborrow

	// OpcodeBand ...
	// `a = band x, y`.
	// Type inferred from `x`.
	OpcodeBand

	// OpcodeBor ...
	// `a = bor x, y`.
	// Type inferred from `x`.
	OpcodeBor

	// OpcodeBxor ...
	// `a = bxor x, y`.
	// Type inferred from `x`.
	OpcodeBxor

	// OpcodeBnot ...
	// `a = bnot x`.
	// Type inferred from `x`.
	OpcodeBnot

	// OpcodeBandNot ...
	// `a = band_not x, y`.
	// Type inferred from `x`.
	OpcodeBandNot

	// OpcodeBorNot ...
	// `a = bor_not x, y`.
	// Type inferred from `x`.
	OpcodeBorNot

	// OpcodeBxorNot ...
	// `a = bxor_not x, y`.
	// Type inferred from `x`.
	OpcodeBxorNot

	// OpcodeBandImm ...
	// `a = band_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeBandImm

	// OpcodeBorImm ...
	// `a = bor_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeBorImm

	// OpcodeBxorImm ...
	// `a = bxor_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeBxorImm

	// OpcodeRotl ...
	// `a = rotl x, y`.
	// Type inferred from `x`.
	OpcodeRotl

	// OpcodeRotr ...
	// `a = rotr x, y`.
	// Type inferred from `x`.
	OpcodeRotr

	// OpcodeRotlImm ...
	// `a = rotl_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeRotlImm

	// OpcodeRotrImm ...
	// `a = rotr_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeRotrImm

	// OpcodeIshl ...
	// `a = ishl x, y`.
	// Type inferred from `x`.
	OpcodeIshl

	// OpcodeUshr ...
	// `a = ushr x, y`.
	// Type inferred from `x`.
	OpcodeUshr

	// OpcodeSshr ...
	// `a = sshr x, y`.
	// Type inferred from `x`.
	OpcodeSshr

	// OpcodeIshlImm ...
	// `a = ishl_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeIshlImm

	// OpcodeUshrImm ...
	// `a = ushr_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeUshrImm

	// OpcodeSshrImm ...
	// `a = sshr_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeSshrImm

	// OpcodeBitrev ...
	// `a = bitrev x`.
	// Type inferred from `x`.
	OpcodeBitrev

	// OpcodeClz ...
	// `a = clz x`.
	// Type inferred from `x`.
	OpcodeClz

	// OpcodeCls ...
	// `a = cls x`.
	// Type inferred from `x`.
	OpcodeCls

	// OpcodeCtz ...
	// `a = ctz x`.
	// Type inferred from `x`.
	OpcodeCtz

	// OpcodeBswap ...
	// `a = bswap x`.
	// Type inferred from `x`.
	OpcodeBswap

	// OpcodePopcnt ...
	// `a = popcnt x`.
	// Type inferred from `x`.
	OpcodePopcnt

	// OpcodeFcmp ...
	// `a = fcmp Cond, x, y`.
	// Type inferred from `x`.
	OpcodeFcmp

	// OpcodeFfcmp ...
	// `f = ffcmp x, y`.
	// Type inferred from `x`.
	OpcodeFfcmp

	// OpcodeFadd ...
	// `a = fadd x, y`.
	// Type inferred from `x`.
	OpcodeFadd

	// OpcodeFsub ...
	// `a = fsub x, y`.
	// Type inferred from `x`.
	OpcodeFsub

	// OpcodeFmul ...
	// `a = fmul x, y`.
	// Type inferred from `x`.
	OpcodeFmul

	// OpcodeFdiv ...
	// `a = fdiv x, y`.
	// Type inferred from `x`.
	OpcodeFdiv

	// OpcodeSqrt ...
	// `a = sqrt x`.
	// Type inferred from `x`.
	OpcodeSqrt

	// OpcodeFma ...
	// `a = fma x, y, z`.
	// Type inferred from `y`.
	OpcodeFma

	// OpcodeFneg ...
	// `a = fneg x`.
	// Type inferred from `x`.
	OpcodeFneg

	// OpcodeFabs ...
	// `a = fabs x`.
	// Type inferred from `x`.
	OpcodeFabs

	// OpcodeFcopysign ...
	// `a = fcopysign x, y`.
	// Type inferred from `x`.
	OpcodeFcopysign

	// OpcodeFmin ...
	// `a = fmin x, y`.
	// Type inferred from `x`.
	OpcodeFmin

	// OpcodeFminPseudo ...
	// `a = fmin_pseudo x, y`.
	// Type inferred from `x`.
	OpcodeFminPseudo

	// OpcodeFmax ...
	// `a = fmax x, y`.
	// Type inferred from `x`.
	OpcodeFmax

	// OpcodeFmaxPseudo ...
	// `a = fmax_pseudo x, y`.
	// Type inferred from `x`.
	OpcodeFmaxPseudo

	// OpcodeCeil ...
	// `a = ceil x`.
	// Type inferred from `x`.
	OpcodeCeil

	// OpcodeFloor ...
	// `a = floor x`.
	// Type inferred from `x`.
	OpcodeFloor

	// OpcodeTrunc ...
	// `a = trunc x`.
	// Type inferred from `x`.
	OpcodeTrunc

	// OpcodeNearest ...
	// `a = nearest x`.
	// Type inferred from `x`.
	OpcodeNearest

	// OpcodeIsNull ...
	// `a = is_null x`.
	// Type inferred from `x`.
	OpcodeIsNull

	// OpcodeIsInvalid ...
	// `a = is_invalid x`.
	// Type inferred from `x`.
	OpcodeIsInvalid

	// OpcodeBitcast ...
	// `a = bitcast MemFlags, x`.
	OpcodeBitcast

	// OpcodeScalarToVector ...
	// `a = scalar_to_vector s`.
	OpcodeScalarToVector

	// OpcodeBmask ...
	// `a = bmask x`.
	OpcodeBmask

	// OpcodeIreduce ...
	// `a = ireduce x`.
	OpcodeIreduce
	// `a = snarrow x, y`.

	// OpcodeSnarrow ...
	// Type inferred from `x`.
	OpcodeSnarrow
	// `a = unarrow x, y`.

	// OpcodeUnarrow ...
	// Type inferred from `x`.
	OpcodeUnarrow
	// `a = uunarrow x, y`.

	// OpcodeUunarrow ...
	// Type inferred from `x`.
	OpcodeUunarrow
	// `a = swiden_low x`.

	// OpcodeSwidenLow ...
	// Type inferred from `x`.
	OpcodeSwidenLow
	// `a = swiden_high x`.

	// OpcodeSwidenHigh ...
	// Type inferred from `x`.
	OpcodeSwidenHigh
	// `a = uwiden_low x`.

	// OpcodeUwidenLow ...
	// Type inferred from `x`.
	OpcodeUwidenLow
	// `a = uwiden_high x`.

	// OpcodeUwidenHigh ...
	// Type inferred from `x`.
	OpcodeUwidenHigh
	// `a = iadd_pairwise x, y`.

	// OpcodeIaddPairwise ...
	// Type inferred from `x`.
	OpcodeIaddPairwise

	// OpcodeWideningPairwiseDotProductS ...
	// `a = widening_pairwise_dot_product_s x, y`.
	OpcodeWideningPairwiseDotProductS

	// OpcodeUextend ...
	// `a = uextend x`.
	OpcodeUextend

	// OpcodeSextend ...
	// `a = sextend x`.
	OpcodeSextend

	// OpcodeFpromote ...
	// `a = fpromote x`.
	OpcodeFpromote

	// OpcodeFdemote ...
	// `a = fdemote x`.
	OpcodeFdemote

	// OpcodeFvdemote ...
	// `a = fvdemote x`.
	OpcodeFvdemote

	// OpcodeFvpromoteLow ...
	// `x = fvpromote_low a`.
	OpcodeFvpromoteLow

	// OpcodeFcvtToUint ...
	// `a = fcvt_to_uint x`.
	OpcodeFcvtToUint

	// OpcodeFcvtToSint ...
	// `a = fcvt_to_sint x`.
	OpcodeFcvtToSint

	// OpcodeFcvtToUintSat ...
	// `a = fcvt_to_uint_sat x`.
	OpcodeFcvtToUintSat

	// OpcodeFcvtToSintSat ...
	// `a = fcvt_to_sint_sat x`.
	OpcodeFcvtToSintSat

	// OpcodeFcvtFromUint ...
	// `a = fcvt_from_uint x`.
	OpcodeFcvtFromUint

	// OpcodeFcvtFromSint ...
	// `a = fcvt_from_sint x`.
	OpcodeFcvtFromSint

	// OpcodeFcvtLowFromSint ...
	// `a = fcvt_low_from_sint x`.
	OpcodeFcvtLowFromSint

	// OpcodeIsplit ...
	// `lo, hi = isplit x`.
	// Type inferred from `x`.
	OpcodeIsplit

	// OpcodeIconcat ...
	// `a = iconcat lo, hi`.
	// Type inferred from `lo`.
	OpcodeIconcat

	// OpcodeAtomicRmw ...
	// `a = atomic_rmw MemFlags, AtomicRmwOp, p, x`.
	OpcodeAtomicRmw

	// OpcodeAtomicCas ...
	// `a = atomic_cas MemFlags, p, e, x`.
	// Type inferred from `x`.
	OpcodeAtomicCas

	// OpcodeAtomicLoad ...
	// `a = atomic_load MemFlags, p`.
	OpcodeAtomicLoad

	// OpcodeAtomicStore ...
	// `atomic_store MemFlags, x, p`.
	// Type inferred from `x`.
	OpcodeAtomicStore

	// OpcodeFence ...
	// `fence`.
	OpcodeFence

	// OpcodeExtractVector ...
	// `a = extract_vector x, y`. (BinaryImm8)
	// Type inferred from `x`.
	OpcodeExtractVector

	opcodeEnd
)

var numReturns = [...]*struct {
	num     int
	unknown bool
}{
	OpcodeJump:     {num: 0, unknown: false},
	OpcodeIconst:   {num: 1, unknown: false},
	OpcodeF32const: {num: 1, unknown: false},
	OpcodeF64const: {num: 1, unknown: false},
	OpcodeReturn:   {num: 0, unknown: false},
	opcodeEnd:      nil,
}

func (o Opcode) numReturns() (num int, unknown bool) {
	rs := numReturns[o]
	if rs == nil {
		panic("TODO: " + o.String())
	} else {
		return rs.num, rs.unknown
	}
}

func (i *Instruction) AsIconst64(v uint64) {
	i.opcode = OpcodeIconst
	i.typ = TypeI64
	i.u64 = v
}

func (i *Instruction) AsIconst32(v uint32) {
	i.opcode = OpcodeIconst
	i.typ = TypeI32
	i.u64 = uint64(v)
}

func init() {
	instructionFormats[OpcodeIconst] = func(i *Instruction) (ret string) {
		switch i.typ {
		case TypeI32:
			ret = fmt.Sprintf("_32 %#x", uint32(i.u64))
		case TypeI64:
			ret = fmt.Sprintf("_64 %#x", i.u64)
		}
		return
	}
}

func (i *Instruction) AsF32const(f float32) {
	i.opcode = OpcodeF32const
	i.typ = TypeF64
	i.u64 = uint64(math.Float32bits(f))
}

func (i *Instruction) AsF64const(f float64) {
	i.opcode = OpcodeF64const
	i.typ = TypeF64
	i.u64 = math.Float64bits(f)
}

func init() {
	instructionFormats[OpcodeF32const] = func(i *Instruction) (ret string) {
		ret = fmt.Sprintf(" %f", math.Float32frombits(uint32(i.u64)))
		return
	}
	instructionFormats[OpcodeF64const] = func(i *Instruction) (ret string) {
		ret = fmt.Sprintf(" %f", math.Float64frombits(i.u64))
		return
	}
}

func (i *Instruction) AsReturn(vs []Value) {
	i.opcode = OpcodeReturn
	i.vs = vs
}

func init() {
	instructionFormats[OpcodeReturn] = func(i *Instruction) (ret string) {
		if len(i.vs) == 0 {
			return
		}
		vs := make([]string, len(i.vs))
		for idx := range vs {
			vs[idx] = i.vs[idx].String()
		}
		ret = fmt.Sprintf(" %s", strings.Join(vs, ","))
		return
	}
}

func (i *Instruction) AsJump(vs []Value, target BasicBlock) {
	i.opcode = OpcodeJump
	i.vs = vs
	i.blk1 = target
}

func init() {
	instructionFormats[OpcodeJump] = func(i *Instruction) (ret string) {
		vs := make([]string, len(i.vs)+1)
		vs[0] = fmt.Sprintf(" blk%d", i.blk1.(*basicBlock).id)
		for idx := range i.vs {
			vs[idx+1] = i.vs[idx].String()
		}

		ret = strings.Join(vs, ", ")
		return
	}
}

// String implements fmt.Stringer.
func (i *Instruction) String() (ret string) {
	fn := instructionFormats[i.opcode]
	if fn == nil {
		panic(fmt.Sprintf("TODO: format for %s", i.opcode))
	}
	instr := i.opcode.String() + fn(i)

	var rvs []string
	if rv := i.rValue; rv.Valid() {
		rvs = append(rvs, rv.String())
	}

	for _, v := range i.rValues {
		rvs = append(rvs, v.String())
	}

	if len(rvs) > 0 {
		return fmt.Sprintf("%s = %s", strings.Join(rvs, ", "), instr)
	} else {
		return instr
	}
}

// String implements fmt.Stringer.
func (o Opcode) String() (ret string) {
	switch o {
	case OpcodeJump:
		return "Jump"
	case OpcodeBrz:
		return "Brz"
	case OpcodeBrnz:
		return "Brnz"
	case OpcodeBrTable:
		return "BrTable"
	case OpcodeDebugtrap:
		return "Debugtrap"
	case OpcodeTrap:
		return "Trap"
	case OpcodeTrapz:
		return "Trapz"
	case OpcodeResumableTrap:
		return "ResumableTrap"
	case OpcodeTrapnz:
		return "Trapnz"
	case OpcodeResumableTrapnz:
		return "ResumableTrapnz"
	case OpcodeReturn:
		return "Return"
	case OpcodeCall:
		return "Call"
	case OpcodeCallIndirect:
		return "CallIndirect"
	case OpcodeFuncAddr:
		return "FuncAddr"
	case OpcodeSplat:
		return "Splat"
	case OpcodeSwizzle:
		return "Swizzle"
	case OpcodeInsertlane:
		return "Insertlane"
	case OpcodeExtractlane:
		return "Extractlane"
	case OpcodeSmin:
		return "Smin"
	case OpcodeUmin:
		return "Umin"
	case OpcodeSmax:
		return "Smax"
	case OpcodeUmax:
		return "Umax"
	case OpcodeAvgRound:
		return "AvgRound"
	case OpcodeUaddSat:
		return "UaddSat"
	case OpcodeSaddSat:
		return "SaddSat"
	case OpcodeUsubSat:
		return "UsubSat"
	case OpcodeSsubSat:
		return "SsubSat"
	case OpcodeLoad:
		return "Load"
	case OpcodeStore:
		return "Store"
	case OpcodeUload8:
		return "Uload8"
	case OpcodeSload8:
		return "Sload8"
	case OpcodeIstore8:
		return "Istore8"
	case OpcodeUload16:
		return "Uload16"
	case OpcodeSload16:
		return "Sload16"
	case OpcodeIstore16:
		return "Istore16"
	case OpcodeUload32:
		return "Uload32"
	case OpcodeSload32:
		return "Sload32"
	case OpcodeIstore32:
		return "Istore32"
	case OpcodeUload8x8:
		return "Uload8x8"
	case OpcodeSload8x8:
		return "Sload8x8"
	case OpcodeUload16x4:
		return "Uload16x4"
	case OpcodeSload16x4:
		return "Sload16x4"
	case OpcodeUload32x2:
		return "Uload32x2"
	case OpcodeSload32x2:
		return "Sload32x2"
	case OpcodeStackLoad:
		return "StackLoad"
	case OpcodeStackStore:
		return "StackStore"
	case OpcodeStackAddr:
		return "StackAddr"
	case OpcodeDynamicStackLoad:
		return "DynamicStackLoad"
	case OpcodeDynamicStackStore:
		return "DynamicStackStore"
	case OpcodeDynamicStackAddr:
		return "DynamicStackAddr"
	case OpcodeGlobalValue:
		return "GlobalValue"
	case OpcodeSymbolValue:
		return "SymbolValue"
	case OpcodeTlsValue:
		return "TlsValue"
	case OpcodeHeapAddr:
		return "HeapAddr"
	case OpcodeHeapLoad:
		return "HeapLoad"
	case OpcodeHeapStore:
		return "HeapStore"
	case OpcodeGetPinnedReg:
		return "GetPinnedReg"
	case OpcodeSetPinnedReg:
		return "SetPinnedReg"
	case OpcodeGetFramePointer:
		return "GetFramePointer"
	case OpcodeGetStackPointer:
		return "GetStackPointer"
	case OpcodeGetReturnAddress:
		return "GetReturnAddress"
	case OpcodeTableAddr:
		return "TableAddr"
	case OpcodeIconst:
		return "Iconst"
	case OpcodeF32const:
		return "F32const"
	case OpcodeF64const:
		return "F64const"
	case OpcodeVconst:
		return "Vconst"
	case OpcodeShuffle:
		return "Shuffle"
	case OpcodeNull:
		return "Null"
	case OpcodeNop:
		return "Nop"
	case OpcodeSelect:
		return "Select"
	case OpcodeSelectSpectreGuard:
		return "SelectSpectreGuard"
	case OpcodeBitselect:
		return "Bitselect"
	case OpcodeVsplit:
		return "Vsplit"
	case OpcodeVconcat:
		return "Vconcat"
	case OpcodeVselect:
		return "Vselect"
	case OpcodeVanyTrue:
		return "VanyTrue"
	case OpcodeVallTrue:
		return "VallTrue"
	case OpcodeVhighBits:
		return "VhighBits"
	case OpcodeIcmp:
		return "Icmp"
	case OpcodeIcmpImm:
		return "IcmpImm"
	case OpcodeIfcmp:
		return "Ifcmp"
	case OpcodeIfcmpImm:
		return "IfcmpImm"
	case OpcodeIadd:
		return "Iadd"
	case OpcodeIsub:
		return "Isub"
	case OpcodeIneg:
		return "Ineg"
	case OpcodeIabs:
		return "Iabs"
	case OpcodeImul:
		return "Imul"
	case OpcodeUmulhi:
		return "Umulhi"
	case OpcodeSmulhi:
		return "Smulhi"
	case OpcodeSqmulRoundSat:
		return "SqmulRoundSat"
	case OpcodeUdiv:
		return "Udiv"
	case OpcodeSdiv:
		return "Sdiv"
	case OpcodeUrem:
		return "Urem"
	case OpcodeSrem:
		return "Srem"
	case OpcodeIaddImm:
		return "IaddImm"
	case OpcodeImulImm:
		return "ImulImm"
	case OpcodeUdivImm:
		return "UdivImm"
	case OpcodeSdivImm:
		return "SdivImm"
	case OpcodeUremImm:
		return "UremImm"
	case OpcodeSremImm:
		return "SremImm"
	case OpcodeIrsubImm:
		return "IrsubImm"
	case OpcodeIaddCin:
		return "IaddCin"
	case OpcodeIaddIfcin:
		return "IaddIfcin"
	case OpcodeIaddCout:
		return "IaddCout"
	case OpcodeIaddIfcout:
		return "IaddIfcout"
	case OpcodeIaddCarry:
		return "IaddCarry"
	case OpcodeIaddIfcarry:
		return "IaddIfcarry"
	case OpcodeUaddOverflowTrap:
		return "UaddOverflowTrap"
	case OpcodeIsubBin:
		return "IsubBin"
	case OpcodeIsubIfbin:
		return "IsubIfbin"
	case OpcodeIsubBout:
		return "IsubBout"
	case OpcodeIsubIfbout:
		return "IsubIfbout"
	case OpcodeIsubBorrow:
		return "IsubBorrow"
	case OpcodeIsubIfborrow:
		return "IsubIfborrow"
	case OpcodeBand:
		return "Band"
	case OpcodeBor:
		return "Bor"
	case OpcodeBxor:
		return "Bxor"
	case OpcodeBnot:
		return "Bnot"
	case OpcodeBandNot:
		return "BandNot"
	case OpcodeBorNot:
		return "BorNot"
	case OpcodeBxorNot:
		return "BxorNot"
	case OpcodeBandImm:
		return "BandImm"
	case OpcodeBorImm:
		return "BorImm"
	case OpcodeBxorImm:
		return "BxorImm"
	case OpcodeRotl:
		return "Rotl"
	case OpcodeRotr:
		return "Rotr"
	case OpcodeRotlImm:
		return "RotlImm"
	case OpcodeRotrImm:
		return "RotrImm"
	case OpcodeIshl:
		return "Ishl"
	case OpcodeUshr:
		return "Ushr"
	case OpcodeSshr:
		return "Sshr"
	case OpcodeIshlImm:
		return "IshlImm"
	case OpcodeUshrImm:
		return "UshrImm"
	case OpcodeSshrImm:
		return "SshrImm"
	case OpcodeBitrev:
		return "Bitrev"
	case OpcodeClz:
		return "Clz"
	case OpcodeCls:
		return "Cls"
	case OpcodeCtz:
		return "Ctz"
	case OpcodeBswap:
		return "Bswap"
	case OpcodePopcnt:
		return "Popcnt"
	case OpcodeFcmp:
		return "Fcmp"
	case OpcodeFfcmp:
		return "Ffcmp"
	case OpcodeFadd:
		return "Fadd"
	case OpcodeFsub:
		return "Fsub"
	case OpcodeFmul:
		return "Fmul"
	case OpcodeFdiv:
		return "Fdiv"
	case OpcodeSqrt:
		return "Sqrt"
	case OpcodeFma:
		return "Fma"
	case OpcodeFneg:
		return "Fneg"
	case OpcodeFabs:
		return "Fabs"
	case OpcodeFcopysign:
		return "Fcopysign"
	case OpcodeFmin:
		return "Fmin"
	case OpcodeFminPseudo:
		return "FminPseudo"
	case OpcodeFmax:
		return "Fmax"
	case OpcodeFmaxPseudo:
		return "FmaxPseudo"
	case OpcodeCeil:
		return "Ceil"
	case OpcodeFloor:
		return "Floor"
	case OpcodeTrunc:
		return "Trunc"
	case OpcodeNearest:
		return "Nearest"
	case OpcodeIsNull:
		return "IsNull"
	case OpcodeIsInvalid:
		return "IsInvalid"
	case OpcodeBitcast:
		return "Bitcast"
	case OpcodeScalarToVector:
		return "ScalarToVector"
	case OpcodeBmask:
		return "Bmask"
	case OpcodeIreduce:
		return "Ireduce"
	case OpcodeSnarrow:
		return "Snarrow"
	case OpcodeUnarrow:
		return "Unarrow"
	case OpcodeUunarrow:
		return "Uunarrow"
	case OpcodeSwidenLow:
		return "SwidenLow"
	case OpcodeSwidenHigh:
		return "SwidenHigh"
	case OpcodeUwidenLow:
		return "UwidenLow"
	case OpcodeUwidenHigh:
		return "UwidenHigh"
	case OpcodeIaddPairwise:
		return "IaddPairwise"
	case OpcodeWideningPairwiseDotProductS:
		return "WideningPairwiseDotProductS"
	case OpcodeUextend:
		return "Uextend"
	case OpcodeSextend:
		return "Sextend"
	case OpcodeFpromote:
		return "Fpromote"
	case OpcodeFdemote:
		return "Fdemote"
	case OpcodeFvdemote:
		return "Fvdemote"
	case OpcodeFvpromoteLow:
		return "FvpromoteLow"
	case OpcodeFcvtToUint:
		return "FcvtToUint"
	case OpcodeFcvtToSint:
		return "FcvtToSint"
	case OpcodeFcvtToUintSat:
		return "FcvtToUintSat"
	case OpcodeFcvtToSintSat:
		return "FcvtToSintSat"
	case OpcodeFcvtFromUint:
		return "FcvtFromUint"
	case OpcodeFcvtFromSint:
		return "FcvtFromSint"
	case OpcodeFcvtLowFromSint:
		return "FcvtLowFromSint"
	case OpcodeIsplit:
		return "Isplit"
	case OpcodeIconcat:
		return "Iconcat"
	case OpcodeAtomicRmw:
		return "AtomicRmw"
	case OpcodeAtomicCas:
		return "AtomicCas"
	case OpcodeAtomicLoad:
		return "AtomicLoad"
	case OpcodeAtomicStore:
		return "AtomicStore"
	case OpcodeFence:
		return "Fence"
	case OpcodeExtractVector:
		return "ExtractVector"
	}
	panic(fmt.Sprintf("unknown opcode %d", o))
}
