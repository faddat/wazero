package frontend

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
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

	blockSignature_vv = 0x40 // 0x40 is the v_v signature in 33-bit signed. See wasm.DecodeBlockType.
)

func TestCompiler_LowerToSSA(t *testing.T) {
	vv := wasm.FunctionType{}
	v_i32 := wasm.FunctionType{Results: []wasm.ValueType{i32}}
	i32_v := wasm.FunctionType{Params: []wasm.ValueType{i32}}
	i32_i32 := wasm.FunctionType{Params: []wasm.ValueType{i32}, Results: []wasm.ValueType{i32}}
	i32i32_i32 := wasm.FunctionType{Params: []wasm.ValueType{i32, i32}, Results: []wasm.ValueType{i32}}
	i32i32_i32i32 := wasm.FunctionType{Params: []wasm.ValueType{i32, i32}, Results: []wasm.ValueType{i32, i32}}
	i32_i32i32 := wasm.FunctionType{Params: []wasm.ValueType{i32}, Results: []wasm.ValueType{i32, i32}}
	i32f32f64_v := wasm.FunctionType{Params: []wasm.ValueType{i32, f32, f64}, Results: nil}

	for _, tc := range []struct {
		name string
		m    *wasm.Module
		exp  string
	}{
		{
			name: "empty", m: singleFunctionModule(vv, []byte{wasm.OpcodeEnd}, nil),
			exp: `
blk0: ()
	Jump blk_ret
`,
		},
		{
			name: "only return", m: singleFunctionModule(vv, []byte{wasm.OpcodeReturn, wasm.OpcodeEnd}, nil),
			exp: `
blk0: ()
	Return
`,
		},
		{
			name: "params", m: singleFunctionModule(i32f32f64_v, []byte{wasm.OpcodeReturn, wasm.OpcodeEnd}, nil),
			exp: `
blk0: (v1: i32, v2: f32, v3: f64)
	Return
`,
		},
		{
			name: "param -> return", m: singleFunctionModule(i32_i32, []byte{
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeEnd,
			}, nil),
			exp: `
blk0: (v1: i32)
	Jump blk_ret, v1
`,
		},
		{
			name: "locals", m: singleFunctionModule(vv, []byte{wasm.OpcodeEnd},
				[]wasm.ValueType{i32, i64, f32, f64}),
			exp: `
blk0: ()
	v1 = Iconst_32 0x0
	v2 = Iconst_64 0x0
	v3 = F32const 0.000000
	v4 = F64const 0.000000
	Jump blk_ret
`,
		},
		{
			name: "locals + params", m: singleFunctionModule(i32f32f64_v, []byte{wasm.OpcodeEnd},
				[]wasm.ValueType{i32, i64, f32, f64}),
			exp: `
blk0: (v1: i32, v2: f32, v3: f64)
	v4 = Iconst_32 0x0
	v5 = Iconst_64 0x0
	v6 = F32const 0.000000
	v7 = F64const 0.000000
	Jump blk_ret
`,
		},
		{
			name: "locals + params return", m: singleFunctionModule(i32_i32i32, []byte{
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeLocalGet, 1,
				wasm.OpcodeEnd,
			},
				[]wasm.ValueType{i32}),
			exp: `
blk0: (v1: i32)
	v2 = Iconst_32 0x0
	Jump blk_ret, v1, v2
`,
		},
		{
			name: "swap param and return", m: singleFunctionModule(i32i32_i32i32, []byte{
				wasm.OpcodeLocalGet, 1,
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeEnd,
			}, nil),
			exp: `
blk0: (v1: i32, v2: i32)
	Jump blk_ret, v2, v1
`,
		},
		{
			name: "swap params and return", m: singleFunctionModule(i32i32_i32i32, []byte{
				wasm.OpcodeLocalGet, 1,
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeLocalSet, 1,
				wasm.OpcodeLocalSet, 0,
				wasm.OpcodeBlock, blockSignature_vv,
				wasm.OpcodeEnd,
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeLocalGet, 1,
				wasm.OpcodeEnd,
			}, nil),
			exp: `
blk0: (v1: i32, v2: i32)
	Jump blk1

blk1: () <-- (blk0)
	Jump blk_ret, v2, v1
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
blk0: ()
	v1 = Iconst_32 0x0
	v2 = Iconst_64 0x0
	v3 = F32const 0.000000
	v4 = F64const 0.000000
	Jump blk1

blk1: () <-- (blk0)
	Jump blk_ret
`,
		},
		{
			name: "block - br_if", m: singleFunctionModule(vv, []byte{
				wasm.OpcodeBlock, 0,
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeBrIf, 0,
				wasm.OpcodeReturn,
				wasm.OpcodeEnd,
				wasm.OpcodeEnd,
			},
				[]wasm.ValueType{i32}),
			exp: `
blk0: ()
	v1 = Iconst_32 0x0
	Brz v1, blk1
	Jump blk2

blk1: () <-- (blk0)
	Jump blk_ret

blk2: () <-- (blk0)
	Return
`,
		},
		{
			name: "loop - br", m: singleFunctionModule(vv, []byte{
				wasm.OpcodeLoop, 0,
				wasm.OpcodeBr, 0,
				wasm.OpcodeEnd,
				wasm.OpcodeEnd,
			}, []wasm.ValueType{}),
			exp: `
blk0: ()
	Jump blk1

blk1: () <-- (blk0,blk1)
	Jump blk1

blk2: ()
	Jump blk_ret
`,
		},
		{
			name: "loop - br_if", m: singleFunctionModule(vv, []byte{
				wasm.OpcodeLoop, 0,
				wasm.OpcodeI32Const, 1,
				wasm.OpcodeBrIf, 0,
				wasm.OpcodeReturn,
				wasm.OpcodeEnd,
				wasm.OpcodeEnd,
			}, []wasm.ValueType{}),
			exp: `
blk0: ()
	Jump blk1

blk1: () <-- (blk0,blk1)
	v1 = Iconst_32 0x1
	Brz v1, blk1
	Jump blk3

blk2: ()
	Jump blk_ret

blk3: () <-- (blk1)
	Return
`,
		},
		{
			name: "block - block - br", m: singleFunctionModule(vv, []byte{
				wasm.OpcodeBlock, 0,
				wasm.OpcodeBlock, 0,
				wasm.OpcodeBr, 1,
				wasm.OpcodeEnd,
				wasm.OpcodeEnd,
				wasm.OpcodeEnd,
			},
				[]wasm.ValueType{i32, i64, f32, f64}),
			exp: `
blk0: ()
	v1 = Iconst_32 0x0
	v2 = Iconst_64 0x0
	v3 = F32const 0.000000
	v4 = F64const 0.000000
	Jump blk1

blk1: () <-- (blk0,blk2)
	Jump blk_ret

blk2: ()
	Jump blk1
`,
		},
		{
			name: "if without else", m: singleFunctionModule(vv, []byte{
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeIf, 0,
				wasm.OpcodeEnd,
				wasm.OpcodeEnd,
			},
				[]wasm.ValueType{i32}),
			exp: `
blk0: ()
	v1 = Iconst_32 0x0
	Brz v1, blk2
	Jump blk1

blk1: () <-- (blk0)
	Jump blk3

blk2: () <-- (blk0)
	Jump blk3

blk3: () <-- (blk1,blk2)
	Jump blk_ret
`,
		},
		{
			name: "if-else", m: singleFunctionModule(vv, []byte{
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeIf, 0,
				wasm.OpcodeElse,
				wasm.OpcodeBr, 1,
				wasm.OpcodeEnd,
				wasm.OpcodeEnd,
			},
				[]wasm.ValueType{i32}),
			exp: `
blk0: ()
	v1 = Iconst_32 0x0
	Brz v1, blk2
	Jump blk1

blk1: () <-- (blk0)
	Jump blk3

blk2: () <-- (blk0)
	Jump blk_ret

blk3: () <-- (blk1)
	Jump blk_ret
`,
		},
		{
			name: "single predecessor local refs", m: &wasm.Module{
				TypeSection:     []wasm.FunctionType{vv, v_i32},
				FunctionSection: []wasm.Index{1},
				CodeSection: []wasm.Code{{
					LocalTypes: []wasm.ValueType{i32, i32, i32},
					Body: []byte{
						wasm.OpcodeLocalGet, 0,
						wasm.OpcodeIf, 0,
						// This is defined in the first block which is the sole predecessor of If.
						wasm.OpcodeLocalGet, 2,
						wasm.OpcodeReturn,
						wasm.OpcodeElse,
						wasm.OpcodeEnd,
						// This is defined in the first block which is the sole predecessor of this block.
						// Note that If block will never reach here because it's returning early.
						wasm.OpcodeLocalGet, 0,
						wasm.OpcodeEnd,
					},
				}},
			},
			exp: `
blk0: ()
	v1 = Iconst_32 0x0
	v2 = Iconst_32 0x0
	v3 = Iconst_32 0x0
	Brz v1, blk2
	Jump blk1

blk1: () <-- (blk0)
	Return v3

blk2: () <-- (blk0)
	Jump blk3

blk3: () <-- (blk2)
	Jump blk_ret, v1
`,
		},
		{
			name: "multi predecessors local ref",
			m: singleFunctionModule(i32i32_i32, []byte{
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeIf, blockSignature_vv,
				// Set the first param to the local.
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeLocalSet, 2,
				wasm.OpcodeElse,
				// Set the second param to the local.
				wasm.OpcodeLocalGet, 1,
				wasm.OpcodeLocalSet, 2,
				wasm.OpcodeEnd,

				// Return the local as a result which has multiple definitions in predecessors (Then and Else).
				wasm.OpcodeLocalGet, 2,
				wasm.OpcodeEnd,
			}, []wasm.ValueType{i32}),
			exp: `
blk0: (v1: i32, v2: i32)
	v3 = Iconst_32 0x0
	Brz v1, blk2
	Jump blk1

blk1: () <-- (blk0)
	Jump blk3, v1

blk2: () <-- (blk0)
	Jump blk3, v2

blk3: (v4: i32) <-- (blk1,blk2)
	Jump blk_ret, v4
`,
		},
		{
			name: "reference value from unsealed block",
			m: singleFunctionModule(i32_i32, []byte{
				wasm.OpcodeLoop, blockSignature_vv,
				// Loop will not be sealed until we reach the end,
				// so this will result in referencing the unsealed definition search.
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeReturn,
				wasm.OpcodeEnd,
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeEnd,
			}, []wasm.ValueType{i32}),
			exp: `
blk0: (v1: i32)
	v2 = Iconst_32 0x0
	Jump blk1, v1

blk1: (v3: i32) <-- (blk0)
	Return v3

blk2: (v4: i32)
	Jump blk_ret, v4
`,
		},
		{
			name: "reference value from unsealed block - #2",
			m: singleFunctionModule(i32_i32, []byte{
				wasm.OpcodeLoop, blockSignature_vv,
				wasm.OpcodeBlock, blockSignature_vv,

				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeBrIf, 1,
				wasm.OpcodeEnd,

				wasm.OpcodeEnd,
				wasm.OpcodeI32Const, 0,
				wasm.OpcodeEnd,
			}, []wasm.ValueType{}),
			exp: `
blk0: (v1: i32)
	Jump blk1, v1

blk1: (v2: i32) <-- (blk0,blk1)
	Brz v2, blk1, v2
	Jump blk4

blk2: () <-- (blk3)
	v3 = Iconst_32 0x0
	Jump blk_ret, v3

blk3: () <-- (blk4)
	Jump blk2

blk4: () <-- (blk1)
	Jump blk3
`,
		},
		{
			name: "reference value from unsealed block - #2",
			m: singleFunctionModule(i32_v, []byte{
				wasm.OpcodeLoop, blockSignature_vv,
				wasm.OpcodeBlock, blockSignature_vv,

				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeBrIf, 2,
				wasm.OpcodeEnd,
				wasm.OpcodeI32Const, 1,
				wasm.OpcodeLocalSet, 0,
				wasm.OpcodeBr, 0,
				wasm.OpcodeEnd,
				wasm.OpcodeI32Const, 0,
				wasm.OpcodeEnd,
			}, []wasm.ValueType{}),
			exp: `
blk0: (v1: i32)
	Jump blk1, v1

blk1: (v2: i32) <-- (blk0,blk3)
	Brz v2, blk_ret
	Jump blk4

blk2: ()
	v4 = Iconst_32 0x0
	Jump blk_ret

blk3: () <-- (blk4)
	v3 = Iconst_32 0x1
	Jump blk1, v3

blk4: () <-- (blk1)
	Jump blk3
`,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			b := ssa.NewBuilder()
			fc := NewFrontendCompiler(tc.m, b)
			typeIndex := tc.m.FunctionSection[0]
			code := &tc.m.CodeSection[0]
			fc.Init(0, &tc.m.TypeSection[typeIndex], code.LocalTypes, code.Body)
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
