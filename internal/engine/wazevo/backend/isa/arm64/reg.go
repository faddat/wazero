package arm64

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
)

// Arm64-specific registers.
//
// See https://developer.arm.com/documentation/dui0801/a/Overview-of-AArch64-state/Predeclared-core-register-names-in-AArch64-state

const (
	// General purpose registers.

	w0 = backend.RealRegInvalid + 1 + iota
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

var (
	w0Vreg  = backend.VRegFromRealRegister(w0)
	w1Vreg  = backend.VRegFromRealRegister(w1)
	w2Vreg  = backend.VRegFromRealRegister(w2)
	w3Vreg  = backend.VRegFromRealRegister(w3)
	w4Vreg  = backend.VRegFromRealRegister(w4)
	w5Vreg  = backend.VRegFromRealRegister(w5)
	w6Vreg  = backend.VRegFromRealRegister(w6)
	w7Vreg  = backend.VRegFromRealRegister(w7)
	w8Vreg  = backend.VRegFromRealRegister(w8)
	w9Vreg  = backend.VRegFromRealRegister(w9)
	w10Vreg = backend.VRegFromRealRegister(w10)
	w11Vreg = backend.VRegFromRealRegister(w11)
	w12Vreg = backend.VRegFromRealRegister(w12)
	w13Vreg = backend.VRegFromRealRegister(w13)
	w14Vreg = backend.VRegFromRealRegister(w14)
	w15Vreg = backend.VRegFromRealRegister(w15)
	w16Vreg = backend.VRegFromRealRegister(w16)
	w17Vreg = backend.VRegFromRealRegister(w17)
	w18Vreg = backend.VRegFromRealRegister(w18)
	w19Vreg = backend.VRegFromRealRegister(w19)
	w20Vreg = backend.VRegFromRealRegister(w20)
	w21Vreg = backend.VRegFromRealRegister(w21)
	w22Vreg = backend.VRegFromRealRegister(w22)
	w23Vreg = backend.VRegFromRealRegister(w23)
	w24Vreg = backend.VRegFromRealRegister(w24)
	w25Vreg = backend.VRegFromRealRegister(w25)
	w26Vreg = backend.VRegFromRealRegister(w26)
	w27Vreg = backend.VRegFromRealRegister(w27)
	w28Vreg = backend.VRegFromRealRegister(w28)
	w29Vreg = backend.VRegFromRealRegister(w29)
	w30Vreg = backend.VRegFromRealRegister(w30)
	wzrVReg = backend.VRegFromRealRegister(wzr)
	xzrVReg = backend.VRegFromRealRegister(xzr)
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

func prettyVReg(r backend.VReg) string {
	if r.RealReg() != backend.RealRegInvalid {
		return regNames[r.RealReg()]
	} else {
		return r.String()
	}
}
