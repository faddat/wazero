package arm64

import (
	"math"
	"testing"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestInstruction_String(t *testing.T) {
	for _, tc := range []struct {
		i   *instruction
		exp string
	}{
		{
			i: &instruction{
				kind: condBr,
				u1:   eq.asCond().asUint64(),
				u2:   label(1).asBranchTarget().asUint64(),
			},
			exp: "b.eq L1",
		},
		{
			i: &instruction{
				kind: condBr,
				u1:   ne.asCond().asUint64(),
				u2:   label(100).asBranchTarget().asUint64(),
			},
			exp: "b.ne L100",
		},
		{
			i: &instruction{
				kind: condBr,
				u1:   registerAsRegZeroCond(w0Vreg).asUint64(),
				u2:   label(100).asBranchTarget().asUint64(),
			},
			exp: "cbz w0, L100",
		},
		{
			i: &instruction{
				kind: condBr,
				u1:   registerAsRegNonZeroCond(w28Vreg).asUint64(),
				u2:   label(50).asBranchTarget().asUint64(),
			},
			exp: "cbnz w28, L50",
		},
		{
			i: &instruction{
				kind: loadFpuConst32,
				u1:   uint64(math.Float32bits(3.0)),
				rd:   operandNR(backend.VReg(backend.VRegIDUnreservedBegin)),
			},
			exp: "ldr v?0, pc+8; b 8; data.f32 3.000000",
		},
		{
			i: &instruction{
				kind: loadFpuConst64,
				u1:   math.Float64bits(12345.987491),
				rd:   operandNR(backend.VReg(backend.VRegIDUnreservedBegin)),
			},
			exp: "ldr v?0, pc+8; b 16; data.f64 12345.987491",
		},
		{exp: "nop0", i: &instruction{kind: nop0}},
		{exp: "b L0", i: &instruction{kind: br, u1: label(0).asBranchTarget().asUint64()}},
	} {
		t.Run(tc.exp, func(t *testing.T) { require.Equal(t, tc.exp, tc.i.String()) })
	}
}
