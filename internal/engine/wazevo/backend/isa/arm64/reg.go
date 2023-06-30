package arm64

import (
	"strings"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
)

// Arm64-specific registers.
//
// See https://developer.arm.com/documentation/dui0801/a/Overview-of-AArch64-state/Predeclared-core-register-names-in-AArch64-state

const (
	// General purpose registers. Note that we do not distinguish wn and xn registers
	// because they are the same from the perspective of register allocator, and
	// the size can be determined by the type of the instruction.

	x0 = backend.RealRegInvalid + 1 + iota
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

var xzrVReg = backend.VReg(backend.VRegIDReserved).SetRealReg(xzr)

var regNames = [...]string{
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

func formatVReg(r backend.VReg) string {
	if r.RealReg() != backend.RealRegInvalid {
		return regNames[r.RealReg()]
	} else {
		return r.String()
	}
}

func formatVRegSized(r backend.VReg, _32bit bool) (ret string) {
	if r.RealReg() != backend.RealRegInvalid {
		ret = regNames[r.RealReg()]
		if _32bit {
			ret = strings.Replace(ret, "x", "w", 1)
		}
	} else {
		ret = r.String()
	}
	return
}
