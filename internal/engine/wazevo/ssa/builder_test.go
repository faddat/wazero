package ssa

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
)

func TestNewBuilder(t *testing.T) {
	b := NewBuilder(&wasm.Module{})
	require.NotNil(t, b)
}
