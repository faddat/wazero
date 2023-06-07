package frontend

import (
	"bytes"
	"strings"

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

	// wasmLocalToVariable maps the index (considered as wasm.Index of locals)
	// to the corresponding ssa.Variable.
	wasmLocalToVariable map[wasm.Index]ssa.Variable
	loweringState       loweringState

	// Followings are reset by per function.

	wasmLocalFunctionIndex wasm.Index
	wasmFunctionTyp        *wasm.FunctionType
	wasmFunctionLocalTypes []wasm.ValueType
	wasmFunctionBody       []byte

	// br is reused during lowering.
	br *bytes.Reader
}

// NewFrontendCompiler returns a frontend Compiler.
func NewFrontendCompiler(m *wasm.Module, ssaBuilder ssa.Builder) *Compiler {
	return &Compiler{
		m:                   m,
		ssaBuilder:          ssaBuilder,
		br:                  bytes.NewReader(nil),
		wasmLocalToVariable: make(map[wasm.Index]ssa.Variable),
	}
}

// Init initializes the state of frontendCompiler and make it ready for a next function.
func (c *Compiler) Init(idx wasm.Index, typ *wasm.FunctionType, localTypes []wasm.ValueType, body []byte) {
	c.ssaBuilder.Reset()
	c.loweringState.reset()

	c.wasmLocalFunctionIndex = idx
	c.wasmFunctionTyp = typ
	c.wasmFunctionLocalTypes = localTypes
	c.wasmFunctionBody = body
}

// LowerToSSA lowers the current function to SSA function which will be held by ssaBuilder.
// After calling this, the caller will be able to access the SSA info in ssa.SSABuilder pased
// when calling NewFrontendCompiler and can share them with the backend.
//
// Note that this only does the naive lowering, and do not do any optimization, instead the caller is expected to do so.
func (c *Compiler) LowerToSSA() error {
	// Set up the entry block.
	entryBlock := c.ssaBuilder.AllocateBasicBlock()
	c.ssaBuilder.SetCurrentBlock(entryBlock)
	// TODO: add moduleContext param as a first argument.
	for i, typ := range c.wasmFunctionTyp.Params {
		st := wasmToSSA(typ)
		variable := entryBlock.AddParam(c.ssaBuilder, st)
		c.wasmLocalToVariable[wasm.Index(i)] = variable
	}
	c.declareWasmLocals(entryBlock)

	c.lowerBody(entryBlock)
	return nil
}

func (c *Compiler) localVariable(index wasm.Index) ssa.Variable {
	return c.wasmLocalToVariable[index]
}

func (c *Compiler) declareWasmLocals(entry ssa.BasicBlock) {
	localCount := wasm.Index(len(c.wasmFunctionTyp.Params))
	for i, typ := range c.wasmFunctionLocalTypes {
		st := wasmToSSA(typ)
		variable := c.ssaBuilder.DeclareVariable(st)
		c.wasmLocalToVariable[wasm.Index(i)+localCount] = variable

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

		c.ssaBuilder.InsertInstruction(zeroInst)
		value, _ := zeroInst.Returns()
		c.ssaBuilder.DefineVariable(variable, value, entry)
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

func (c *Compiler) addBlockParamsFromWasmTypes(tps []wasm.ValueType, blk ssa.BasicBlock) {
	for _, typ := range tps {
		st := wasmToSSA(typ)
		_ = blk.AddParam(c.ssaBuilder, st)
	}
}

// formatBuilder outputs the constructed SSA function as a string with a source information.
func (c *Compiler) formatBuilder() string {
	// TODO: use source position to add the Wasm-level source info.

	builder := c.ssaBuilder

	str := strings.Builder{}
	for _, b := range builder.Blocks() {
		header := b.String()
		str.WriteByte('\n')
		str.WriteString(header)
		str.WriteByte('\n')
		for cur := b.Root(); cur != nil; cur = cur.Next() {
			str.WriteByte('\t')
			str.WriteString(cur.String())
			str.WriteByte('\n')
		}
	}
	return str.String()
}
