package arm64

import (
	"testing"

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
				u1:   registerAsRegZeroCond(w0).asUint64(),
				u2:   label(100).asBranchTarget().asUint64(),
			},
			exp: "cbz w0, L100",
		},
		{
			i: &instruction{
				kind: condBr,
				u1:   registerAsRegNonZeroCond(x29).asUint64(),
				u2:   label(50).asBranchTarget().asUint64(),
			},
			exp: "cbnz x29, L50",
		},
		{exp: "nop0", i: &instruction{kind: nop0}},
		{exp: "b L0", i: &instruction{kind: br, u1: label(0).asBranchTarget().asUint64()}},
	} {
		t.Run(tc.exp, func(t *testing.T) { require.Equal(t, tc.exp, tc.i.String()) })
	}
}
