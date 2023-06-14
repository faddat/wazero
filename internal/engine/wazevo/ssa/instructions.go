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
	v          Value
	v2         Value
	vs         []Value
	typ        Type
	blk        BasicBlock
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

// Followings match the generated code from https://github.com/bytecodealliance/wasmtime/blob/v9.0.3/cranelift/codegen/meta/src/shared/instructions.rs
// TODO: complete opcode comments.
// TODO: there should be unnecessary opcodes.
const (
	// OpcodeJump takes the list of args to the `block` and unconditionally jumps to it.
	OpcodeJump Opcode = 1 + iota

	// OpcodeBrz branches into `blk` with `args`  if the value `c` equals zero: `Brz c, blk, args`.
	OpcodeBrz

	// OpcodeBrnz branches into `blk` with `args`  if the value `c` is not zero: `Brnz c, blk, args`.
	OpcodeBrnz

	// OpcodeBrTable ...
	// `BrTable x, block, JT`.
	OpcodeBrTable

	// OpcodeTrap exit the execution immediately.
	OpcodeTrap

	// OpcodeReturn returns from the function: `return rvalues`.
	OpcodeReturn

	// OpcodeCall calls a function specified by the symbol FN with arguments `args`.
	// `returnvals = Call FN, args...`
	OpcodeCall

	// OpcodeCallIndirect ...
	// `rvals = call_indirect SIG, callee, args`.
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
	OpcodeInsertlane

	// OpcodeExtractlane ...
	// `a = extractlane x, Idx`. (BinaryImm8)
	OpcodeExtractlane

	// OpcodeSmin ...
	// `a = smin x, y`.
	OpcodeSmin

	// OpcodeUmin ...
	// `a = umin x, y`.
	OpcodeUmin

	// OpcodeSmax ...
	// `a = smax x, y`.
	OpcodeSmax

	// OpcodeUmax ...
	// `a = umax x, y`.
	OpcodeUmax

	// OpcodeAvgRound ...
	// `a = avg_round x, y`.
	OpcodeAvgRound

	// OpcodeUaddSat ...
	// `a = uadd_sat x, y`.
	OpcodeUaddSat

	// OpcodeSaddSat ...
	// `a = sadd_sat x, y`.
	OpcodeSaddSat

	// OpcodeUsubSat ...
	// `a = usub_sat x, y`.
	OpcodeUsubSat

	// OpcodeSsubSat ...
	// `a = ssub_sat x, y`.
	OpcodeSsubSat

	// OpcodeLoad ...
	// `a = load MemFlags, p, Offset`.
	OpcodeLoad

	// OpcodeStore ...
	// `store MemFlags, x, p, Offset`.
	OpcodeStore

	// OpcodeUload8 ...
	// `a = uload8 MemFlags, p, Offset`.
	OpcodeUload8

	// OpcodeSload8 ...
	// `a = sload8 MemFlags, p, Offset`.
	OpcodeSload8

	// OpcodeIstore8 ...
	// `istore8 MemFlags, x, p, Offset`.
	OpcodeIstore8

	// OpcodeUload16 ...
	// `a = uload16 MemFlags, p, Offset`.
	OpcodeUload16

	// OpcodeSload16 ...
	// `a = sload16 MemFlags, p, Offset`.
	OpcodeSload16

	// OpcodeIstore16 ...
	// `istore16 MemFlags, x, p, Offset`.
	OpcodeIstore16

	// OpcodeUload32 ...
	// `a = uload32 MemFlags, p, Offset`.
	OpcodeUload32

	// OpcodeSload32 ...
	// `a = sload32 MemFlags, p, Offset`.
	OpcodeSload32

	// OpcodeIstore32 ...
	// `istore32 MemFlags, x, p, Offset`.
	OpcodeIstore32

	// OpcodeUload8x8 ...
	// `a = uload8x8 MemFlags, p, Offset`.
	OpcodeUload8x8

	// OpcodeSload8x8 ...
	// `a = sload8x8 MemFlags, p, Offset`.
	OpcodeSload8x8

	// OpcodeUload16x4 ...
	// `a = uload16x4 MemFlags, p, Offset`.
	OpcodeUload16x4

	// OpcodeSload16x4 ...
	// `a = sload16x4 MemFlags, p, Offset`.
	OpcodeSload16x4

	// OpcodeUload32x2 ...
	// `a = uload32x2 MemFlags, p, Offset`.
	OpcodeUload32x2

	// OpcodeSload32x2 ...
	// `a = sload32x2 MemFlags, p, Offset`.
	OpcodeSload32x2

	// OpcodeGlobalValue ...
	// `a = global_value GV`.
	OpcodeGlobalValue

	// OpcodeSymbolValue ...
	// `a = symbol_value GV`.
	OpcodeSymbolValue

	// OpcodeHeapAddr ...
	// `addr = heap_addr H, index, Offset, Size`.
	OpcodeHeapAddr

	// OpcodeHeapLoad ...
	// `a = heap_load heap_imm, index`.
	OpcodeHeapLoad

	// OpcodeHeapStore ...
	// `heap_store heap_imm, index, a`.
	OpcodeHeapStore

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
	OpcodeSelect

	// OpcodeSelectSpectreGuard ...
	// `a = select_spectre_guard c, x, y`.
	OpcodeSelectSpectreGuard

	// OpcodeBitselect ...
	// `a = bitselect c, x, y`.
	OpcodeBitselect

	// OpcodeVsplit ...
	// `lo, hi = vsplit x`.
	OpcodeVsplit

	// OpcodeVconcat ...
	// `a = vconcat x, y`.
	OpcodeVconcat

	// OpcodeVselect ...
	// `a = vselect c, x, y`.
	OpcodeVselect

	// OpcodeVanyTrue ...
	// `s = vany_true a`.
	OpcodeVanyTrue

	// OpcodeVallTrue ...
	// `s = vall_true a`.
	OpcodeVallTrue

	// OpcodeVhighBits ...
	// `x = vhigh_bits a`.
	OpcodeVhighBits

	// OpcodeIcmp ...
	// `a = icmp Cond, x, y`.
	OpcodeIcmp

	// OpcodeIcmpImm ...
	// `a = icmp_imm Cond, x, Y`.
	OpcodeIcmpImm

	// OpcodeIfcmp ...
	// `f = ifcmp x, y`.
	OpcodeIfcmp

	// OpcodeIfcmpImm ...
	// `f = ifcmp_imm x, Y`. (BinaryImm64)
	OpcodeIfcmpImm

	// OpcodeIadd performs an integer addition.
	// `a = iadd x, y`.
	OpcodeIadd

	// OpcodeIsub ...
	// `a = isub x, y`.
	OpcodeIsub

	// OpcodeIneg ...
	// `a = ineg x`.
	OpcodeIneg

	// OpcodeIabs ...
	// `a = iabs x`.
	OpcodeIabs

	// OpcodeImul ...
	// `a = imul x, y`.
	OpcodeImul

	// OpcodeUmulhi ...
	// `a = umulhi x, y`.
	OpcodeUmulhi

	// OpcodeSmulhi ...
	// `a = smulhi x, y`.
	OpcodeSmulhi

	// OpcodeSqmulRoundSat ...
	// `a = sqmul_round_sat x, y`.
	OpcodeSqmulRoundSat

	// OpcodeUdiv ...
	// `a = udiv x, y`.
	OpcodeUdiv

	// OpcodeSdiv ...
	// `a = sdiv x, y`.
	OpcodeSdiv

	// OpcodeUrem ...
	// `a = urem x, y`.
	OpcodeUrem

	// OpcodeSrem ...
	// `a = srem x, y`.
	OpcodeSrem

	// OpcodeIaddImm ...
	// `a = iadd_imm x, Y`. (BinaryImm64)
	OpcodeIaddImm

	// OpcodeImulImm ...
	// `a = imul_imm x, Y`. (BinaryImm64)
	OpcodeImulImm

	// OpcodeUdivImm ...
	// `a = udiv_imm x, Y`. (BinaryImm64)
	OpcodeUdivImm

	// OpcodeSdivImm ...
	// `a = sdiv_imm x, Y`. (BinaryImm64)
	OpcodeSdivImm

	// OpcodeUremImm ...
	// `a = urem_imm x, Y`. (BinaryImm64)
	OpcodeUremImm

	// OpcodeSremImm ...
	// `a = srem_imm x, Y`. (BinaryImm64)
	OpcodeSremImm

	// OpcodeIrsubImm ...
	// `a = irsub_imm x, Y`. (BinaryImm64)
	OpcodeIrsubImm

	// OpcodeIaddCin ...
	// `a = iadd_cin x, y, c_in`.
	OpcodeIaddCin

	// OpcodeIaddIfcin ...
	// `a = iadd_ifcin x, y, c_in`.
	OpcodeIaddIfcin

	// OpcodeIaddCout ...
	// `a, c_out = iadd_cout x, y`.
	OpcodeIaddCout

	// OpcodeIaddIfcout ...
	// `a, c_out = iadd_ifcout x, y`.
	OpcodeIaddIfcout

	// OpcodeIaddCarry ...
	// `a, c_out = iadd_carry x, y, c_in`.
	OpcodeIaddCarry

	// OpcodeIaddIfcarry ...
	// `a, c_out = iadd_ifcarry x, y, c_in`.
	OpcodeIaddIfcarry

	// OpcodeUaddOverflowTrap ...
	// `a = uadd_overflow_trap x, y, code`.
	OpcodeUaddOverflowTrap

	// OpcodeIsubBin ...
	// `a = isub_bin x, y, b_in`.
	OpcodeIsubBin

	// OpcodeIsubIfbin ...
	// `a = isub_ifbin x, y, b_in`.
	OpcodeIsubIfbin

	// OpcodeIsubBout ...
	// `a, b_out = isub_bout x, y`.
	OpcodeIsubBout

	// OpcodeIsubIfbout ...
	// `a, b_out = isub_ifbout x, y`.
	OpcodeIsubIfbout

	// OpcodeIsubBorrow ...
	// `a, b_out = isub_borrow x, y, b_in`.
	OpcodeIsubBorrow

	// OpcodeIsubIfborrow ...
	// `a, b_out = isub_ifborrow x, y, b_in`.
	OpcodeIsubIfborrow

	// OpcodeBand ...
	// `a = band x, y`.
	OpcodeBand

	// OpcodeBor ...
	// `a = bor x, y`.
	OpcodeBor

	// OpcodeBxor ...
	// `a = bxor x, y`.
	OpcodeBxor

	// OpcodeBnot ...
	// `a = bnot x`.
	OpcodeBnot

	// OpcodeBandNot ...
	// `a = band_not x, y`.
	OpcodeBandNot

	// OpcodeBorNot ...
	// `a = bor_not x, y`.
	OpcodeBorNot

	// OpcodeBxorNot ...
	// `a = bxor_not x, y`.
	OpcodeBxorNot

	// OpcodeBandImm ...
	// `a = band_imm x, Y`. (BinaryImm64)
	OpcodeBandImm

	// OpcodeBorImm ...
	// `a = bor_imm x, Y`. (BinaryImm64)
	OpcodeBorImm

	// OpcodeBxorImm ...
	// `a = bxor_imm x, Y`. (BinaryImm64)
	OpcodeBxorImm

	// OpcodeRotl ...
	// `a = rotl x, y`.
	OpcodeRotl

	// OpcodeRotr ...
	// `a = rotr x, y`.
	OpcodeRotr

	// OpcodeRotlImm ...
	// `a = rotl_imm x, Y`. (BinaryImm64)
	OpcodeRotlImm

	// OpcodeRotrImm ...
	// `a = rotr_imm x, Y`. (BinaryImm64)
	OpcodeRotrImm

	// OpcodeIshl ...
	// `a = ishl x, y`.
	OpcodeIshl

	// OpcodeUshr ...
	// `a = ushr x, y`.
	OpcodeUshr

	// OpcodeSshr ...
	// `a = sshr x, y`.
	OpcodeSshr

	// OpcodeIshlImm ...
	// `a = ishl_imm x, Y`. (BinaryImm64)
	OpcodeIshlImm

	// OpcodeUshrImm ...
	// `a = ushr_imm x, Y`. (BinaryImm64)
	OpcodeUshrImm

	// OpcodeSshrImm ...
	// `a = sshr_imm x, Y`. (BinaryImm64)
	OpcodeSshrImm

	// OpcodeBitrev ...
	// `a = bitrev x`.
	OpcodeBitrev

	// OpcodeClz ...
	// `a = clz x`.
	OpcodeClz

	// OpcodeCls ...
	// `a = cls x`.
	OpcodeCls

	// OpcodeCtz ...
	// `a = ctz x`.
	OpcodeCtz

	// OpcodeBswap ...
	// `a = bswap x`.
	OpcodeBswap

	// OpcodePopcnt ...
	// `a = popcnt x`.
	OpcodePopcnt

	// OpcodeFcmp ...
	// `a = fcmp Cond, x, y`.
	OpcodeFcmp

	// OpcodeFfcmp ...
	// `f = ffcmp x, y`.
	OpcodeFfcmp

	// OpcodeFadd ...
	// `a = fadd x, y`.
	OpcodeFadd

	// OpcodeFsub ...
	// `a = fsub x, y`.
	OpcodeFsub

	// OpcodeFmul ...
	// `a = fmul x, y`.
	OpcodeFmul

	// OpcodeFdiv ...
	// `a = fdiv x, y`.
	OpcodeFdiv

	// OpcodeSqrt ...
	// `a = sqrt x`.
	OpcodeSqrt

	// OpcodeFma ...
	// `a = fma x, y, z`.
	OpcodeFma

	// OpcodeFneg ...
	// `a = fneg x`.
	OpcodeFneg

	// OpcodeFabs ...
	// `a = fabs x`.
	OpcodeFabs

	// OpcodeFcopysign ...
	// `a = fcopysign x, y`.
	OpcodeFcopysign

	// OpcodeFmin ...
	// `a = fmin x, y`.
	OpcodeFmin

	// OpcodeFminPseudo ...
	// `a = fmin_pseudo x, y`.
	OpcodeFminPseudo

	// OpcodeFmax ...
	// `a = fmax x, y`.
	OpcodeFmax

	// OpcodeFmaxPseudo ...
	// `a = fmax_pseudo x, y`.
	OpcodeFmaxPseudo

	// OpcodeCeil ...
	// `a = ceil x`.
	OpcodeCeil

	// OpcodeFloor ...
	// `a = floor x`.
	OpcodeFloor

	// OpcodeTrunc ...
	// `a = trunc x`.
	OpcodeTrunc

	// OpcodeNearest ...
	// `a = nearest x`.
	OpcodeNearest

	// OpcodeIsNull ...
	// `a = is_null x`.
	OpcodeIsNull

	// OpcodeIsInvalid ...
	// `a = is_invalid x`.
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
	OpcodeSnarrow
	// `a = unarrow x, y`.

	// OpcodeUnarrow ...
	OpcodeUnarrow
	// `a = uunarrow x, y`.

	// OpcodeUunarrow ...
	OpcodeUunarrow
	// `a = swiden_low x`.

	// OpcodeSwidenLow ...
	OpcodeSwidenLow
	// `a = swiden_high x`.

	// OpcodeSwidenHigh ...
	OpcodeSwidenHigh
	// `a = uwiden_low x`.

	// OpcodeUwidenLow ...
	OpcodeUwidenLow
	// `a = uwiden_high x`.

	// OpcodeUwidenHigh ...
	OpcodeUwidenHigh
	// `a = iadd_pairwise x, y`.

	// OpcodeIaddPairwise ...
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
	OpcodeIsplit

	// OpcodeIconcat ...
	// `a = iconcat lo, hi`.
	OpcodeIconcat

	// OpcodeAtomicRmw ...
	// `a = atomic_rmw MemFlags, AtomicRmwOp, p, x`.
	OpcodeAtomicRmw

	// OpcodeAtomicCas ...
	// `a = atomic_cas MemFlags, p, e, x`.
	OpcodeAtomicCas

	// OpcodeAtomicLoad ...
	// `a = atomic_load MemFlags, p`.
	OpcodeAtomicLoad

	// OpcodeAtomicStore ...
	// `atomic_store MemFlags, x, p`.
	OpcodeAtomicStore

	// OpcodeFence ...
	// `fence`.
	OpcodeFence

	// OpcodeExtractVector ...
	// `a = extract_vector x, y`. (BinaryImm8)
	OpcodeExtractVector

	OpcodeAlias

	opcodeEnd
)

type returnTypesFn func(b *builder, instr *Instruction) (t1 Type, ts []Type)

var (
	returnTypesFnNoReturns returnTypesFn = func(b *builder, instr *Instruction) (t1 Type, ts []Type) { return TypeInvalid, nil }
	returnTypesFnF32                     = func(b *builder, instr *Instruction) (t1 Type, ts []Type) { return TypeF32, nil }
	returnTypesFnF64                     = func(b *builder, instr *Instruction) (t1 Type, ts []Type) { return TypeF64, nil }
)

var instructionReturnTypes = [...]returnTypesFn{
	OpcodeJump:   returnTypesFnNoReturns,
	OpcodeIconst: func(b *builder, instr *Instruction) (t1 Type, ts []Type) { return instr.typ, nil },
	OpcodeCall: func(b *builder, instr *Instruction) (t1 Type, ts []Type) {
		sigID := SignatureID(instr.v)
		sig, ok := b.signatures[sigID]
		if !ok {
			panic("BUG")
		}
		switch len(sig.Results) {
		case 0:
		case 1:
			t1 = sig.Results[0]
		default:
			t1, ts = sig.Results[0], sig.Results[1:]
		}
		return
	},
	OpcodeF32const: returnTypesFnF32,
	OpcodeF64const: returnTypesFnF64,
	OpcodeStore:    returnTypesFnNoReturns,
	OpcodeTrap:     returnTypesFnNoReturns,
	OpcodeReturn:   returnTypesFnNoReturns,
	OpcodeBrz:      returnTypesFnNoReturns,
	opcodeEnd:      nil,
}

func (i *Instruction) AsStore(value, ptr Value, offset uint32) {
	i.opcode = OpcodeStore
	i.typ = TypeI64
	i.v = value
	i.v2 = ptr
	i.u64 = uint64(offset)
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

func (i *Instruction) AsReturn(vs []Value) {
	i.opcode = OpcodeReturn
	i.vs = vs
}

func (i *Instruction) AsTrap() {
	i.opcode = OpcodeTrap
}

func (i *Instruction) AsJump(vs []Value, target BasicBlock) {
	i.opcode = OpcodeJump
	i.vs = vs
	i.blk = target
}

func (i *Instruction) AsBrz(v Value, args []Value, target BasicBlock) {
	i.opcode = OpcodeBrz
	i.v = v
	i.vs = args
	i.blk = target
}

func (i *Instruction) AsBrnz(v Value, args []Value, target BasicBlock) {
	i.opcode = OpcodeBrnz
	i.v = v
	i.vs = args
	i.blk = target
}

func (i *Instruction) AsCall(ref FuncRef, sig *Signature, args []Value) {
	i.opcode = OpcodeCall
	i.typ = TypeF64
	i.u64 = uint64(ref)
	i.vs = args
	i.v = Value(sig.ID)
	sig.used = true
}

func (i *Instruction) Format(b Builder) string {
	var instSuffix string
	switch i.opcode {
	case OpcodeTrap:
	case OpcodeCall:
		vs := make([]string, len(i.vs))
		for idx := range vs {
			vs[idx] = i.vs[idx].format(b)
		}
		instSuffix = fmt.Sprintf(" %s:%s, %s", FuncRef(i.u64), SignatureID(i.v), strings.Join(vs, ", "))
	case OpcodeStore:
		instSuffix = fmt.Sprintf(" %s, %s, %#x", i.v.format(b), i.v2.format(b), int32(i.u64))
	case OpcodeIconst:
		switch i.typ {
		case TypeI32:
			instSuffix = fmt.Sprintf("_32 %#x", uint32(i.u64))
		case TypeI64:
			instSuffix = fmt.Sprintf("_64 %#x", i.u64)
		}
	case OpcodeF32const:
		instSuffix = fmt.Sprintf(" %f", math.Float32frombits(uint32(i.u64)))
	case OpcodeF64const:
		instSuffix = fmt.Sprintf(" %f", math.Float64frombits(i.u64))
	case OpcodeReturn:
		if len(i.vs) == 0 {
			break
		}
		vs := make([]string, len(i.vs))
		for idx := range vs {
			vs[idx] = i.vs[idx].format(b)
		}
		instSuffix = fmt.Sprintf(" %s", strings.Join(vs, ", "))
	case OpcodeJump:
		vs := make([]string, len(i.vs)+1)
		vs[0] = " " + i.blk.(*basicBlock).Name()
		for idx := range i.vs {
			vs[idx+1] = i.vs[idx].format(b)
		}

		instSuffix = strings.Join(vs, ", ")
	case OpcodeBrz, OpcodeBrnz:
		vs := make([]string, len(i.vs)+2)
		vs[0] = " " + i.v.format(b)
		vs[1] = i.blk.(*basicBlock).Name()
		for idx := range i.vs {
			vs[idx+2] = i.vs[idx].format(b)
		}
		instSuffix = strings.Join(vs, ", ")
	default:
		panic(fmt.Sprintf("TODO: format for %s", i.opcode))
	}

	instr := i.opcode.String() + instSuffix

	var rvs []string
	if rv := i.rValue; rv.valid() {
		rvs = append(rvs, rv.formatWithType(b))
	}

	for _, v := range i.rValues {
		rvs = append(rvs, v.formatWithType(b))
	}

	if len(rvs) > 0 {
		return fmt.Sprintf("%s = %s", strings.Join(rvs, ", "), instr)
	} else {
		return instr
	}
}

func (i *Instruction) addArgument(v Value) {
	switch i.opcode {
	case OpcodeJump, OpcodeBrz, OpcodeBrnz:
		i.vs = append(i.vs, v)
	default:
		panic("BUG: " + i.typ.String())
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
	case OpcodeTrap:
		return "Trap"
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
	case OpcodeGlobalValue:
		return "GlobalValue"
	case OpcodeSymbolValue:
		return "SymbolValue"
	case OpcodeHeapAddr:
		return "HeapAddr"
	case OpcodeHeapLoad:
		return "HeapLoad"
	case OpcodeHeapStore:
		return "HeapStore"
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
