package frontend

import (
	"fmt"
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

	nextVariable ssa.Variable

	loweringState loweringState

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
	c.ssaBuilder.Reset()
	c.loweringState.reset()
	c.nextVariable = 0

	c.wasmLocalFunctionIndex = idx
	c.wasmFunctionTyp = typ
	c.wasmFunctionLocalTypes = localTypes
	c.wasmFunctionBody = body
}

// LowerToSSA lowers the current function to SSA function which will be held by ssaBuilder.
// After calling this, the caller will be able to access the SSA info in ssa.SSABuilder pased
// when calling NewFrontendCompiler and can share them with the backend.
func (c *Compiler) LowerToSSA() error {
	// Set up the entry block.
	entryBlock := c.ssaBuilder.AllocateBasicBlock()
	c.ssaBuilder.SetCurrentBlock(entryBlock)
	// TODO: add moduleContext param as a first argument, then adjust this to 1.
	c.nextVariable = 0
	c.declareWasmFunctionParams(entryBlock)
	c.declareWasmLocals(entryBlock)

	// Set up the end block for function return.
	//endBlock := c.ssaBuilder.AllocateBasicBlock()
	//c.declareWasmFunctionReturns(endBlock)

	c.lowerBody()
	return nil
}

func (c *Compiler) declareWasmFunctionParams(entry ssa.BasicBlock) {
	for i, typ := range c.wasmFunctionTyp.Params {
		variable := c.allocateVar()

		st := wasmToSSA(typ)
		c.ssaBuilder.DeclareVariable(variable, st)

		value := entry.AddParam(st)
		c.ssaBuilder.DefineVariable(variable, value, entry)

		// TODO: put this debugging info behind flag.
		c.ssaBuilder.AnnotateVariable(variable, fmt.Sprintf("function_params[%d]", i))
	}
}

func (c *Compiler) declareWasmLocals(entry ssa.BasicBlock) {
	for i, typ := range c.wasmFunctionLocalTypes {
		variable := c.allocateVar()

		st := wasmToSSA(typ)
		c.ssaBuilder.DeclareVariable(variable, st)

		zeroInst := c.ssaBuilder.AllocateInstruction()
		switch st {
		case ssa.TypeI32:
			zeroInst.AsIconst32(0)
		case ssa.TypeI64:
			zeroInst.AsIconst64(0)
		case ssa.TypeF32:
			zeroInst.AsF32const(0)
		case ssa.TypeF64:
			zeroInst.AsF64const(0)
		default:
			panic("TODO: " + wasm.ValueTypeName(typ))
		}

		c.ssaBuilder.DefineVariable(variable, zeroInst, entry)

		// TODO: put this debugging info behind flag.
		c.ssaBuilder.AnnotateVariable(variable, fmt.Sprintf("function_locals[%d]", i))
	}
}

// TODO: I guess we don't need this.
func (c *Compiler) declareWasmFunctionReturns(endBlock ssa.BasicBlock) {
	for i, typ := range c.wasmFunctionTyp.Results {
		variable := c.allocateVar()

		st := wasmToSSA(typ)
		c.ssaBuilder.DeclareVariable(variable, st)

		value := endBlock.AddParam(st)
		c.ssaBuilder.DefineVariable(variable, value, endBlock)

		// TODO: put this debugging info behind flag.
		c.ssaBuilder.AnnotateVariable(variable, fmt.Sprintf("function_returns[%d]", i))
	}
}

func (c *Compiler) allocateVar() (ret ssa.Variable) {
	ret = c.nextVariable
	c.nextVariable++
	return
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
