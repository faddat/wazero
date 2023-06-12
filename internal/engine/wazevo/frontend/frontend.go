package frontend

import (
	"bytes"
	"strings"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// Compiler is in charge of lowering Wasm to SSA IR, and does the optimization
// on top of it in architecture-independent way.
type Compiler struct {
	// Per-module data that is used across all functions.

	m       *wasm.Module
	offsets wazevoapi.OffsetData
	// ssaBuilder is a ssa.Builder used by this frontend.
	ssaBuilder ssa.Builder
	// trapBlocks maps wazevoapi.TrapCode to the corresponding BasicBlock which
	// exits the execution with the code.
	trapBlocks [wazevoapi.TrapCodeCount]ssa.BasicBlock
	signatures map[*wasm.FunctionType]*ssa.Signature

	// Followings are reset by per function.

	// wasmLocalToVariable maps the index (considered as wasm.Index of locals)
	// to the corresponding ssa.Variable.
	wasmLocalToVariable    map[wasm.Index]ssa.Variable
	loweringState          loweringState
	wasmLocalFunctionIndex wasm.Index
	wasmFunctionTyp        *wasm.FunctionType
	wasmFunctionLocalTypes []wasm.ValueType
	wasmFunctionBody       []byte
	// br is reused during lowering.
	br *bytes.Reader

	execCtxPtrValue, moduleCtxPtrValue ssa.Value
}

// NewFrontendCompiler returns a frontend Compiler.
func NewFrontendCompiler(od wazevoapi.OffsetData, m *wasm.Module, ssaBuilder ssa.Builder) *Compiler {
	c := &Compiler{
		m:                   m,
		offsets:             od,
		ssaBuilder:          ssaBuilder,
		br:                  bytes.NewReader(nil),
		wasmLocalToVariable: make(map[wasm.Index]ssa.Variable),
	}

	c.signatures = make(map[*wasm.FunctionType]*ssa.Signature, len(m.TypeSection))
	for i := range m.TypeSection {
		wasmSig := &m.TypeSection[i]
		sig := &ssa.Signature{
			ID: ssa.SignatureID(i),
			// +2 to pass moduleContextPtr and executionContextPtr. See the inline comment LowerToSSA.
			Params:  make([]ssa.Type, len(wasmSig.Params)+2),
			Results: make([]ssa.Type, len(wasmSig.Results)),
		}
		sig.Params[0] = executionContextPtrTyp
		sig.Params[1] = moduleContextPtrTyp
		for j, typ := range wasmSig.Params {
			sig.Params[j+2] = wasmToSSA(typ)
		}
		for j, typ := range wasmSig.Results {
			sig.Results[j] = wasmToSSA(typ)
		}
		c.signatures[wasmSig] = sig
		c.ssaBuilder.DeclareSignature(sig)
	}
	return c
}

// Init initializes the state of frontendCompiler and make it ready for a next function.
func (c *Compiler) Init(idx wasm.Index, typ *wasm.FunctionType, localTypes []wasm.ValueType, body []byte) {
	c.ssaBuilder.Reset()
	c.loweringState.reset()
	c.trapBlocks = [wazevoapi.TrapCodeCount]ssa.BasicBlock{}

	c.wasmLocalFunctionIndex = idx
	c.wasmFunctionTyp = typ
	c.wasmFunctionLocalTypes = localTypes
	c.wasmFunctionBody = body
}

// Note: this assumes 64-bit platform (I believe we won't have 32-bit backend ;)).
const executionContextPtrTyp, moduleContextPtrTyp = ssa.TypeI64, ssa.TypeI64

// LowerToSSA lowers the current function to SSA function which will be held by ssaBuilder.
// After calling this, the caller will be able to access the SSA info in ssa.SSABuilder pased
// when calling NewFrontendCompiler and can share them with the backend.
//
// Note that this only does the naive lowering, and do not do any optimization, instead the caller is expected to do so.
func (c *Compiler) LowerToSSA() error {
	builder := c.ssaBuilder

	// Set up the entry block.
	entryBlock := builder.AllocateBasicBlock()
	builder.SetCurrentBlock(entryBlock)

	// Functions always take two parameters in addition to Wasm-level parameters:
	//
	// 	1. moduleContextPtr: pointer to the *moduleContextOpaque in wazevo package.
	//	  This will be used to access memory, etc. Also, this will be used during host function calls.
	//
	//  2. executionContextPtr: pointer to the *executionContext in wazevo package.
	//    This will be used to exit the execution in the face of trap, plus used for host function calls.
	//
	// Note: it's clear that sometimes a function won't need them. For example,
	//  if the function doesn't trap and doesn't make function call, then
	// 	we might be able to eliminate the parameter. However, if that function
	//	can be called via call_indirect, then we cannot eliminate because the
	//  signature won't match with the expected one.
	//  TODO: maybe there's some way to do this optimization without glitches, but so far I have no clue about the feasibility.
	//
	// Note: In Wasmtime or many other runtimes, moduleContextPtr is called "vmContext". Also note that `moduleContextPtr`
	//  is wazero-specific since other runtimes can naturally use the OS-level signal to do this job thanks to the fact that
	//  they can use native stack vs wazero cannot use Go-routine stack and have to use Go-runtime allocated []byte as a stack.
	_, execCtxPtrValue := entryBlock.AddParam(builder, executionContextPtrTyp)
	_, moduleCtxPtrValue := entryBlock.AddParam(builder, moduleContextPtrTyp)
	builder.AnnotateValue(execCtxPtrValue, "exec_ctx")
	builder.AnnotateValue(moduleCtxPtrValue, "module_ctx")
	c.execCtxPtrValue, c.moduleCtxPtrValue = execCtxPtrValue, moduleCtxPtrValue

	for i, typ := range c.wasmFunctionTyp.Params {
		st := wasmToSSA(typ)
		variable, _ := entryBlock.AddParam(builder, st)
		c.wasmLocalToVariable[wasm.Index(i)] = variable
	}
	c.declareWasmLocals(entryBlock)

	c.lowerBody(entryBlock)
	c.emitTrapBlocks()
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
		blk.AddParam(c.ssaBuilder, st)
	}
}

// formatBuilder outputs the constructed SSA function as a string with a source information.
func (c *Compiler) formatBuilder() string {
	// TODO: use source position to add the Wasm-level source info.

	builder := c.ssaBuilder

	str := strings.Builder{}

	usedSigs := builder.UsedSignatures()
	if len(usedSigs) > 0 {
		str.WriteByte('\n')
		str.WriteString("signatures:\n")
		for _, sig := range usedSigs {
			str.WriteByte('\t')
			str.WriteString(sig.String())
			str.WriteByte('\n')
		}
	}

	for _, b := range builder.Blocks() {
		str.WriteByte('\n')
		str.WriteString(b.FormatHeader(builder))
		str.WriteByte('\n')
		for cur := b.Root(); cur != nil; cur = cur.Next() {
			str.WriteByte('\t')
			str.WriteString(cur.Format(builder))
			str.WriteByte('\n')
		}
	}
	return str.String()
}

func (c *Compiler) getOrCreateTrapBlock(code wazevoapi.TrapCode) ssa.BasicBlock {
	blk := c.trapBlocks[code]
	if blk == nil {
		blk = c.ssaBuilder.AllocateBasicBlock()
		c.trapBlocks[code] = blk
	}
	return blk
}

func (c *Compiler) emitTrapBlocks() {
	builder := c.ssaBuilder
	for trapCode := wazevoapi.TrapCode(0); trapCode < wazevoapi.TrapCodeCount; trapCode++ {
		blk := c.trapBlocks[trapCode]
		if blk == nil {
			continue
		}
		builder.SetCurrentBlock(blk)

		trapCodeInstr := builder.AllocateInstruction()
		trapCodeInstr.AsIconst32(uint32(trapCode))
		builder.InsertInstruction(trapCodeInstr)
		trapCodeVal, _ := trapCodeInstr.Returns()

		execCtx := c.execCtxPtrValue
		store := builder.AllocateInstruction()
		store.AsStore(trapCodeVal, execCtx, c.offsets.ExecutionContextTrapCodeOffset.U32())
		builder.InsertInstruction(store)

		trap := builder.AllocateInstruction()
		trap.AsTrap()
		builder.InsertInstruction(trap)
	}
}
