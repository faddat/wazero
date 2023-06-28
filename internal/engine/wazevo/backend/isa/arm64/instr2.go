package arm64

// aluOp determines the type of ALU operation. Instructions whose kind is one of
// aluRRR, aluRRRR, aluRRImm12, aluRRImmLogic, aluRRImmShift, aluRRRShift and aluRRRExtend
// would use this type.
type aluOp int

const (
	// 32-bit Add.
	add32 aluOp = iota
	// 64-bit Add.
	add64
	// 32-bit Subtract.
	sub32
	// 64-bit Subtract.
	sub64
	// 32-bit Bitwise OR.
	orr32
	// 64-bit Bitwise OR.
	orr64
	// 32-bit Bitwise OR NOT.
	orrNot32
	// 64-bit Bitwise OR NOT.
	orrNot64
	// 32-bit Bitwise AND.
	and32
	// 64-bit Bitwise AND.
	and64
	// 32-bit Bitwise AND NOT.
	andNot32
	// 64-bit Bitwise AND NOT.
	andNot64
	// 32-bit Bitwise XOR (Exclusive OR).
	eor32
	// 64-bit Bitwise XOR (Exclusive OR).
	eor64
	// 32-bit Bitwise XNOR (Exclusive OR NOT).
	eorNot32
	// 64-bit Bitwise XNOR (Exclusive OR NOT).
	eorNot64
	// 32-bit Add setting flags.
	addS32
	// 64-bit Add setting flags.
	addS64
	// 32-bit Subtract setting flags.
	subS32
	// 64-bit Subtract setting flags.
	subS64
	// Signed multiply, high-word result.
	sMulH
	// Unsigned multiply, high-word result.
	uMulH
	// 64-bit Signed divide.
	sDiv64
	// 64-bit Unsigned divide.
	uDiv64
	// 32-bit Rotate right.
	rotR32
	// 64-bit Rotate right.
	rotR64
	// 32-bit Logical shift right.
	lsr32
	// 64-bit Logical shift right.
	lsr64
	// 32-bit Arithmetic shift right.
	asr32
	// 64-bit Arithmetic shift right.
	asr64
	// 32-bit Logical shift left.
	lsl32
	// 64-bit Logical shift left.
	lsl64
)

// extensionMode represents the mode of a register operand extension.
// For example, aluRRRExtend instructions need this info to determine the extensions.
type extensionMode byte

const (
	// extensionModeZeroExtend32 indicates a zero-extension to 32 bits if the original bit size is less than 32.
	extensionModeZeroExtend32 extensionMode = iota
	// extensionModeSignExtend32 represents a sign-extension to 32 bits if the original bit size is less than 32.
	extensionModeSignExtend32
	// extensionModeZeroExtend64 suggests a zero-extension to 64 bits if the original bit size is less than 64.
	extensionModeZeroExtend64
	// extensionModeSignExtend64 stands for a sign-extension to 64 bits if the original bit size is less than 64.
	extensionModeSignExtend64
)
