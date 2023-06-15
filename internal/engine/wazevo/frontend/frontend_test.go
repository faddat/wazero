package frontend

import (
	"fmt"
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
	v_i32i32 := wasm.FunctionType{Results: []wasm.ValueType{i32, i32}}
	i32_v := wasm.FunctionType{Params: []wasm.ValueType{i32}}
	i32_i32 := wasm.FunctionType{Params: []wasm.ValueType{i32}, Results: []wasm.ValueType{i32}}
	i32i32_i32 := wasm.FunctionType{Params: []wasm.ValueType{i32, i32}, Results: []wasm.ValueType{i32}}
	i32i32_i32i32 := wasm.FunctionType{Params: []wasm.ValueType{i32, i32}, Results: []wasm.ValueType{i32, i32}}
	i32_i32i32 := wasm.FunctionType{Params: []wasm.ValueType{i32}, Results: []wasm.ValueType{i32, i32}}
	i32f32f64_v := wasm.FunctionType{Params: []wasm.ValueType{i32, f32, f64}, Results: nil}
	i64f32f64_i64f32f64 := wasm.FunctionType{Params: []wasm.ValueType{i64, f32, f64}, Results: []wasm.ValueType{i64, f32, f64}}

	for _, tc := range []struct {
		name string
		// m is the *wasm.Module who
		m *wasm.Module
		// targetIndex is the index of a local function to be compiled in this test.
		targetIndex wasm.Index
		// exp is the *unoptimized* expected SSA IR for the function m.FunctionSection[targetIndex].
		exp string
		// expAfterOpt is not empty when we want to check the result after optimization passes.
		expAfterOpt string
	}{
		{
			name: "empty", m: singleFunctionModule(vv, []byte{wasm.OpcodeEnd}, nil),
			exp: `
blk0: (exec_ctx:i64, module_ctx:i64)
	Jump blk_ret
`,
		},
		{
			name: "unreachable", m: singleFunctionModule(vv, []byte{wasm.OpcodeUnreachable, wasm.OpcodeEnd}, nil),
			exp: `
blk0: (exec_ctx:i64, module_ctx:i64)
	Jump blk1

blk1: () <-- (blk0)
	v2:i32 = Iconst_32 0x0
	Store v2, exec_ctx, 0x0
	Trap
`,
		},
		{
			name: "only return", m: singleFunctionModule(vv, []byte{wasm.OpcodeReturn, wasm.OpcodeEnd}, nil),
			exp: `
blk0: (exec_ctx:i64, module_ctx:i64)
	Return
`,
		},
		{
			name: "params", m: singleFunctionModule(i32f32f64_v, []byte{wasm.OpcodeReturn, wasm.OpcodeEnd}, nil),
			exp: `
blk0: (exec_ctx:i64, module_ctx:i64, v2:i32, v3:f32, v4:f64)
	Return
`,
		},
		{
			name: "add/sub params return", m: singleFunctionModule(i32i32_i32, []byte{
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeLocalGet, 1,
				wasm.OpcodeI32Add,
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeI32Sub,
				wasm.OpcodeEnd,
			}, nil),
			exp: `
blk0: (exec_ctx:i64, module_ctx:i64, v2:i32, v3:i32)
	v4:i32 = Iadd v2, v3
	v5:i32 = Isub v4, v2
	Jump blk_ret, v5
`,
		},
		{
			name: "locals", m: singleFunctionModule(vv, []byte{wasm.OpcodeEnd},
				[]wasm.ValueType{i32, i64, f32, f64}),
			exp: `
blk0: (exec_ctx:i64, module_ctx:i64)
	v2:i32 = Iconst_32 0x0
	v3:i64 = Iconst_64 0x0
	v4:f32 = F32const 0.000000
	v5:f64 = F64const 0.000000
	Jump blk_ret
`,
			expAfterOpt: `
blk0: (exec_ctx:i64, module_ctx:i64)
	Jump blk_ret
`,
		},
		{
			name: "locals + params", m: singleFunctionModule(
				i64f32f64_i64f32f64,
				[]byte{
					wasm.OpcodeLocalGet, 0,
					wasm.OpcodeLocalGet, 0,
					wasm.OpcodeI64Add,
					wasm.OpcodeLocalGet, 0,
					wasm.OpcodeI64Sub,

					wasm.OpcodeLocalGet, 1,
					wasm.OpcodeLocalGet, 1,
					wasm.OpcodeF32Add,
					wasm.OpcodeLocalGet, 1,
					wasm.OpcodeF32Sub,

					wasm.OpcodeLocalGet, 2,
					wasm.OpcodeLocalGet, 2,
					wasm.OpcodeF64Add,
					wasm.OpcodeLocalGet, 2,
					wasm.OpcodeF64Sub,

					wasm.OpcodeEnd,
				}, []wasm.ValueType{i32, i64, f32, f64}),
			exp: `
blk0: (exec_ctx:i64, module_ctx:i64, v2:i64, v3:f32, v4:f64)
	v5:i32 = Iconst_32 0x0
	v6:i64 = Iconst_64 0x0
	v7:f32 = F32const 0.000000
	v8:f64 = F64const 0.000000
	v9:i64 = Iadd v2, v2
	v10:i64 = Isub v9, v2
	v11:f32 = Fadd v3, v3
	v12:f32 = Fsub v11, v3
	v13:f64 = Fadd v4, v4
	v14:f64 = Fsub v13, v4
	Jump blk_ret, v10, v12, v14
`,
			expAfterOpt: `
blk0: (exec_ctx:i64, module_ctx:i64, v2:i64, v3:f32, v4:f64)
	v9:i64 = Iadd v2, v2
	v10:i64 = Isub v9, v2
	v11:f32 = Fadd v3, v3
	v12:f32 = Fsub v11, v3
	v13:f64 = Fadd v4, v4
	v14:f64 = Fsub v13, v4
	Jump blk_ret, v10, v12, v14
`,
		},
		{
			name: "locals + params + add return", m: singleFunctionModule(i32_i32i32, []byte{
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeLocalGet, 1,
				wasm.OpcodeEnd,
			},
				[]wasm.ValueType{i32}),
			exp: `
blk0: (exec_ctx:i64, module_ctx:i64, v2:i32)
	v3:i32 = Iconst_32 0x0
	Jump blk_ret, v2, v3
`,
		},
		{
			name: "swap param and return", m: singleFunctionModule(i32i32_i32i32, []byte{
				wasm.OpcodeLocalGet, 1,
				wasm.OpcodeLocalGet, 0,
				wasm.OpcodeEnd,
			}, nil),
			exp: `
blk0: (exec_ctx:i64, module_ctx:i64, v2:i32, v3:i32)
	Jump blk_ret, v3, v2
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
blk0: (exec_ctx:i64, module_ctx:i64, v2:i32, v3:i32)
	Jump blk1

blk1: () <-- (blk0)
	Jump blk_ret, v3, v2
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
blk0: (exec_ctx:i64, module_ctx:i64)
	v2:i32 = Iconst_32 0x0
	v3:i64 = Iconst_64 0x0
	v4:f32 = F32const 0.000000
	v5:f64 = F64const 0.000000
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
blk0: (exec_ctx:i64, module_ctx:i64)
	v2:i32 = Iconst_32 0x0
	Brz v2, blk1
	Jump blk2

blk1: () <-- (blk0)
	Jump blk_ret

blk2: () <-- (blk0)
	Jump blk3

blk3: () <-- (blk2)
	v3:i32 = Iconst_32 0x0
	Store v3, exec_ctx, 0x0
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
blk0: (exec_ctx:i64, module_ctx:i64)
	Jump blk1

blk1: () <-- (blk0,blk1)
	Jump blk1

blk2: ()
	Jump blk_ret
`,
			expAfterOpt: `
blk0: (exec_ctx:i64, module_ctx:i64)
	Jump blk1

blk1: () <-- (blk0,blk1)
	Jump blk1
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
blk0: (exec_ctx:i64, module_ctx:i64)
	Jump blk1

blk1: () <-- (blk0,blk1)
	v2:i32 = Iconst_32 0x1
	Brz v2, blk1
	Jump blk3

blk2: ()
	Jump blk_ret

blk3: () <-- (blk1)
	Return
`,
			expAfterOpt: `
blk0: (exec_ctx:i64, module_ctx:i64)
	Jump blk1

blk1: () <-- (blk0,blk1)
	v2:i32 = Iconst_32 0x1
	Brz v2, blk1
	Jump blk3

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
blk0: (exec_ctx:i64, module_ctx:i64)
	v2:i32 = Iconst_32 0x0
	v3:i64 = Iconst_64 0x0
	v4:f32 = F32const 0.000000
	v5:f64 = F64const 0.000000
	Jump blk1

blk1: () <-- (blk0,blk2)
	Jump blk_ret

blk2: ()
	Jump blk1
`,
			expAfterOpt: `
blk0: (exec_ctx:i64, module_ctx:i64)
	Jump blk1

blk1: () <-- (blk0)
	Jump blk_ret
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
blk0: (exec_ctx:i64, module_ctx:i64)
	v2:i32 = Iconst_32 0x0
	Brz v2, blk2
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
blk0: (exec_ctx:i64, module_ctx:i64)
	v2:i32 = Iconst_32 0x0
	Brz v2, blk2
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
blk0: (exec_ctx:i64, module_ctx:i64)
	v2:i32 = Iconst_32 0x0
	v3:i32 = Iconst_32 0x0
	v4:i32 = Iconst_32 0x0
	Brz v2, blk2
	Jump blk1

blk1: () <-- (blk0)
	Return v4

blk2: () <-- (blk0)
	Jump blk3

blk3: () <-- (blk2)
	Jump blk_ret, v2
`,
			expAfterOpt: `
blk0: (exec_ctx:i64, module_ctx:i64)
	v2:i32 = Iconst_32 0x0
	v4:i32 = Iconst_32 0x0
	Brz v2, blk2
	Jump blk1

blk1: () <-- (blk0)
	Return v4

blk2: () <-- (blk0)
	Jump blk3

blk3: () <-- (blk2)
	Jump blk_ret, v2
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
blk0: (exec_ctx:i64, module_ctx:i64, v2:i32, v3:i32)
	v4:i32 = Iconst_32 0x0
	Brz v2, blk2
	Jump blk1

blk1: () <-- (blk0)
	Jump blk3, v2

blk2: () <-- (blk0)
	Jump blk3, v3

blk3: (v5:i32) <-- (blk1,blk2)
	Jump blk_ret, v5
`,
			expAfterOpt: `
blk0: (exec_ctx:i64, module_ctx:i64, v2:i32, v3:i32)
	Brz v2, blk2
	Jump blk1

blk1: () <-- (blk0)
	Jump blk3, v2

blk2: () <-- (blk0)
	Jump blk3, v3

blk3: (v5:i32) <-- (blk1,blk2)
	Jump blk_ret, v5
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
blk0: (exec_ctx:i64, module_ctx:i64, v2:i32)
	v3:i32 = Iconst_32 0x0
	Jump blk1, v2

blk1: (v4:i32) <-- (blk0)
	Return v4

blk2: (v5:i32)
	Jump blk_ret, v5
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
blk0: (exec_ctx:i64, module_ctx:i64, v2:i32)
	Jump blk1, v2

blk1: (v3:i32) <-- (blk0,blk1)
	Brz v3, blk1, v3
	Jump blk4

blk2: () <-- (blk3)
	v4:i32 = Iconst_32 0x0
	Jump blk_ret, v4

blk3: () <-- (blk4)
	Jump blk2

blk4: () <-- (blk1)
	Jump blk3
`,
			expAfterOpt: `
blk0: (exec_ctx:i64, module_ctx:i64, v2:i32)
	Jump blk1

blk1: () <-- (blk0,blk1)
	Brz v2, blk1
	Jump blk4

blk2: () <-- (blk3)
	v4:i32 = Iconst_32 0x0
	Jump blk_ret, v4

blk3: () <-- (blk4)
	Jump blk2

blk4: () <-- (blk1)
	Jump blk3
`,
		},
		{
			name: "reference value from unsealed block - #3",
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
blk0: (exec_ctx:i64, module_ctx:i64, v2:i32)
	Jump blk1, v2

blk1: (v3:i32) <-- (blk0,blk3)
	Brz v3, blk_ret
	Jump blk4

blk2: ()
	Jump blk_ret

blk3: () <-- (blk4)
	v4:i32 = Iconst_32 0x1
	Jump blk1, v4

blk4: () <-- (blk1)
	Jump blk3
`,
			expAfterOpt: `
blk0: (exec_ctx:i64, module_ctx:i64, v2:i32)
	Jump blk1, v2

blk1: (v3:i32) <-- (blk0,blk3)
	Brz v3, blk_ret
	Jump blk4

blk3: () <-- (blk4)
	v4:i32 = Iconst_32 0x1
	Jump blk1, v4

blk4: () <-- (blk1)
	Jump blk3
`,
		},
		{
			name: "call",
			m: &wasm.Module{
				TypeSection:     []wasm.FunctionType{v_i32i32, v_i32, i32i32_i32, i32_i32i32},
				FunctionSection: []wasm.Index{0, 1, 2, 3},
				CodeSection: []wasm.Code{
					{Body: []byte{
						// Call v_i32.
						wasm.OpcodeCall, 1,
						// Call i32i32_i32.
						wasm.OpcodeI32Const, 5,
						wasm.OpcodeCall, 2,
						// Call i32_i32i32.
						wasm.OpcodeCall, 3,
						wasm.OpcodeEnd,
					}},
					{Body: []byte{wasm.OpcodeI32Const, 1, wasm.OpcodeEnd}},
					{Body: []byte{wasm.OpcodeLocalGet, 0, wasm.OpcodeEnd}},
					{Body: []byte{wasm.OpcodeLocalGet, 0, wasm.OpcodeLocalGet, 0, wasm.OpcodeEnd}},
				},
			},
			exp: `
signatures:
	sig1: i64i64_i32
	sig2: i64i64i32i32_i32
	sig3: i64i64i32_i32i32

blk0: (exec_ctx:i64, module_ctx:i64)
	Store module_ctx, exec_ctx, 0x8
	v2:i32 = Call f1:sig1, exec_ctx, module_ctx
	v3:i32 = Iconst_32 0x5
	Store module_ctx, exec_ctx, 0x8
	v4:i32 = Call f2:sig2, exec_ctx, module_ctx, v2, v3
	Store module_ctx, exec_ctx, 0x8
	v5:i32, v6:i32 = Call f3:sig3, exec_ctx, module_ctx, v4
	Jump blk_ret, v5, v6
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
			typeIndex := tc.m.FunctionSection[tc.targetIndex]
			code := &tc.m.CodeSection[tc.targetIndex]
			fc.Init(tc.targetIndex, &tc.m.TypeSection[typeIndex], code.LocalTypes, code.Body)

			err = fc.LowerToSSA()
			require.NoError(t, err)

			actual := fc.formatBuilder()
			fmt.Println(actual)
			require.Equal(t, tc.exp, actual)

			b.Optimize()
			if expAfterOpt := tc.expAfterOpt; expAfterOpt != "" {
				actualAfterOpt := fc.formatBuilder()
				fmt.Println(actualAfterOpt)
				require.Equal(t, expAfterOpt, actualAfterOpt)
			}
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
