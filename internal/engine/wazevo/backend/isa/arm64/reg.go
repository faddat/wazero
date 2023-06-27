package arm64

import "github.com/tetratelabs/wazero/internal/engine/wazevo/backend"

// Arm64-specific registers.
//
// See https://developer.arm.com/documentation/dui0801/a/Overview-of-AArch64-state/Predeclared-core-register-names-in-AArch64-state
const (
	// General purpose registers.

	w0 backend.RealReg = iota
	w1
	w2
	w3
	w4
	w5
	w6
	w7
	w8
	w9
	w10
	w11
	w12
	w13
	w14
	w15
	w16
	w17
	w18
	w19
	w20
	w21
	w22
	w23
	w24
	w25
	w26
	w27
	w28
	w29
	w30

	// Vectors registers.

	x0
	x1
	x2
	x3
	x4
	x5
	x6
	x7
	x8
	x9
	x10
	x11
	x12
	x13
	x14
	x15
	x16
	x17
	x18
	x19
	x20
	x21
	x22
	x23
	x24
	x25
	x26
	x27
	x28
	x29
	x30

	// Special registers

	wzr
	xzr
	wsp
	sp
	lr

	numRegisters
)

var regNames = [...]string{
	w0:  "w0",
	w1:  "w1",
	w2:  "w2",
	w3:  "w3",
	w4:  "w4",
	w5:  "w5",
	w6:  "w6",
	w7:  "w7",
	w8:  "w8",
	w9:  "w9",
	w10: "w10",
	w11: "w11",
	w12: "w12",
	w13: "w13",
	w14: "w14",
	w15: "w15",
	w16: "w16",
	w17: "w17",
	w18: "w18",
	w19: "w19",
	w20: "w20",
	w21: "w21",
	w22: "w22",
	w23: "w23",
	w24: "w24",
	w25: "w25",
	w26: "w26",
	w27: "w27",
	w28: "w28",
	w29: "w29",
	w30: "w30",
	x0:  "x0",
	x1:  "x1",
	x2:  "x2",
	x3:  "x3",
	x4:  "x4",
	x5:  "x5",
	x6:  "x6",
	x7:  "x7",
	x8:  "x8",
	x9:  "x9",
	x10: "x10",
	x11: "x11",
	x12: "x12",
	x13: "x13",
	x14: "x14",
	x15: "x15",
	x16: "x16",
	x17: "x17",
	x18: "x18",
	x19: "x19",
	x20: "x20",
	x21: "x21",
	x22: "x22",
	x23: "x23",
	x24: "x24",
	x25: "x25",
	x26: "x26",
	x27: "x27",
	x28: "x28",
	x29: "x29",
	x30: "x30",
	wzr: "wzr",
	xzr: "xzr",
	wsp: "wsp",
	sp:  "sp",
	lr:  "lr",
}
