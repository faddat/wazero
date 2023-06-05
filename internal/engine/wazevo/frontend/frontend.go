package frontend

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// Compiler is in charge of lowering Wasm to SSA IR, and does the optimization
// on top of it in architecture-independent way.
type Compiler struct {
	// Per-module data that is used across all functions.
	m *wasm.Module
	// ssaBuilder is a ssa.Builder used by this frontend.
	ssaBuilder ssa.Builder

	wasmFunctionParamBeginInEntryBlock ssa.Variable

	// Followings are reset by per function.

	wasmLocalFunctionIndex wasm.Index
	wasmFunctionTyp        *wasm.FunctionType
	wasmFunctionLocalTypes []wasm.ValueType
	wasmFunctionBody       []byte
}

// NewFrontendCompiler returns a frontend Compiler.
func NewFrontendCompiler(m *wasm.Module, ssaBuilder ssa.Builder) *Compiler {
	return &Compiler{m: m, ssaBuilder: ssaBuilder}
}

// Init initializes the state of frontendCompiler and make it ready for a next function.
func (c *Compiler) Init(idx wasm.Index, typ *wasm.FunctionType, localTypes []wasm.ValueType, body []byte) {
	c.ssaBuilder.Reset() // Clears the previous state.
	c.wasmLocalFunctionIndex = idx
	c.wasmFunctionTyp = typ
	c.wasmFunctionLocalTypes = localTypes
	c.wasmFunctionBody = body
}

// LowerToSSA lowers the current function to SSA function which will be held by ssaBuilder.
// After calling this, the caller will be able to access the SSA info in ssa.SSABuilder pased
// when calling NewFrontendCompiler and can share them with the backend.
func (c *Compiler) LowerToSSA() error {
	entryBlock := c.ssaBuilder.AllocateBasicBlock()
	c.ssaBuilder.SetCurrentBlock(entryBlock)

	// TODO: add moduleContext param as a first argument, then adjust this to 1.
	c.wasmFunctionParamBeginInEntryBlock = 0
	c.declareFunctionParams(entryBlock)

	// Declare locals.
	return nil
}

func (c *Compiler) declareFunctionParams(entry ssa.BasicBlock) {
	var paramVar = c.wasmFunctionParamBeginInEntryBlock
	for _, pt := range c.wasmFunctionTyp.Params {
		st := wasmToSSA(pt)
		c.ssaBuilder.DeclareVariable(paramVar, st)

		value := entry.AddParam(st)
		c.ssaBuilder.DefineVariable(paramVar, value, entry)
	}
}

func wasmToSSA(vt wasm.ValueType) ssa.Type {
	switch vt {
	case wasm.ValueTypeI32:
		return ssa.TypeI32
	case wasm.ValueTypeI64:
		return ssa.TypeI64
	case wasm.ValueTypeF32:
		return ssa.TypeF32
	case wasm.ValueTypeF64:
		return ssa.TypeF64
	default:
		panic("TODO: " + wasm.ValueTypeName(vt))
	}
}
