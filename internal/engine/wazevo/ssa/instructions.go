package ssa

// Opcode represents a SSA instruction.
type Opcode uint32

// Instruction represents an instruction whose opcode is specified by
// Opcode. Since Go doesn't have union type, we use this flattened type
// for all instructions, and therefore each field has different meaning
// depending on Opcode.
//
// Instruction implements Value because some instructions produces values
// and can be used in subsequent instructions as inputs.
type Instruction struct {
	Opcode Opcode

	// TODO: adds fields
}

// Followings match the generated code from https://github.com/bytecodealliance/wasmtime/blob/v9.0.3/cranelift/codegen/meta/src/shared/instructions.rs
// TODO: complete opcode comments.
// TODO: there should be unnecessary opcodes.
const (
	// OpcodeJump ...
	// `jump block, args`. (Jump)
	OpcodeJump Opcode = 1 + iota

	// OpcodeBrz ...
	// `brz c, block, args`. (Branch)
	// Type inferred from `c`.
	OpcodeBrz

	// OpcodeBrnz ...
	// `brnz c, block, args`. (Branch)
	// Type inferred from `c`.
	OpcodeBrnz

	// OpcodeBrTable ...
	// `br_table x, block, JT`. (BranchTable)
	OpcodeBrTable

	// OpcodeDebugtrap ...
	// `debugtrap`. (NullAry)
	OpcodeDebugtrap

	// OpcodeTrap ...
	// `trap code`. (Trap)
	OpcodeTrap

	// OpcodeTrapz ...
	// `trapz c, code`. (CondTrap)
	// Type inferred from `c`.
	OpcodeTrapz

	// OpcodeResumableTrap ...
	// `resumable_trap code`. (Trap)
	OpcodeResumableTrap

	// OpcodeTrapnz ...
	// `trapnz c, code`. (CondTrap)
	// Type inferred from `c`.
	OpcodeTrapnz

	// OpcodeResumableTrapnz ...
	// `resumable_trapnz c, code`. (CondTrap)
	// Type inferred from `c`.
	OpcodeResumableTrapnz

	// OpcodeReturn ...
	// `return rvals`. (MultiAry)
	OpcodeReturn

	// OpcodeCall ...
	// `rvals = call FN, args`. (Call)
	OpcodeCall

	// OpcodeCallIndirect ...
	// `rvals = call_indirect SIG, callee, args`. (CallIndirect)
	// Type inferred from `callee`.
	OpcodeCallIndirect

	// OpcodeFuncAddr ...
	// `addr = func_addr FN`. (FuncAddr)
	OpcodeFuncAddr

	// OpcodeSplat ...
	// `a = splat x`. (Unary)
	OpcodeSplat

	// OpcodeSwizzle ...
	// `a = swizzle x, y`. (Binary)
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
	// `a = smin x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeSmin

	// OpcodeUmin ...
	// `a = umin x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeUmin

	// OpcodeSmax ...
	// `a = smax x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeSmax

	// OpcodeUmax ...
	// `a = umax x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeUmax

	// OpcodeAvgRound ...
	// `a = avg_round x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeAvgRound

	// OpcodeUaddSat ...
	// `a = uadd_sat x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeUaddSat

	// OpcodeSaddSat ...
	// `a = sadd_sat x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeSaddSat

	// OpcodeUsubSat ...
	// `a = usub_sat x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeUsubSat

	// OpcodeSsubSat ...
	// `a = ssub_sat x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeSsubSat

	// OpcodeLoad ...
	// `a = load MemFlags, p, Offset`. (Load)
	OpcodeLoad

	// OpcodeStore ...
	// `store MemFlags, x, p, Offset`. (Store)
	// Type inferred from `x`.
	OpcodeStore

	// OpcodeUload8 ...
	// `a = uload8 MemFlags, p, Offset`. (Load)
	OpcodeUload8

	// OpcodeSload8 ...
	// `a = sload8 MemFlags, p, Offset`. (Load)
	OpcodeSload8

	// OpcodeIstore8 ...
	// `istore8 MemFlags, x, p, Offset`. (Store)
	// Type inferred from `x`.
	OpcodeIstore8

	// OpcodeUload16 ...
	// `a = uload16 MemFlags, p, Offset`. (Load)
	OpcodeUload16

	// OpcodeSload16 ...
	// `a = sload16 MemFlags, p, Offset`. (Load)
	OpcodeSload16

	// OpcodeIstore16 ...
	// `istore16 MemFlags, x, p, Offset`. (Store)
	// Type inferred from `x`.
	OpcodeIstore16

	// OpcodeUload32 ...
	// `a = uload32 MemFlags, p, Offset`. (Load)
	// Type inferred from `p`.
	OpcodeUload32

	// OpcodeSload32 ...
	// `a = sload32 MemFlags, p, Offset`. (Load)
	// Type inferred from `p`.
	OpcodeSload32

	// OpcodeIstore32 ...
	// `istore32 MemFlags, x, p, Offset`. (Store)
	// Type inferred from `x`.
	OpcodeIstore32

	// OpcodeUload8x8 ...
	// `a = uload8x8 MemFlags, p, Offset`. (Load)
	// Type inferred from `p`.
	OpcodeUload8x8

	// OpcodeSload8x8 ...
	// `a = sload8x8 MemFlags, p, Offset`. (Load)
	// Type inferred from `p`.
	OpcodeSload8x8

	// OpcodeUload16x4 ...
	// `a = uload16x4 MemFlags, p, Offset`. (Load)
	// Type inferred from `p`.
	OpcodeUload16x4

	// OpcodeSload16x4 ...
	// `a = sload16x4 MemFlags, p, Offset`. (Load)
	// Type inferred from `p`.
	OpcodeSload16x4

	// OpcodeUload32x2 ...
	// `a = uload32x2 MemFlags, p, Offset`. (Load)
	// Type inferred from `p`.
	OpcodeUload32x2

	// OpcodeSload32x2 ...
	// `a = sload32x2 MemFlags, p, Offset`. (Load)
	// Type inferred from `p`.
	OpcodeSload32x2

	// OpcodeStackLoad ...
	// `a = stack_load SS, Offset`. (StackLoad)
	OpcodeStackLoad

	// OpcodeStackStore ...
	// `stack_store x, SS, Offset`. (StackStore)
	// Type inferred from `x`.
	OpcodeStackStore

	// OpcodeStackAddr ...
	// `addr = stack_addr SS, Offset`. (StackLoad)
	OpcodeStackAddr

	// OpcodeDynamicStackLoad ...
	// `a = dynamic_stack_load DSS`. (DynamicStackLoad)
	OpcodeDynamicStackLoad

	// OpcodeDynamicStackStore ...
	// `dynamic_stack_store x, DSS`. (DynamicStackStore)
	// Type inferred from `x`.
	OpcodeDynamicStackStore

	// OpcodeDynamicStackAddr ...
	// `addr = dynamic_stack_addr DSS`. (DynamicStackLoad)
	OpcodeDynamicStackAddr

	// OpcodeGlobalValue ...
	// `a = global_value GV`. (UnaryGlobalValue)
	OpcodeGlobalValue

	// OpcodeSymbolValue ...
	// `a = symbol_value GV`. (UnaryGlobalValue)
	OpcodeSymbolValue

	// OpcodeTlsValue ...
	// `a = tls_value GV`. (UnaryGlobalValue)
	OpcodeTlsValue

	// OpcodeHeapAddr ...
	// `addr = heap_addr H, index, Offset, Size`. (HeapAddr)
	OpcodeHeapAddr

	// OpcodeHeapLoad ...
	// `a = heap_load heap_imm, index`. (HeapLoad)
	OpcodeHeapLoad

	// OpcodeHeapStore ...
	// `heap_store heap_imm, index, a`. (HeapStore)
	// Type inferred from `index`.
	OpcodeHeapStore

	// OpcodeGetPinnedReg ...
	// `addr = get_pinned_reg`. (NullAry)
	OpcodeGetPinnedReg

	// OpcodeSetPinnedReg ...
	// `set_pinned_reg addr`. (Unary)
	// Type inferred from `addr`.
	OpcodeSetPinnedReg

	// OpcodeGetFramePointer ...
	// `addr = get_frame_pointer`. (NullAry)
	OpcodeGetFramePointer

	// OpcodeGetStackPointer ...
	// `addr = get_stack_pointer`. (NullAry)
	OpcodeGetStackPointer

	// OpcodeGetReturnAddress ...
	// `addr = get_return_address`. (NullAry)
	OpcodeGetReturnAddress

	// OpcodeTableAddr ...
	// `addr = table_addr T, p, Offset`. (TableAddr)
	OpcodeTableAddr

	// OpcodeIconst ...
	// `a = iconst N`. (UnaryImm)
	OpcodeIconst

	// OpcodeF32const ...
	// `a = f32const N`. (UnaryIeee32)
	OpcodeF32const

	// OpcodeF64const ...
	// `a = f64const N`. (UnaryIeee64)
	OpcodeF64const

	// OpcodeVconst ...
	// `a = vconst N`. (UnaryConst)
	OpcodeVconst

	// OpcodeShuffle ...
	// `a = shuffle a, b, mask`. (Shuffle)
	OpcodeShuffle

	// OpcodeNull ...
	// `a = null`. (NullAry)
	OpcodeNull

	// OpcodeNop ...
	// `nop`. (NullAry)
	OpcodeNop

	// OpcodeSelect ...
	// `a = select c, x, y`. (Ternary)
	// Type inferred from `x`.
	OpcodeSelect

	// OpcodeSelectSpectreGuard ...
	// `a = select_spectre_guard c, x, y`. (Ternary)
	// Type inferred from `x`.
	OpcodeSelectSpectreGuard

	// OpcodeBitselect ...
	// `a = bitselect c, x, y`. (Ternary)
	// Type inferred from `x`.
	OpcodeBitselect

	// OpcodeVsplit ...
	// `lo, hi = vsplit x`. (Unary)
	// Type inferred from `x`.
	OpcodeVsplit

	// OpcodeVconcat ...
	// `a = vconcat x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeVconcat

	// OpcodeVselect ...
	// `a = vselect c, x, y`. (Ternary)
	// Type inferred from `x`.
	OpcodeVselect

	// OpcodeVanyTrue ...
	// `s = vany_true a`. (Unary)
	// Type inferred from `a`.
	OpcodeVanyTrue

	// OpcodeVallTrue ...
	// `s = vall_true a`. (Unary)
	// Type inferred from `a`.
	OpcodeVallTrue

	// OpcodeVhighBits ...
	// `x = vhigh_bits a`. (Unary)
	OpcodeVhighBits

	// OpcodeIcmp ...
	// `a = icmp Cond, x, y`. (IntCompare)
	// Type inferred from `x`.
	OpcodeIcmp

	// OpcodeIcmpImm ...
	// `a = icmp_imm Cond, x, Y`. (IntCompareImm)
	// Type inferred from `x`.
	OpcodeIcmpImm

	// OpcodeIfcmp ...
	// `f = ifcmp x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeIfcmp

	// OpcodeIfcmpImm ...
	// `f = ifcmp_imm x, Y`. (BinaryImm64)
	// Type inferred from `x`.
	OpcodeIfcmpImm

	// OpcodeIadd ...
	// `a = iadd x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeIadd

	// OpcodeIsub ...
	// `a = isub x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeIsub

	// OpcodeIneg ...
	// `a = ineg x`. (Unary)
	// Type inferred from `x`.
	OpcodeIneg

	// OpcodeIabs ...
	// `a = iabs x`. (Unary)
	// Type inferred from `x`.
	OpcodeIabs

	// OpcodeImul ...
	// `a = imul x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeImul

	// OpcodeUmulhi ...
	// `a = umulhi x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeUmulhi

	// OpcodeSmulhi ...
	// `a = smulhi x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeSmulhi

	// OpcodeSqmulRoundSat ...
	// `a = sqmul_round_sat x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeSqmulRoundSat

	// OpcodeUdiv ...
	// `a = udiv x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeUdiv

	// OpcodeSdiv ...
	// `a = sdiv x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeSdiv

	// OpcodeUrem ...
	// `a = urem x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeUrem

	// OpcodeSrem ...
	// `a = srem x, y`. (Binary)
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
	// `a = iadd_cin x, y, c_in`. (Ternary)
	// Type inferred from `y`.
	OpcodeIaddCin

	// OpcodeIaddIfcin ...
	// `a = iadd_ifcin x, y, c_in`. (Ternary)
	// Type inferred from `y`.
	OpcodeIaddIfcin

	// OpcodeIaddCout ...
	// `a, c_out = iadd_cout x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeIaddCout

	// OpcodeIaddIfcout ...
	// `a, c_out = iadd_ifcout x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeIaddIfcout

	// OpcodeIaddCarry ...
	// `a, c_out = iadd_carry x, y, c_in`. (Ternary)
	// Type inferred from `y`.
	OpcodeIaddCarry

	// OpcodeIaddIfcarry ...
	// `a, c_out = iadd_ifcarry x, y, c_in`. (Ternary)
	// Type inferred from `y`.
	OpcodeIaddIfcarry

	// OpcodeUaddOverflowTrap ...
	// `a = uadd_overflow_trap x, y, code`. (IntAddTrap)
	// Type inferred from `x`.
	OpcodeUaddOverflowTrap

	// OpcodeIsubBin ...
	// `a = isub_bin x, y, b_in`. (Ternary)
	// Type inferred from `y`.
	OpcodeIsubBin

	// OpcodeIsubIfbin ...
	// `a = isub_ifbin x, y, b_in`. (Ternary)
	// Type inferred from `y`.
	OpcodeIsubIfbin

	// OpcodeIsubBout ...
	// `a, b_out = isub_bout x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeIsubBout

	// OpcodeIsubIfbout ...
	// `a, b_out = isub_ifbout x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeIsubIfbout

	// OpcodeIsubBorrow ...
	// `a, b_out = isub_borrow x, y, b_in`. (Ternary)
	// Type inferred from `y`.
	OpcodeIsubBorrow

	// OpcodeIsubIfborrow ...
	// `a, b_out = isub_ifborrow x, y, b_in`. (Ternary)
	// Type inferred from `y`.
	OpcodeIsubIfborrow

	// OpcodeBand ...
	// `a = band x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeBand

	// OpcodeBor ...
	// `a = bor x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeBor

	// OpcodeBxor ...
	// `a = bxor x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeBxor

	// OpcodeBnot ...
	// `a = bnot x`. (Unary)
	// Type inferred from `x`.
	OpcodeBnot

	// OpcodeBandNot ...
	// `a = band_not x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeBandNot

	// OpcodeBorNot ...
	// `a = bor_not x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeBorNot

	// OpcodeBxorNot ...
	// `a = bxor_not x, y`. (Binary)
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
	// `a = rotl x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeRotl

	// OpcodeRotr ...
	// `a = rotr x, y`. (Binary)
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
	// `a = ishl x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeIshl

	// OpcodeUshr ...
	// `a = ushr x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeUshr

	// OpcodeSshr ...
	// `a = sshr x, y`. (Binary)
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
	// `a = bitrev x`. (Unary)
	// Type inferred from `x`.
	OpcodeBitrev

	// OpcodeClz ...
	// `a = clz x`. (Unary)
	// Type inferred from `x`.
	OpcodeClz

	// OpcodeCls ...
	// `a = cls x`. (Unary)
	// Type inferred from `x`.
	OpcodeCls

	// OpcodeCtz ...
	// `a = ctz x`. (Unary)
	// Type inferred from `x`.
	OpcodeCtz

	// OpcodeBswap ...
	// `a = bswap x`. (Unary)
	// Type inferred from `x`.
	OpcodeBswap

	// OpcodePopcnt ...
	// `a = popcnt x`. (Unary)
	// Type inferred from `x`.
	OpcodePopcnt

	// OpcodeFcmp ...
	// `a = fcmp Cond, x, y`. (FloatCompare)
	// Type inferred from `x`.
	OpcodeFcmp

	// OpcodeFfcmp ...
	// `f = ffcmp x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeFfcmp

	// OpcodeFadd ...
	// `a = fadd x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeFadd

	// OpcodeFsub ...
	// `a = fsub x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeFsub

	// OpcodeFmul ...
	// `a = fmul x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeFmul

	// OpcodeFdiv ...
	// `a = fdiv x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeFdiv

	// OpcodeSqrt ...
	// `a = sqrt x`. (Unary)
	// Type inferred from `x`.
	OpcodeSqrt

	// OpcodeFma ...
	// `a = fma x, y, z`. (Ternary)
	// Type inferred from `y`.
	OpcodeFma

	// OpcodeFneg ...
	// `a = fneg x`. (Unary)
	// Type inferred from `x`.
	OpcodeFneg

	// OpcodeFabs ...
	// `a = fabs x`. (Unary)
	// Type inferred from `x`.
	OpcodeFabs

	// OpcodeFcopysign ...
	// `a = fcopysign x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeFcopysign

	// OpcodeFmin ...
	// `a = fmin x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeFmin

	// OpcodeFminPseudo ...
	// `a = fmin_pseudo x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeFminPseudo

	// OpcodeFmax ...
	// `a = fmax x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeFmax

	// OpcodeFmaxPseudo ...
	// `a = fmax_pseudo x, y`. (Binary)
	// Type inferred from `x`.
	OpcodeFmaxPseudo

	// OpcodeCeil ...
	// `a = ceil x`. (Unary)
	// Type inferred from `x`.
	OpcodeCeil

	// OpcodeFloor ...
	// `a = floor x`. (Unary)
	// Type inferred from `x`.
	OpcodeFloor

	// OpcodeTrunc ...
	// `a = trunc x`. (Unary)
	// Type inferred from `x`.
	OpcodeTrunc

	// OpcodeNearest ...
	// `a = nearest x`. (Unary)
	// Type inferred from `x`.
	OpcodeNearest

	// OpcodeIsNull ...
	// `a = is_null x`. (Unary)
	// Type inferred from `x`.
	OpcodeIsNull

	// OpcodeIsInvalid ...
	// `a = is_invalid x`. (Unary)
	// Type inferred from `x`.
	OpcodeIsInvalid

	// OpcodeBitcast ...
	// `a = bitcast MemFlags, x`. (LoadNoOffset)
	OpcodeBitcast

	// OpcodeScalarToVector ...
	// `a = scalar_to_vector s`. (Unary)
	OpcodeScalarToVector

	// OpcodeBmask ...
	// `a = bmask x`. (Unary)
	OpcodeBmask

	// OpcodeIreduce ...
	// `a = ireduce x`. (Unary)
	OpcodeIreduce
	// `a = snarrow x, y`. (Binary)

	// OpcodeSnarrow ...
	// Type inferred from `x`.
	OpcodeSnarrow
	// `a = unarrow x, y`. (Binary)

	// OpcodeUnarrow ...
	// Type inferred from `x`.
	OpcodeUnarrow
	// `a = uunarrow x, y`. (Binary)

	// OpcodeUunarrow ...
	// Type inferred from `x`.
	OpcodeUunarrow
	// `a = swiden_low x`. (Unary)

	// OpcodeSwidenLow ...
	// Type inferred from `x`.
	OpcodeSwidenLow
	// `a = swiden_high x`. (Unary)

	// OpcodeSwidenHigh ...
	// Type inferred from `x`.
	OpcodeSwidenHigh
	// `a = uwiden_low x`. (Unary)

	// OpcodeUwidenLow ...
	// Type inferred from `x`.
	OpcodeUwidenLow
	// `a = uwiden_high x`. (Unary)

	// OpcodeUwidenHigh ...
	// Type inferred from `x`.
	OpcodeUwidenHigh
	// `a = iadd_pairwise x, y`. (Binary)

	// OpcodeIaddPairwise ...
	// Type inferred from `x`.
	OpcodeIaddPairwise

	// OpcodeWideningPairwiseDotProductS ...
	// `a = widening_pairwise_dot_product_s x, y`. (Binary)
	OpcodeWideningPairwiseDotProductS

	// OpcodeUextend ...
	// `a = uextend x`. (Unary)
	OpcodeUextend

	// OpcodeSextend ...
	// `a = sextend x`. (Unary)
	OpcodeSextend

	// OpcodeFpromote ...
	// `a = fpromote x`. (Unary)
	OpcodeFpromote

	// OpcodeFdemote ...
	// `a = fdemote x`. (Unary)
	OpcodeFdemote

	// OpcodeFvdemote ...
	// `a = fvdemote x`. (Unary)
	OpcodeFvdemote

	// OpcodeFvpromoteLow ...
	// `x = fvpromote_low a`. (Unary)
	OpcodeFvpromoteLow

	// OpcodeFcvtToUint ...
	// `a = fcvt_to_uint x`. (Unary)
	OpcodeFcvtToUint

	// OpcodeFcvtToSint ...
	// `a = fcvt_to_sint x`. (Unary)
	OpcodeFcvtToSint

	// OpcodeFcvtToUintSat ...
	// `a = fcvt_to_uint_sat x`. (Unary)
	OpcodeFcvtToUintSat

	// OpcodeFcvtToSintSat ...
	// `a = fcvt_to_sint_sat x`. (Unary)
	OpcodeFcvtToSintSat

	// OpcodeFcvtFromUint ...
	// `a = fcvt_from_uint x`. (Unary)
	OpcodeFcvtFromUint

	// OpcodeFcvtFromSint ...
	// `a = fcvt_from_sint x`. (Unary)
	OpcodeFcvtFromSint

	// OpcodeFcvtLowFromSint ...
	// `a = fcvt_low_from_sint x`. (Unary)
	OpcodeFcvtLowFromSint

	// OpcodeIsplit ...
	// `lo, hi = isplit x`. (Unary)
	// Type inferred from `x`.
	OpcodeIsplit

	// OpcodeIconcat ...
	// `a = iconcat lo, hi`. (Binary)
	// Type inferred from `lo`.
	OpcodeIconcat

	// OpcodeAtomicRmw ...
	// `a = atomic_rmw MemFlags, AtomicRmwOp, p, x`. (AtomicRmw)
	OpcodeAtomicRmw

	// OpcodeAtomicCas ...
	// `a = atomic_cas MemFlags, p, e, x`. (AtomicCas)
	// Type inferred from `x`.
	OpcodeAtomicCas

	// OpcodeAtomicLoad ...
	// `a = atomic_load MemFlags, p`. (LoadNoOffset)
	OpcodeAtomicLoad

	// OpcodeAtomicStore ...
	// `atomic_store MemFlags, x, p`. (StoreNoOffset)
	// Type inferred from `x`.
	OpcodeAtomicStore

	// OpcodeFence ...
	// `fence`. (NullAry)
	OpcodeFence

	// OpcodeExtractVector ...
	// `a = extract_vector x, y`. (BinaryImm8)
	// Type inferred from `x`.
	OpcodeExtractVector
)

// String implements fmt.Stringer.
func (o Instruction) String() (ret string) {
	switch o.Opcode {
	case OpcodeJump:
		panic("TODO: format for OpcodeJump")
	case OpcodeBrz:
		panic("TODO: format for OpcodeBrz")
	case OpcodeBrnz:
		panic("TODO: format for OpcodeBrnz")
	case OpcodeBrTable:
		panic("TODO: format for OpcodeBrTable")
	case OpcodeDebugtrap:
		panic("TODO: format for OpcodeDebugtrap")
	case OpcodeTrap:
		panic("TODO: format for OpcodeTrap")
	case OpcodeTrapz:
		panic("TODO: format for OpcodeTrapz")
	case OpcodeResumableTrap:
		panic("TODO: format for OpcodeResumableTrap")
	case OpcodeTrapnz:
		panic("TODO: format for OpcodeTrapnz")
	case OpcodeResumableTrapnz:
		panic("TODO: format for OpcodeResumableTrapnz")
	case OpcodeReturn:
		panic("TODO: format for OpcodeReturn")
	case OpcodeCall:
		panic("TODO: format for OpcodeCall")
	case OpcodeCallIndirect:
		panic("TODO: format for OpcodeCallIndirect")
	case OpcodeFuncAddr:
		panic("TODO: format for OpcodeFuncAddr")
	case OpcodeSplat:
		panic("TODO: format for OpcodeSplat")
	case OpcodeSwizzle:
		panic("TODO: format for OpcodeSwizzle")
	case OpcodeInsertlane:
		panic("TODO: format for OpcodeInsertlane")
	case OpcodeExtractlane:
		panic("TODO: format for OpcodeExtractlane")
	case OpcodeSmin:
		panic("TODO: format for OpcodeSmin")
	case OpcodeUmin:
		panic("TODO: format for OpcodeUmin")
	case OpcodeSmax:
		panic("TODO: format for OpcodeSmax")
	case OpcodeUmax:
		panic("TODO: format for OpcodeUmax")
	case OpcodeAvgRound:
		panic("TODO: format for OpcodeAvgRound")
	case OpcodeUaddSat:
		panic("TODO: format for OpcodeUaddSat")
	case OpcodeSaddSat:
		panic("TODO: format for OpcodeSaddSat")
	case OpcodeUsubSat:
		panic("TODO: format for OpcodeUsubSat")
	case OpcodeSsubSat:
		panic("TODO: format for OpcodeSsubSat")
	case OpcodeLoad:
		panic("TODO: format for OpcodeLoad")
	case OpcodeStore:
		panic("TODO: format for OpcodeStore")
	case OpcodeUload8:
		panic("TODO: format for OpcodeUload8")
	case OpcodeSload8:
		panic("TODO: format for OpcodeSload8")
	case OpcodeIstore8:
		panic("TODO: format for OpcodeIstore8")
	case OpcodeUload16:
		panic("TODO: format for OpcodeUload16")
	case OpcodeSload16:
		panic("TODO: format for OpcodeSload16")
	case OpcodeIstore16:
		panic("TODO: format for OpcodeIstore16")
	case OpcodeUload32:
		panic("TODO: format for OpcodeUload32")
	case OpcodeSload32:
		panic("TODO: format for OpcodeSload32")
	case OpcodeIstore32:
		panic("TODO: format for OpcodeIstore32")
	case OpcodeUload8x8:
		panic("TODO: format for OpcodeUload8x8")
	case OpcodeSload8x8:
		panic("TODO: format for OpcodeSload8x8")
	case OpcodeUload16x4:
		panic("TODO: format for OpcodeUload16x4")
	case OpcodeSload16x4:
		panic("TODO: format for OpcodeSload16x4")
	case OpcodeUload32x2:
		panic("TODO: format for OpcodeUload32x2")
	case OpcodeSload32x2:
		panic("TODO: format for OpcodeSload32x2")
	case OpcodeStackLoad:
		panic("TODO: format for OpcodeStackLoad")
	case OpcodeStackStore:
		panic("TODO: format for OpcodeStackStore")
	case OpcodeStackAddr:
		panic("TODO: format for OpcodeStackAddr")
	case OpcodeDynamicStackLoad:
		panic("TODO: format for OpcodeDynamicStackLoad")
	case OpcodeDynamicStackStore:
		panic("TODO: format for OpcodeDynamicStackStore")
	case OpcodeDynamicStackAddr:
		panic("TODO: format for OpcodeDynamicStackAddr")
	case OpcodeGlobalValue:
		panic("TODO: format for OpcodeGlobalValue")
	case OpcodeSymbolValue:
		panic("TODO: format for OpcodeSymbolValue")
	case OpcodeTlsValue:
		panic("TODO: format for OpcodeTlsValue")
	case OpcodeHeapAddr:
		panic("TODO: format for OpcodeHeapAddr")
	case OpcodeHeapLoad:
		panic("TODO: format for OpcodeHeapLoad")
	case OpcodeHeapStore:
		panic("TODO: format for OpcodeHeapStore")
	case OpcodeGetPinnedReg:
		panic("TODO: format for OpcodeGetPinnedReg")
	case OpcodeSetPinnedReg:
		panic("TODO: format for OpcodeSetPinnedReg")
	case OpcodeGetFramePointer:
		panic("TODO: format for OpcodeGetFramePointer")
	case OpcodeGetStackPointer:
		panic("TODO: format for OpcodeGetStackPointer")
	case OpcodeGetReturnAddress:
		panic("TODO: format for OpcodeGetReturnAddress")
	case OpcodeTableAddr:
		panic("TODO: format for OpcodeTableAddr")
	case OpcodeIconst:
		panic("TODO: format for OpcodeIconst")
	case OpcodeF32const:
		panic("TODO: format for OpcodeF32const")
	case OpcodeF64const:
		panic("TODO: format for OpcodeF64const")
	case OpcodeVconst:
		panic("TODO: format for OpcodeVconst")
	case OpcodeShuffle:
		panic("TODO: format for OpcodeShuffle")
	case OpcodeNull:
		panic("TODO: format for OpcodeNull")
	case OpcodeNop:
		panic("TODO: format for OpcodeNop")
	case OpcodeSelect:
		panic("TODO: format for OpcodeSelect")
	case OpcodeSelectSpectreGuard:
		panic("TODO: format for OpcodeSelectSpectreGuard")
	case OpcodeBitselect:
		panic("TODO: format for OpcodeBitselect")
	case OpcodeVsplit:
		panic("TODO: format for OpcodeVsplit")
	case OpcodeVconcat:
		panic("TODO: format for OpcodeVconcat")
	case OpcodeVselect:
		panic("TODO: format for OpcodeVselect")
	case OpcodeVanyTrue:
		panic("TODO: format for OpcodeVanyTrue")
	case OpcodeVallTrue:
		panic("TODO: format for OpcodeVallTrue")
	case OpcodeVhighBits:
		panic("TODO: format for OpcodeVhighBits")
	case OpcodeIcmp:
		panic("TODO: format for OpcodeIcmp")
	case OpcodeIcmpImm:
		panic("TODO: format for OpcodeIcmpImm")
	case OpcodeIfcmp:
		panic("TODO: format for OpcodeIfcmp")
	case OpcodeIfcmpImm:
		panic("TODO: format for OpcodeIfcmpImm")
	case OpcodeIadd:
		panic("TODO: format for OpcodeIadd")
	case OpcodeIsub:
		panic("TODO: format for OpcodeIsub")
	case OpcodeIneg:
		panic("TODO: format for OpcodeIneg")
	case OpcodeIabs:
		panic("TODO: format for OpcodeIabs")
	case OpcodeImul:
		panic("TODO: format for OpcodeImul")
	case OpcodeUmulhi:
		panic("TODO: format for OpcodeUmulhi")
	case OpcodeSmulhi:
		panic("TODO: format for OpcodeSmulhi")
	case OpcodeSqmulRoundSat:
		panic("TODO: format for OpcodeSqmulRoundSat")
	case OpcodeUdiv:
		panic("TODO: format for OpcodeUdiv")
	case OpcodeSdiv:
		panic("TODO: format for OpcodeSdiv")
	case OpcodeUrem:
		panic("TODO: format for OpcodeUrem")
	case OpcodeSrem:
		panic("TODO: format for OpcodeSrem")
	case OpcodeIaddImm:
		panic("TODO: format for OpcodeIaddImm")
	case OpcodeImulImm:
		panic("TODO: format for OpcodeImulImm")
	case OpcodeUdivImm:
		panic("TODO: format for OpcodeUdivImm")
	case OpcodeSdivImm:
		panic("TODO: format for OpcodeSdivImm")
	case OpcodeUremImm:
		panic("TODO: format for OpcodeUremImm")
	case OpcodeSremImm:
		panic("TODO: format for OpcodeSremImm")
	case OpcodeIrsubImm:
		panic("TODO: format for OpcodeIrsubImm")
	case OpcodeIaddCin:
		panic("TODO: format for OpcodeIaddCin")
	case OpcodeIaddIfcin:
		panic("TODO: format for OpcodeIaddIfcin")
	case OpcodeIaddCout:
		panic("TODO: format for OpcodeIaddCout")
	case OpcodeIaddIfcout:
		panic("TODO: format for OpcodeIaddIfcout")
	case OpcodeIaddCarry:
		panic("TODO: format for OpcodeIaddCarry")
	case OpcodeIaddIfcarry:
		panic("TODO: format for OpcodeIaddIfcarry")
	case OpcodeUaddOverflowTrap:
		panic("TODO: format for OpcodeUaddOverflowTrap")
	case OpcodeIsubBin:
		panic("TODO: format for OpcodeIsubBin")
	case OpcodeIsubIfbin:
		panic("TODO: format for OpcodeIsubIfbin")
	case OpcodeIsubBout:
		panic("TODO: format for OpcodeIsubBout")
	case OpcodeIsubIfbout:
		panic("TODO: format for OpcodeIsubIfbout")
	case OpcodeIsubBorrow:
		panic("TODO: format for OpcodeIsubBorrow")
	case OpcodeIsubIfborrow:
		panic("TODO: format for OpcodeIsubIfborrow")
	case OpcodeBand:
		panic("TODO: format for OpcodeBand")
	case OpcodeBor:
		panic("TODO: format for OpcodeBor")
	case OpcodeBxor:
		panic("TODO: format for OpcodeBxor")
	case OpcodeBnot:
		panic("TODO: format for OpcodeBnot")
	case OpcodeBandNot:
		panic("TODO: format for OpcodeBandNot")
	case OpcodeBorNot:
		panic("TODO: format for OpcodeBorNot")
	case OpcodeBxorNot:
		panic("TODO: format for OpcodeBxorNot")
	case OpcodeBandImm:
		panic("TODO: format for OpcodeBandImm")
	case OpcodeBorImm:
		panic("TODO: format for OpcodeBorImm")
	case OpcodeBxorImm:
		panic("TODO: format for OpcodeBxorImm")
	case OpcodeRotl:
		panic("TODO: format for OpcodeRotl")
	case OpcodeRotr:
		panic("TODO: format for OpcodeRotr")
	case OpcodeRotlImm:
		panic("TODO: format for OpcodeRotlImm")
	case OpcodeRotrImm:
		panic("TODO: format for OpcodeRotrImm")
	case OpcodeIshl:
		panic("TODO: format for OpcodeIshl")
	case OpcodeUshr:
		panic("TODO: format for OpcodeUshr")
	case OpcodeSshr:
		panic("TODO: format for OpcodeSshr")
	case OpcodeIshlImm:
		panic("TODO: format for OpcodeIshlImm")
	case OpcodeUshrImm:
		panic("TODO: format for OpcodeUshrImm")
	case OpcodeSshrImm:
		panic("TODO: format for OpcodeSshrImm")
	case OpcodeBitrev:
		panic("TODO: format for OpcodeBitrev")
	case OpcodeClz:
		panic("TODO: format for OpcodeClz")
	case OpcodeCls:
		panic("TODO: format for OpcodeCls")
	case OpcodeCtz:
		panic("TODO: format for OpcodeCtz")
	case OpcodeBswap:
		panic("TODO: format for OpcodeBswap")
	case OpcodePopcnt:
		panic("TODO: format for OpcodePopcnt")
	case OpcodeFcmp:
		panic("TODO: format for OpcodeFcmp")
	case OpcodeFfcmp:
		panic("TODO: format for OpcodeFfcmp")
	case OpcodeFadd:
		panic("TODO: format for OpcodeFadd")
	case OpcodeFsub:
		panic("TODO: format for OpcodeFsub")
	case OpcodeFmul:
		panic("TODO: format for OpcodeFmul")
	case OpcodeFdiv:
		panic("TODO: format for OpcodeFdiv")
	case OpcodeSqrt:
		panic("TODO: format for OpcodeSqrt")
	case OpcodeFma:
		panic("TODO: format for OpcodeFma")
	case OpcodeFneg:
		panic("TODO: format for OpcodeFneg")
	case OpcodeFabs:
		panic("TODO: format for OpcodeFabs")
	case OpcodeFcopysign:
		panic("TODO: format for OpcodeFcopysign")
	case OpcodeFmin:
		panic("TODO: format for OpcodeFmin")
	case OpcodeFminPseudo:
		panic("TODO: format for OpcodeFminPseudo")
	case OpcodeFmax:
		panic("TODO: format for OpcodeFmax")
	case OpcodeFmaxPseudo:
		panic("TODO: format for OpcodeFmaxPseudo")
	case OpcodeCeil:
		panic("TODO: format for OpcodeCeil")
	case OpcodeFloor:
		panic("TODO: format for OpcodeFloor")
	case OpcodeTrunc:
		panic("TODO: format for OpcodeTrunc")
	case OpcodeNearest:
		panic("TODO: format for OpcodeNearest")
	case OpcodeIsNull:
		panic("TODO: format for OpcodeIsNull")
	case OpcodeIsInvalid:
		panic("TODO: format for OpcodeIsInvalid")
	case OpcodeBitcast:
		panic("TODO: format for OpcodeBitcast")
	case OpcodeScalarToVector:
		panic("TODO: format for OpcodeScalarToVector")
	case OpcodeBmask:
		panic("TODO: format for OpcodeBmask")
	case OpcodeIreduce:
		panic("TODO: format for OpcodeIreduce")
	case OpcodeSnarrow:
		panic("TODO: format for OpcodeSnarrow")
	case OpcodeUnarrow:
		panic("TODO: format for OpcodeUnarrow")
	case OpcodeUunarrow:
		panic("TODO: format for OpcodeUunarrow")
	case OpcodeSwidenLow:
		panic("TODO: format for OpcodeSwidenLow")
	case OpcodeSwidenHigh:
		panic("TODO: format for OpcodeSwidenHigh")
	case OpcodeUwidenLow:
		panic("TODO: format for OpcodeUwidenLow")
	case OpcodeUwidenHigh:
		panic("TODO: format for OpcodeUwidenHigh")
	case OpcodeIaddPairwise:
		panic("TODO: format for OpcodeIaddPairwise")
	case OpcodeWideningPairwiseDotProductS:
		panic("TODO: format for OpcodeWideningPairwiseDotProductS")
	case OpcodeUextend:
		panic("TODO: format for OpcodeUextend")
	case OpcodeSextend:
		panic("TODO: format for OpcodeSextend")
	case OpcodeFpromote:
		panic("TODO: format for OpcodeFpromote")
	case OpcodeFdemote:
		panic("TODO: format for OpcodeFdemote")
	case OpcodeFvdemote:
		panic("TODO: format for OpcodeFvdemote")
	case OpcodeFvpromoteLow:
		panic("TODO: format for OpcodeFvpromoteLow")
	case OpcodeFcvtToUint:
		panic("TODO: format for OpcodeFcvtToUint")
	case OpcodeFcvtToSint:
		panic("TODO: format for OpcodeFcvtToSint")
	case OpcodeFcvtToUintSat:
		panic("TODO: format for OpcodeFcvtToUintSat")
	case OpcodeFcvtToSintSat:
		panic("TODO: format for OpcodeFcvtToSintSat")
	case OpcodeFcvtFromUint:
		panic("TODO: format for OpcodeFcvtFromUint")
	case OpcodeFcvtFromSint:
		panic("TODO: format for OpcodeFcvtFromSint")
	case OpcodeFcvtLowFromSint:
		panic("TODO: format for OpcodeFcvtLowFromSint")
	case OpcodeIsplit:
		panic("TODO: format for OpcodeIsplit")
	case OpcodeIconcat:
		panic("TODO: format for OpcodeIconcat")
	case OpcodeAtomicRmw:
		panic("TODO: format for OpcodeAtomicRmw")
	case OpcodeAtomicCas:
		panic("TODO: format for OpcodeAtomicCas")
	case OpcodeAtomicLoad:
		panic("TODO: format for OpcodeAtomicLoad")
	case OpcodeAtomicStore:
		panic("TODO: format for OpcodeAtomicStore")
	case OpcodeFence:
		panic("TODO: format for OpcodeFence")
	case OpcodeExtractVector:
		panic("TODO: format for OpcodeExtractVector")
	}
	return
}
