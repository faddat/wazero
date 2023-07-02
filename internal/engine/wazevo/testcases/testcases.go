package testcases

import "github.com/tetratelabs/wazero/internal/wasm"

var (
	Empty              = TestCase{Name: "empty", Module: singleFunctionModule(vv, []byte{wasm.OpcodeEnd}, nil)}
	Unreachable        = TestCase{Name: "unreachable", Module: singleFunctionModule(vv, []byte{wasm.OpcodeUnreachable, wasm.OpcodeEnd}, nil)}
	OnlyReturn         = TestCase{Name: "only_return", Module: singleFunctionModule(vv, []byte{wasm.OpcodeReturn, wasm.OpcodeEnd}, nil)}
	Params             = TestCase{Name: "params", Module: singleFunctionModule(i32f32f64_v, []byte{wasm.OpcodeReturn, wasm.OpcodeEnd}, nil)}
	AddSubParamsReturn = TestCase{
		Name: "add_sub_params_return",
		Module: singleFunctionModule(i32i32_i32, []byte{
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeI32Add,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeI32Sub,
			wasm.OpcodeEnd,
		}, nil),
	}
	Locals       = TestCase{Name: "locals", Module: singleFunctionModule(vv, []byte{wasm.OpcodeEnd}, []wasm.ValueType{i32, i64, f32, f64})}
	LocalsParams = TestCase{
		Name: "locals_params",
		Module: singleFunctionModule(
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
			}, []wasm.ValueType{i32, i64, f32, f64},
		),
	}
	LocalsParamsAddReturn = TestCase{
		Name: "locals_params_add_return",
		Module: singleFunctionModule(i32_i32i32, []byte{
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeEnd,
		}, []wasm.ValueType{i32}),
	}
	SwapParamAndReturn = TestCase{
		Name: "swap_param_and_return",
		Module: singleFunctionModule(i32i32_i32i32, []byte{
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeEnd,
		}, nil),
	}
	SwapParamsAndReturn = TestCase{
		Name: "swap_params_and_return",
		Module: singleFunctionModule(i32i32_i32i32, []byte{
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
	}
	BlockBr = TestCase{
		Name: "block_br",
		Module: singleFunctionModule(vv, []byte{
			wasm.OpcodeBlock, 0,
			wasm.OpcodeBr, 0,
			wasm.OpcodeEnd,
			wasm.OpcodeEnd,
		}, []wasm.ValueType{i32, i64, f32, f64}),
	}
	BlockBrIf = TestCase{
		Name: "block_br_if",
		Module: singleFunctionModule(vv, []byte{
			wasm.OpcodeBlock, 0,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeBrIf, 0,
			wasm.OpcodeUnreachable,
			wasm.OpcodeEnd,
			wasm.OpcodeEnd,
		}, []wasm.ValueType{i32}),
	}
	LoopBr = TestCase{
		Name: "loop_br",
		Module: singleFunctionModule(vv, []byte{
			wasm.OpcodeLoop, 0,
			wasm.OpcodeBr, 0,
			wasm.OpcodeEnd,
			wasm.OpcodeEnd,
		}, []wasm.ValueType{}),
	}
	LoopBrIf = TestCase{
		Name: "loop_br_if",
		Module: singleFunctionModule(vv, []byte{
			wasm.OpcodeLoop, 0,
			wasm.OpcodeI32Const, 1,
			wasm.OpcodeBrIf, 0,
			wasm.OpcodeReturn,
			wasm.OpcodeEnd,
			wasm.OpcodeEnd,
		}, []wasm.ValueType{}),
	}
	BlockBlockBr = TestCase{
		Name: "block_block_br",
		Module: singleFunctionModule(vv, []byte{
			wasm.OpcodeBlock, 0,
			wasm.OpcodeBlock, 0,
			wasm.OpcodeBr, 1,
			wasm.OpcodeEnd,
			wasm.OpcodeEnd,
			wasm.OpcodeEnd,
		}, []wasm.ValueType{i32, i64, f32, f64}),
	}
	IfWithoutElse = TestCase{
		Name: "if_without_else",
		Module: singleFunctionModule(vv, []byte{
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeIf, 0,
			wasm.OpcodeEnd,
			wasm.OpcodeEnd,
		}, []wasm.ValueType{i32}),
	}
	IfElse = TestCase{
		Name: "if_else",
		Module: singleFunctionModule(vv, []byte{
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeIf, 0,
			wasm.OpcodeElse,
			wasm.OpcodeBr, 1,
			wasm.OpcodeEnd,
			wasm.OpcodeEnd,
		}, []wasm.ValueType{i32}),
	}
	SinglePredecessorLocalRefs = TestCase{
		Name: "single_predecessor_local_refs",
		Module: &wasm.Module{
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
	}
	MultiPredecessorLocalRef = TestCase{
		Name: "multi_predecessor_local_ref",
		Module: singleFunctionModule(i32i32_i32, []byte{
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
	}
	ReferenceValueFromUnsealedBlock = TestCase{
		Name: "reference_value_from_unsealed_block",
		Module: singleFunctionModule(i32_i32, []byte{
			wasm.OpcodeLoop, blockSignature_vv,
			// Loop will not be sealed until we reach the end,
			// so this will result in referencing the unsealed definition search.
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeReturn,
			wasm.OpcodeEnd,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeEnd,
		}, []wasm.ValueType{i32}),
	}
	ReferenceValueFromUnsealedBlock2 = TestCase{
		Name: "reference_value_from_unsealed_block2",
		Module: singleFunctionModule(i32_i32, []byte{
			wasm.OpcodeLoop, blockSignature_vv,
			wasm.OpcodeBlock, blockSignature_vv,

			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeBrIf, 1,
			wasm.OpcodeEnd,

			wasm.OpcodeEnd,
			wasm.OpcodeI32Const, 0,
			wasm.OpcodeEnd,
		}, []wasm.ValueType{}),
	}
	ReferenceValueFromUnsealedBlock3 = TestCase{
		Name: "reference_value_from_unsealed_block3",
		Module: singleFunctionModule(i32_v, []byte{
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
	}
	Call = TestCase{
		Name: "call",
		Module: &wasm.Module{
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
	}

	IntegerComparisons = TestCase{
		Name: "integer_comparisons",
		Module: singleFunctionModule(vv, []byte{
			// eq.
			wasm.OpcodeI32Const, 1,
			wasm.OpcodeI32Const, 2,
			wasm.OpcodeI32Eq,
			wasm.OpcodeI64Const, 1,
			wasm.OpcodeI64Const, 2,
			wasm.OpcodeI64Eq,
			wasm.OpcodeDrop,
			wasm.OpcodeDrop,
			// neq.
			wasm.OpcodeI32Const, 1,
			wasm.OpcodeI32Const, 2,
			wasm.OpcodeI32Ne,
			wasm.OpcodeI64Const, 1,
			wasm.OpcodeI64Const, 2,
			wasm.OpcodeI64Ne,
			wasm.OpcodeDrop,
			wasm.OpcodeDrop,
			// LtS.
			wasm.OpcodeI32Const, 1,
			wasm.OpcodeI32Const, 2,
			wasm.OpcodeI32LtS,
			wasm.OpcodeI64Const, 1,
			wasm.OpcodeI64Const, 2,
			wasm.OpcodeI64LtS,
			wasm.OpcodeDrop,
			wasm.OpcodeDrop,
			// LtU.
			wasm.OpcodeI32Const, 1,
			wasm.OpcodeI32Const, 2,
			wasm.OpcodeI32LtU,
			wasm.OpcodeI64Const, 1,
			wasm.OpcodeI64Const, 2,
			wasm.OpcodeI64LtU,
			wasm.OpcodeDrop,
			wasm.OpcodeDrop,
			// GtS.
			wasm.OpcodeI32Const, 1,
			wasm.OpcodeI32Const, 2,
			wasm.OpcodeI32GtS,
			wasm.OpcodeI64Const, 1,
			wasm.OpcodeI64Const, 2,
			wasm.OpcodeI64GtS,
			wasm.OpcodeDrop,
			wasm.OpcodeDrop,
			// GtU.
			wasm.OpcodeI32Const, 1,
			wasm.OpcodeI32Const, 2,
			wasm.OpcodeI32GtU,
			wasm.OpcodeI64Const, 1,
			wasm.OpcodeI64Const, 2,
			wasm.OpcodeI64GtU,
			wasm.OpcodeDrop,
			wasm.OpcodeDrop,
			// LeS.
			wasm.OpcodeI32Const, 1,
			wasm.OpcodeI32Const, 2,
			wasm.OpcodeI32LeS,
			wasm.OpcodeI64Const, 1,
			wasm.OpcodeI64Const, 2,
			wasm.OpcodeI64LeS,
			wasm.OpcodeDrop,
			wasm.OpcodeDrop,
			// LeU.
			wasm.OpcodeI32Const, 1,
			wasm.OpcodeI32Const, 2,
			wasm.OpcodeI32LeU,
			wasm.OpcodeI64Const, 1,
			wasm.OpcodeI64Const, 2,
			wasm.OpcodeI64LeU,
			wasm.OpcodeDrop,
			wasm.OpcodeDrop,
			// GeS.
			wasm.OpcodeI32Const, 1,
			wasm.OpcodeI32Const, 2,
			wasm.OpcodeI32GeS,
			wasm.OpcodeI64Const, 1,
			wasm.OpcodeI64Const, 2,
			wasm.OpcodeI64GeS,
			wasm.OpcodeDrop,
			wasm.OpcodeDrop,
			// GeU.
			wasm.OpcodeI32Const, 1,
			wasm.OpcodeI32Const, 2,
			wasm.OpcodeI32GeU,
			wasm.OpcodeI64Const, 1,
			wasm.OpcodeI64Const, 2,
			wasm.OpcodeI64GeU,
			wasm.OpcodeDrop,
			wasm.OpcodeDrop,
			wasm.OpcodeEnd,
		}, []wasm.ValueType{}),
	}
	IntegerShift = TestCase{
		Name: "integer_shift",
		Module: singleFunctionModule(vv, []byte{
			wasm.OpcodeI32Const, 1,
			wasm.OpcodeI32Const, 2,
			wasm.OpcodeI32Shl,
			wasm.OpcodeDrop,

			wasm.OpcodeI64Const, 1,
			wasm.OpcodeI64Const, 2,
			wasm.OpcodeI64Shl,
			wasm.OpcodeDrop,
			wasm.OpcodeEnd,
		}, []wasm.ValueType{}),
	}
	IntegerExtensions = TestCase{
		Name: "integer_extensions",
		Module: singleFunctionModule(vv, []byte{
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeI64ExtendI32S,
			wasm.OpcodeDrop,

			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeI64ExtendI32U,
			wasm.OpcodeDrop,

			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeI64Extend8S,
			wasm.OpcodeDrop,

			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeI64Extend16S,
			wasm.OpcodeDrop,

			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeI64Extend32S,
			wasm.OpcodeDrop,

			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeI32Extend8S,
			wasm.OpcodeDrop,

			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeI32Extend16S,
			wasm.OpcodeDrop,

			wasm.OpcodeEnd,
		}, []wasm.ValueType{i32, i64}),
	}

	FloatComparisons = TestCase{
		Name: "float_comparisons",
		Module: singleFunctionModule(vv, []byte{
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeF32Eq,
			wasm.OpcodeDrop,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeF32Ne,
			wasm.OpcodeDrop,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeF32Lt,
			wasm.OpcodeDrop,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeF32Gt,
			wasm.OpcodeDrop,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeF32Le,
			wasm.OpcodeDrop,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeLocalGet, 0,
			wasm.OpcodeF32Ge,
			wasm.OpcodeDrop,

			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeF64Eq,
			wasm.OpcodeDrop,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeF64Ne,
			wasm.OpcodeDrop,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeF64Lt,
			wasm.OpcodeDrop,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeF64Gt,
			wasm.OpcodeDrop,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeF64Le,
			wasm.OpcodeDrop,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeLocalGet, 1,
			wasm.OpcodeF64Ge,
			wasm.OpcodeDrop,

			wasm.OpcodeEnd,
		}, []wasm.ValueType{f32, f64}),
	}
)

type TestCase struct {
	Name   string
	Module *wasm.Module
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

var (
	vv                  = wasm.FunctionType{}
	v_i32               = wasm.FunctionType{Results: []wasm.ValueType{i32}}
	v_i32i32            = wasm.FunctionType{Results: []wasm.ValueType{i32, i32}}
	i32_v               = wasm.FunctionType{Params: []wasm.ValueType{i32}}
	i32_i32             = wasm.FunctionType{Params: []wasm.ValueType{i32}, Results: []wasm.ValueType{i32}}
	i32i32_i32          = wasm.FunctionType{Params: []wasm.ValueType{i32, i32}, Results: []wasm.ValueType{i32}}
	i32i32_i32i32       = wasm.FunctionType{Params: []wasm.ValueType{i32, i32}, Results: []wasm.ValueType{i32, i32}}
	i32_i32i32          = wasm.FunctionType{Params: []wasm.ValueType{i32}, Results: []wasm.ValueType{i32, i32}}
	i32f32f64_v         = wasm.FunctionType{Params: []wasm.ValueType{i32, f32, f64}, Results: nil}
	i64f32f64_i64f32f64 = wasm.FunctionType{Params: []wasm.ValueType{i64, f32, f64}, Results: []wasm.ValueType{i64, f32, f64}}
)

const (
	i32 = wasm.ValueTypeI32
	i64 = wasm.ValueTypeI64
	f32 = wasm.ValueTypeF32
	f64 = wasm.ValueTypeF64

	blockSignature_vv = 0x40 // 0x40 is the v_v signature in 33-bit signed. See wasm.DecodeBlockType.
)
