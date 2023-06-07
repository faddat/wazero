package frontend

import (
	"fmt"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
	"strings"
	"testing"
)

func TestNewFrontendCompiler(t *testing.T) {
	b := ssa.NewBuilder()
	fc := NewFrontendCompiler(&wasm.Module{}, b)
	require.NotNil(t, fc)
}

const (
	i32 = wasm.ValueTypeI32
	i64 = wasm.ValueTypeI64
	f32 = wasm.ValueTypeF32
	f64 = wasm.ValueTypeF64
)

func TestCompiler_LowerToSSA(t *testing.T) {
	vv := wasm.FunctionType{}
	i32f32f64_v := wasm.FunctionType{Params: []wasm.ValueType{i32, f32, f64}, Results: nil}

	for _, tc := range []struct {
		name string
		m    *wasm.Module
		exp  string
	}{
		{
			name: "empty", m: singleFunctionModule(vv, []byte{wasm.OpcodeEnd}, nil),
			exp: `
entrypoint: ()
	Return ()
`,
		},
		{
			name: "only return", m: singleFunctionModule(vv, []byte{wasm.OpcodeReturn, wasm.OpcodeEnd}, nil),
			exp: `
entrypoint: ()
	Return ()
`,
		},
		{
			name: "params", m: singleFunctionModule(i32f32f64_v, []byte{wasm.OpcodeReturn, wasm.OpcodeEnd}, nil),
			exp: `
entrypoint: (v1: i32, v2: f32, v3: f64)
	Return ()
`,
		},
		{
			name: "locals", m: singleFunctionModule(vv, []byte{wasm.OpcodeEnd},
				[]wasm.ValueType{i32, i64, f32, f64}),
			exp: `
entrypoint: ()
	v1 = Iconst_32 0x0
	v2 = Iconst_64 0x0
	v3 = F32const 0.000000
	v4 = F64const 0.000000
	Return ()
`,
		},
		{
			name: "locals + params", m: singleFunctionModule(i32f32f64_v, []byte{wasm.OpcodeEnd},
				[]wasm.ValueType{i32, i64, f32, f64}),
			exp: `
entrypoint: (v1: i32, v2: f32, v3: f64)
	v4 = Iconst_32 0x0
	v5 = Iconst_64 0x0
	v6 = F32const 0.000000
	v7 = F64const 0.000000
	Return ()
`,
		},
		{
			name: "block - br", m: singleFunctionModule(vv, []byte{
				wasm.OpcodeBlock, 0,
				wasm.OpcodeBr, 0,
				wasm.OpcodeEnd,
				wasm.OpcodeEnd,
			},
				[]wasm.ValueType{i32, i64, f32, f64}),
			exp: `
entrypoint: ()
	v1 = Iconst_32 0x0
	v2 = Iconst_64 0x0
	v3 = F32const 0.000000
	v4 = F64const 0.000000
	Jump blk1, ()

blk1: () <-- (blk0)
	Return ()
`,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			b := ssa.NewBuilder()
			fc := NewFrontendCompiler(tc.m, b)
			code := &tc.m.CodeSection[0]
			fc.Init(0, &tc.m.TypeSection[0], code.LocalTypes, code.Body)
			err := fc.LowerToSSA()
			require.NoError(t, err)
			exp := strings.TrimPrefix(tc.exp, "\n\n")
			exp = strings.TrimSuffix(exp, "\n\n")
			actual := fc.formatBuilder()
			fmt.Println(actual)
			require.Equal(t, exp, actual)
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
