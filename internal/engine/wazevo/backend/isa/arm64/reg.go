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

	// Vector registers. Note that we do not distinguish vn and dn, ... registers
	// because they are the same from the perspective of register allocator, and
	// the size can be determined by the type of the instruction.

	v0
	v1
	v2
	v3
	v4
	v5
	v6
	v7
	v8
	v9
	v10
	v11
	v12
	v13
	v14
	v15
	v16
	v17
	v18
	v19
	v20
	v21
	v22
	v23
	v24
	v25
	v26
	v27
	v28
	v29
	v30

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
	v0:  "v0",
	v1:  "v1",
	v2:  "v2",
	v3:  "v3",
	v4:  "v4",
	v5:  "v5",
	v6:  "v6",
	v7:  "v7",
	v8:  "v8",
	v9:  "v9",
	v10: "v10",
	v11: "v11",
	v12: "v12",
	v13: "v13",
	v14: "v14",
	v15: "v15",
	v16: "v16",
	v17: "v17",
	v18: "v18",
	v19: "v19",
	v20: "v20",
	v21: "v21",
	v22: "v22",
	v23: "v23",
	v24: "v24",
	v25: "v25",
	v26: "v26",
	v27: "v27",
	v28: "v28",
	v29: "v29",
	v30: "v30",
}

func formatVReg(r backend.VReg) string {
	if r.RealReg() != backend.RealRegInvalid {
		return regNames[r.RealReg()]
	} else {
		return r.String()
	}
}

func formatVRegSized(r backend.VReg, size byte) (ret string) {
	if r.RealReg() != backend.RealRegInvalid {
		ret = regNames[r.RealReg()]
		switch ret[0] {
		case 'x':
			switch size {
			case 32:
				ret = strings.Replace(ret, "x", "w", 1)
			}
		case 'v':
			switch size {
			case 32:
				ret = strings.Replace(ret, "v", "w", 1)
			case 64:
				ret = strings.Replace(ret, "v", "d", 1)
			default:
				panic("TODO")
			}
		}
	} else {
		ret = r.String()
	}
	return
}
