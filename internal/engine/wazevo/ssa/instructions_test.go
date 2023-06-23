package ssa

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestInstruction_InvertConditionalBrx(t *testing.T) {
	i := &Instruction{opcode: OpcodeBrnz}
	i.InvertConditionalBrx()
	require.Equal(t, OpcodeBrz, i.opcode)
	i.InvertConditionalBrx()
	require.Equal(t, OpcodeBrnz, i.opcode)
}
