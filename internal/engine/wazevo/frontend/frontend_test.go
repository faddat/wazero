package frontend

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
	"testing"
)

func TestNewFrontendCompiler(t *testing.T) {
	b := ssa.NewBuilder()
	fc := NewFrontendCompiler(&wasm.Module{}, b)
	require.NotNil(t, fc)
}
