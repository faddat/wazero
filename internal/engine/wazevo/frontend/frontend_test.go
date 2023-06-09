package frontend

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/internal/wasm"
)

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
blk0: (exec_ctx: i64, module_ctx: i64)
	Jump blk_ret
`,
		},
		{
			name: "unreachable", m: singleFunctionModule(vv, []byte{wasm.OpcodeUnreachable, wasm.OpcodeEnd}, nil),
			exp: `
blk0: (exec_ctx: i64, module_ctx: i64)
	Jump blk1

blk1: () <-- (blk0)
	v3 = Iconst_32 0x0
	Store v3, exec_ctx, 0x0
	Trap
`,
		},
		{
			name: "only return", m: singleFunctionModule(vv, []byte{wasm.OpcodeReturn, wasm.OpcodeEnd}, nil),
			exp: `
blk0: (exec_ctx: i64, module_ctx: i64)
	Return
`,
		},
		{
			name: "params", m: singleFunctionModule(i32f32f64_v, []byte{wasm.OpcodeReturn, wasm.OpcodeEnd}, nil),
			exp: `
blk0: (exec_ctx: i64, module_ctx: i64, v3: i32, v4: f32, v5: f64)
	Return
`,
		},
		{
			name: "param -> return", m: singleFunctionModule(i32_i32, []byte{
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeEnd,
			}, nil),
			exp: `
blk0: (exec_ctx: i64, module_ctx: i64, v3: i32)
	Jump blk_ret, v3
`,
		},
		{
			name: "locals", m: singleFunctionModule(vv, []byte{wasm.OpcodeEnd},
				[]wasm.ValueType{i32, i64, f32, f64}),
			exp: `
blk0: (exec_ctx: i64, module_ctx: i64)
	v3 = Iconst_32 0x0
	v4 = Iconst_64 0x0
	v5 = F32const 0.000000
	v6 = F64const 0.000000
	Jump blk_ret
`,
		},
		{
			name: "locals + params", m: singleFunctionModule(i32f32f64_v, []byte{wasm.OpcodeEnd},
				[]wasm.ValueType{i32, i64, f32, f64}),
			exp: `
blk0: (exec_ctx: i64, module_ctx: i64, v3: i32, v4: f32, v5: f64)
	v6 = Iconst_32 0x0
	v7 = Iconst_64 0x0
	v8 = F32const 0.000000
	v9 = F64const 0.000000
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
blk0: (exec_ctx: i64, module_ctx: i64, v3: i32)
	v4 = Iconst_32 0x0
	Jump blk_ret, v3, v4
`,
		},
		{
			name: "swap param and return", m: singleFunctionModule(i32i32_i32i32, []byte{
				wasm.OpcodeLocalGet, 1,
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeEnd,
			}, nil),
			exp: `
blk0: (exec_ctx: i64, module_ctx: i64, v3: i32, v4: i32)
	Jump blk_ret, v4, v3
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
blk0: (exec_ctx: i64, module_ctx: i64, v3: i32, v4: i32)
	Jump blk1

blk1: () <-- (blk0)
	Jump blk_ret, v4, v3
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
blk0: (exec_ctx: i64, module_ctx: i64)
	v3 = Iconst_32 0x0
	v4 = Iconst_64 0x0
	v5 = F32const 0.000000
	v6 = F64const 0.000000
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
				wasm.OpcodeUnreachable,
				wasm.OpcodeEnd,
				wasm.OpcodeEnd,
			},
				[]wasm.ValueType{i32}),
			exp: `
blk0: (exec_ctx: i64, module_ctx: i64)
	v3 = Iconst_32 0x0
	Brz v3, blk1
	Jump blk2

blk1: () <-- (blk0)
	Jump blk_ret

blk2: () <-- (blk0)
	Jump blk3

blk3: () <-- (blk2)
	v4 = Iconst_32 0x0
	Store v4, exec_ctx, 0x0
	Trap
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
blk0: (exec_ctx: i64, module_ctx: i64)
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
blk0: (exec_ctx: i64, module_ctx: i64)
	Jump blk1

blk1: () <-- (blk0,blk1)
	v3 = Iconst_32 0x1
	Brz v3, blk1
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
blk0: (exec_ctx: i64, module_ctx: i64)
	v3 = Iconst_32 0x0
	v4 = Iconst_64 0x0
	v5 = F32const 0.000000
	v6 = F64const 0.000000
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
blk0: (exec_ctx: i64, module_ctx: i64)
	v3 = Iconst_32 0x0
	Brz v3, blk2
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
blk0: (exec_ctx: i64, module_ctx: i64)
	v3 = Iconst_32 0x0
	Brz v3, blk2
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
blk0: (exec_ctx: i64, module_ctx: i64)
	v3 = Iconst_32 0x0
	v4 = Iconst_32 0x0
	v5 = Iconst_32 0x0
	Brz v3, blk2
	Jump blk1

blk1: () <-- (blk0)
	Return v5

blk2: () <-- (blk0)
	Jump blk3

blk3: () <-- (blk2)
	Jump blk_ret, v3
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
blk0: (exec_ctx: i64, module_ctx: i64, v3: i32, v4: i32)
	v5 = Iconst_32 0x0
	Brz v3, blk2
	Jump blk1

blk1: () <-- (blk0)
	Jump blk3, v3

blk2: () <-- (blk0)
	Jump blk3, v4

blk3: (v6: i32) <-- (blk1,blk2)
	Jump blk_ret, v6
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
blk0: (exec_ctx: i64, module_ctx: i64, v3: i32)
	v4 = Iconst_32 0x0
	Jump blk1, v3

blk1: (v5: i32) <-- (blk0)
	Return v5

blk2: (v6: i32)
	Jump blk_ret, v6
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
blk0: (exec_ctx: i64, module_ctx: i64, v3: i32)
	Jump blk1, v3

blk1: (v4: i32) <-- (blk0,blk1)
	Brz v4, blk1, v4
	Jump blk4

blk2: () <-- (blk3)
	v5 = Iconst_32 0x0
	Jump blk_ret, v5

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
				wasm.OpcodeEnd,
			}, []wasm.ValueType{}),
			exp: `
blk0: (exec_ctx: i64, module_ctx: i64, v3: i32)
	Jump blk1, v3

blk1: (v4: i32) <-- (blk0,blk3)
	Brz v4, blk_ret
	Jump blk4

blk2: ()
	Jump blk_ret

blk3: () <-- (blk4)
	v5 = Iconst_32 0x1
	Jump blk1, v5

blk4: () <-- (blk1)
	Jump blk3
`,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Just in case let's check the test module is valid.
			err := tc.m.Validate(api.CoreFeaturesV2)
			require.NoError(t, err, "invalid test case module!")

			b := ssa.NewBuilder()
			od := wazevoapi.NewOffsetData(tc.m)

			fc := NewFrontendCompiler(od, tc.m, b)
			typeIndex := tc.m.FunctionSection[0]
			code := &tc.m.CodeSection[0]
			fc.Init(0, &tc.m.TypeSection[typeIndex], code.LocalTypes, code.Body)
			err = fc.LowerToSSA()
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
