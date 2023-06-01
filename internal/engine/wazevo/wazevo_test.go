package wazevo

import (
	"context"
	"testing"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
)

var ctx = context.Background()

func TestNewEngine(t *testing.T) {
	e := NewEngine(ctx, api.CoreFeaturesV1, nil)
	require.NotNil(t, e)
}

func TestEngine_CompiledModuleCount(t *testing.T) {
	e, ok := NewEngine(ctx, api.CoreFeaturesV1, nil).(*engine)
	require.True(t, ok)
	require.Equal(t, uint32(0), e.CompiledModuleCount())
	e.compiledModules[wasm.ModuleID{}] = &compiledModule{}
	require.Equal(t, uint32(1), e.CompiledModuleCount())
}

func TestEngine_DeleteCompiledModule(t *testing.T) {
	e, ok := NewEngine(ctx, api.CoreFeaturesV1, nil).(*engine)
	require.True(t, ok)
	id := wasm.ModuleID{0xaa}
	e.compiledModules[id] = &compiledModule{}
	require.Equal(t, uint32(1), e.CompiledModuleCount())
	e.DeleteCompiledModule(&wasm.Module{ID: id})
	require.Equal(t, uint32(0), e.CompiledModuleCount())
}

// TestEngine_CompileModule tracks what can be compiled with the current implementation.
// This will be eventually removed after wazevo reaches some maturity (e.g. passing v1 spectest).
func TestEngine_CompileModule(t *testing.T) {
	vv := wasm.FunctionType{}

	for _, tc := range []struct {
		name string
		m    *wasm.Module
	}{
		{name: "empty", m: &wasm.Module{}},
		{name: "empty return", m: singleFunctionModule(vv, []byte{wasm.OpcodeReturn}, nil)},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			e := NewEngine(ctx, api.CoreFeaturesV1, nil)
			err := e.CompileModule(ctx, tc.m, nil, false)
			require.NoError(t, err)
		})
	}
}

func singleFunctionModule(typ wasm.FunctionType, body []byte, localTypes []wasm.ValueType) *wasm.Module {
	return &wasm.Module{
		TypeSection:     []wasm.FunctionType{typ},
		FunctionSection: []wasm.Index{0},
		CodeSection: []wasm.Code{{
			LocalTypes: localTypes,
			Body:       body,
		}},
	}
}
