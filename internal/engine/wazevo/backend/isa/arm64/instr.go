package arm64

import "fmt"

type (
	// instruction represents either a real instruction in arm64, or the meta instructions
	// that are convenient for code generation. For example, inline constants are also treated
	// as instructions.
	//
	// Basically, each instruction knows how to get encoded in binaries. Hence, the final output of compilation
	// can be considered equivalent to the sequence of such instructions.
	//
	// Each field is interpreted depending on the kind.
	//
	// TODO: optimize the layout later once the impl settles.
	instruction struct {
		kind       instructionKind
		prev, next *instruction
		u1, u2     uint64
	}

	// instructionKind represents the kind of instruction.
	// This controls how the instruction struct is interpreted.
	instructionKind int
)

func (i *instruction) asNop0() {
	i.kind = nop0
}

func (i *instruction) asCondBr(c cond, target branchTarget) {
	i.kind = condBr
	i.u1 = c.asUint64()
	i.u2 = target.asUint64()
}

func (i *instruction) asBr(target branchTarget) {
	i.kind = condBr
	i.u1 = target.asUint64()
}

// String implements fmt.Stringer.
func (i *instruction) String() (str string) {
	switch i.kind {
	case nop0:
		str = "nop0"
	case nop4:
		panic("TODO")
	case aluRRR:
		panic("TODO")
	case aluRRRR:
		panic("TODO")
	case aluRRImm12:
		panic("TODO")
	case aluRRImmLogic:
		panic("TODO")
	case aluRRImmShift:
		panic("TODO")
	case aluRRRShift:
		panic("TODO")
	case aluRRRExtend:
		panic("TODO")
	case bitRR:
		panic("TODO")
	case uLoad8:
		panic("TODO")
	case sLoad8:
		panic("TODO")
	case uLoad16:
		panic("TODO")
	case sLoad16:
		panic("TODO")
	case uLoad32:
		panic("TODO")
	case sLoad32:
		panic("TODO")
	case uLoad64:
		panic("TODO")
	case store8:
		panic("TODO")
	case store16:
		panic("TODO")
	case store32:
		panic("TODO")
	case store64:
		panic("TODO")
	case storeP64:
		panic("TODO")
	case loadP64:
		panic("TODO")
	case mov64:
		panic("TODO")
	case mov32:
		panic("TODO")
	case movZ:
		panic("TODO")
	case movN:
		panic("TODO")
	case movK:
		panic("TODO")
	case extend:
		panic("TODO")
	case cSel:
		panic("TODO")
	case cSet:
		panic("TODO")
	case cCmpImm:
		panic("TODO")
	case fpuMove64:
		panic("TODO")
	case fpuMove128:
		panic("TODO")
	case fpuMoveFromVec:
		panic("TODO")
	case fpuRR:
		panic("TODO")
	case fpuRRR:
		panic("TODO")
	case fpuRRI:
		panic("TODO")
	case fpuRRRR:
		panic("TODO")
	case fpuCmp32:
		panic("TODO")
	case fpuCmp64:
		panic("TODO")
	case fpuLoad32:
		panic("TODO")
	case fpuStore32:
		panic("TODO")
	case fpuLoad64:
		panic("TODO")
	case fpuStore64:
		panic("TODO")
	case fpuLoad128:
		panic("TODO")
	case fpuStore128:
		panic("TODO")
	case loadFpuConst32:
		panic("TODO")
	case loadFpuConst64:
		panic("TODO")
	case loadFpuConst128:
		panic("TODO")
	case fpuToInt:
		panic("TODO")
	case intToFpu:
		panic("TODO")
	case fpuCSel32:
		panic("TODO")
	case fpuCSel64:
		panic("TODO")
	case fpuRound:
		panic("TODO")
	case movToFpu:
		panic("TODO")
	case movToVec:
		panic("TODO")
	case movFromVec:
		panic("TODO")
	case movFromVecSigned:
		panic("TODO")
	case vecDup:
		panic("TODO")
	case vecDupFromFpu:
		panic("TODO")
	case vecExtend:
		panic("TODO")
	case vecMovElement:
		panic("TODO")
	case vecMiscNarrow:
		panic("TODO")
	case vecRRR:
		panic("TODO")
	case vecMisc:
		panic("TODO")
	case vecLanes:
		panic("TODO")
	case vecTbl:
		panic("TODO")
	case vecTbl2:
		panic("TODO")
	case movToNZCV:
		panic("TODO")
	case movFromNZCV:
		panic("TODO")
	case call:
		panic("TODO")
	case callInd:
		panic("TODO")
	case ret:
		panic("TODO")
	case epiloguePlaceholder:
		panic("TODO")
	case br:
		target := branchTarget(i.u1)
		str = fmt.Sprintf("b %s", target.String())
	case condBr:
		c := cond(i.u1)
		target := branchTarget(i.u2)
		switch c.kind() {
		case condKindRegisterZero:
			str = fmt.Sprintf("cbz %s, %s", regNames[c.register()], target.String())
		case condKindRegisterNotZero:
			str = fmt.Sprintf("cbnz %s, %s", regNames[c.register()], target.String())
		case condKindCondFlagSet:
			str = fmt.Sprintf("b.%s %s", c.flag(), target.String())
		}
	case trapIf:
		panic("TODO")
	case indirectBr:
		panic("TODO")
	case adr:
		panic("TODO")
	case word4:
		panic("TODO")
	case word8:
		panic("TODO")
	case jtSequence:
		panic("TODO")
	case loadAddr:
		panic("TODO")
	default:
		panic(i.kind)
	}
	return
}

// TODO: delete unnecessary things. Currently they are derived from
// https://github.com/bytecodealliance/wasmtime/blob/cb306fd514f34e7dd818bb17658b93fba98e2567/cranelift/codegen/src/isa/aarch64/inst/mod.rs
const (
	// nop0 represents a no-op of zero size.
	nop0 instructionKind = iota
	// nop4 represents a no-op that is one instruction large.
	nop4
	// aluRRR represents an ALU operation with two register sources and a register destination.
	aluRRR
	// aluRRRR represents an ALU operation with three register sources and a register destination.
	aluRRRR
	// aluRRImm12 represents an ALU operation with a register source and an immediate-12 source, with a register destination.
	aluRRImm12
	// aluRRImmLogic represents an ALU operation with a register source and an immediate-logic source, with a register destination.
	aluRRImmLogic
	// aluRRImmShift represents an ALU operation with a register source and an immediate-shiftamt source, with a register destination.
	aluRRImmShift
	// aluRRRShift represents an ALU operation with two register sources, one of which can be shifted, with a register destination.
	aluRRRShift
	// aluRRRExtend represents an ALU operation with two register sources, one of which can be extended and shifted, with a register destination.
	aluRRRExtend
	// bitRR represents a bit op instruction with a single register source.
	bitRR
	// uLoad8 represents an unsigned 8-bit load.
	uLoad8
	// sLoad8 represents a signed 8-bit load.
	sLoad8
	// uLoad16 represents an unsigned 16-bit load.
	uLoad16
	// sLoad16 represents a signed 16-bit load.
	sLoad16
	// uLoad32 represents an unsigned 32-bit load.
	uLoad32
	// sLoad32 represents a signed 32-bit load.
	sLoad32
	// uLoad64 represents a 64-bit load.
	uLoad64
	// store8 represents an 8-bit store.
	store8
	// store16 represents a 16-bit store.
	store16
	// store32 represents a 32-bit store.
	store32
	// store64 represents a 64-bit store.
	store64
	// storeP64 represents a store of a pair of registers.
	storeP64
	// loadP64 represents a load of a pair of registers.
	loadP64
	// mov64 represents a MOV instruction. These are encoded as ORR's but we keep them separate for better handling.
	mov64
	// mov32 represents a 32-bit MOV. This zeroes the top 32 bits of the destination.
	mov32
	// movZ represents a MOVZ with a 16-bit immediate.
	movZ
	// movN represents a MOVN with a 16-bit immediate.
	movN
	// movK represents a MOVK with a 16-bit immediate.
	movK
	// extend represents a sign- or zero-extend operation.
	extend
	// cSel represents a conditional-select operation.
	cSel
	// cSet represents a conditional-set operation.
	cSet
	// cCmpImm represents a conditional comparison with an immediate.
	cCmpImm
	// fpuMove64 represents a FPU move. Distinct from a vector-register move; moving just 64 bits appears to be significantly faster.
	fpuMove64
	// fpuMove128 represents a vector register move.
	fpuMove128
	// fpuMoveFromVec represents a move to scalar from a vector element.
	fpuMoveFromVec
	// fpuRR represents a 1-op FPU instruction.
	fpuRR
	// fpuRRR represents a 2-op FPU instruction.
	fpuRRR
	// fpuRRI represents a 2-op FPU instruction with immediate value.
	fpuRRI
	// fpuRRRR represents a 3-op FPU instruction.
	fpuRRRR
	// fpuCmp32 represents a FPU comparison, single-precision (32 bit).
	fpuCmp32
	// fpuCmp64 represents a FPU comparison, double-precision (64 bit).
	fpuCmp64
	// fpuLoad32 represents a floating-point load, single-precision (32 bit).
	fpuLoad32
	// fpuStore32 represents a floating-point store, single-precision (32 bit).
	fpuStore32
	// fpuLoad64 represents a floating-point load, double-precision (64 bit).
	fpuLoad64
	// fpuStore64 represents a floating-point store, double-precision (64 bit).
	fpuStore64
	// fpuLoad128 represents a floating-point/vector load, 128 bit.
	fpuLoad128
	// fpuStore128 represents a floating-point/vector store, 128 bit.
	fpuStore128
	// loadFpuConst32 represents a load of a 32-bit floating-point constant.
	loadFpuConst32
	// loadFpuConst64 represents a load of a 64-bit floating-point constant.
	loadFpuConst64
	// loadFpuConst128 represents a load of a 128-bit floating-point constant.
	loadFpuConst128
	// fpuToInt represents a conversion from FP to integer.
	fpuToInt
	// intToFpu represents a conversion from integer to FP.
	intToFpu
	// fpuCSel32 represents a 32-bit FP conditional select.
	fpuCSel32
	// fpuCSel64 represents a 64-bit FP conditional select.
	fpuCSel64
	// fpuRound represents a rounding to integer operation.
	fpuRound
	// movToFpu represents a move from a GPR to a scalar FP register.
	movToFpu
	// movToVec represents a move to a vector element from a GPR.
	movToVec
	// movFromVec represents an unsigned move from a vector element to a GPR.
	movFromVec
	// movFromVecSigned represents a signed move from a vector element to a GPR.
	movFromVecSigned
	// vecDup represents a duplication of general-purpose register to vector.
	vecDup
	// vecDupFromFpu represents a duplication of scalar to vector.
	vecDupFromFpu
	// vecExtend represents a vector extension operation.
	vecExtend
	// vecMovElement represents a move vector element to another vector element operation.
	vecMovElement
	// vecMiscNarrow represents a vector narrowing operation.
	vecMiscNarrow
	// vecRRR represents a vector ALU operation.
	vecRRR
	// vecMisc represents a vector two register miscellaneous instruction.
	vecMisc
	// vecLanes represents a vector instruction across lanes.
	vecLanes
	// vecTbl represents a table vector lookup - single register table.
	vecTbl
	// vecTbl2 represents a table vector lookup - two register table.
	vecTbl2
	// movToNZCV represents a move to the NZCV flags.
	movToNZCV
	// movFromNZCV represents a move from the NZCV flags.
	movFromNZCV
	// call represents a machine call instruction.
	call
	// callInd represents a machine indirect-call instruction.
	callInd
	// ret represents a machine return instruction.
	ret
	// epiloguePlaceholder is a placeholder instruction, generating no code, meaning that a function epilogue must be
	// inserted there.
	epiloguePlaceholder
	// br represents an unconditional branch.
	br
	// condBr represents a conditional branch.
	condBr
	// trapIf represents a conditional trap.
	trapIf
	// indirectBr represents an indirect branch through a register.
	indirectBr
	// adr represents a compute the address (using a PC-relative offset) of a memory location.
	adr
	// word4 represents a raw 32-bit word.
	word4
	// word8 represents a raw 64-bit word.
	word8
	// jtSequence represents a jump-table sequence.
	jtSequence
	// loadAddr represents a load address instruction.
	loadAddr
)
