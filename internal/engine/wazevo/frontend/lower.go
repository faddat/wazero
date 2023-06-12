package frontend

import (
	"fmt"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
	"github.com/tetratelabs/wazero/internal/leb128"
	"github.com/tetratelabs/wazero/internal/wasm"
)

type (
	loweringState struct {
		values           []ssa.Value
		controlFrames    []controlFrame
		unreachable      bool
		unreachableDepth int
		pc               int
	}
	controlFrame struct {
		kind controlFrameKind
		// originalStackLen holds the number of values on the Wasm stack
		// when start executing this control frame minus params for the block.
		originalStackLenWithoutParam int
		// blk is the loop header if this is loop, and is the else-block if this is an if frame.
		blk,
		// followingBlock is the basic block we enter if we reach "end" of block.
		followingBlock ssa.BasicBlock
		blockType *wasm.FunctionType
		// clonedArgs hold the arguments to Else block.
		clonedArgs []ssa.Value
	}

	controlFrameKind byte
)

const (
	controlFrameKindFunction = iota + 1
	controlFrameKindLoop
	controlFrameKindIfWithElse
	controlFrameKindIfWithoutElse
	controlFrameKindBlock
)

func (ctrl *controlFrame) isLoop() bool {
	return ctrl.kind == controlFrameKindLoop
}

func (l *loweringState) reset() {
	l.values = l.values[:0]
	l.controlFrames = l.controlFrames[:0]
	l.pc = 0
	l.unreachable = false
	l.unreachableDepth = 0
}

func (l *loweringState) pop() (ret ssa.Value) {
	tail := len(l.values) - 1
	ret = l.values[tail]
	l.values = l.values[:tail]
	return
}

func (l *loweringState) push(ret ssa.Value) {
	l.values = append(l.values, ret)
}

func (l *loweringState) nPeek(n int) []ssa.Value {
	if n == 0 {
		return nil
	}
	tail := len(l.values)
	view := l.values[tail-n : tail]
	return view
}

func (l *loweringState) nPeekDup(n int) []ssa.Value {
	if n == 0 {
		return nil
	}
	tail := len(l.values)
	view := l.values[tail-n : tail]
	return cloneValuesList(view)
}

func (l *loweringState) ctrlPop() (ret controlFrame) {
	tail := len(l.controlFrames) - 1
	ret = l.controlFrames[tail]
	l.controlFrames = l.controlFrames[:tail]
	return
}

func (l *loweringState) ctrlPush(ret controlFrame) {
	l.controlFrames = append(l.controlFrames, ret)
}

func (l *loweringState) ctrlPeekAt(n int) (ret *controlFrame) {
	tail := len(l.controlFrames) - 1
	return &l.controlFrames[tail-n]
}

func (c *Compiler) lowerBody(entryBlk ssa.BasicBlock) {
	c.ssaBuilder.Seal(entryBlk)

	// Pushes the empty control frame which corresponds to the function return.
	c.loweringState.ctrlPush(controlFrame{
		kind:           controlFrameKindFunction,
		blockType:      c.wasmFunctionTyp,
		followingBlock: ssa.BasicBlockReturn,
	})

	for c.loweringState.pc < len(c.wasmFunctionBody) {
		op := c.wasmFunctionBody[c.loweringState.pc]
		// TODO: delete prints.
		fmt.Println("--------- Translated " + wasm.InstructionName(op) + " --------")
		c.lowerOpcode(op)
		fmt.Println(c.formatBuilder())
		fmt.Println("--------------------------")
		c.loweringState.pc++
	}
}

func (c *Compiler) lowerOpcode(op wasm.Opcode) {
	builder := c.ssaBuilder
	state := &c.loweringState
	switch op {
	case wasm.OpcodeI32Const:
		c := c.readI32s()
		if state.unreachable {
			return
		}

		iconst := builder.AllocateInstruction()
		iconst.AsIconst32(uint32(c))
		builder.InsertInstruction(iconst)
		value, _ := iconst.Returns()
		state.push(value)

	case wasm.OpcodeLocalGet:
		index := c.readI32u()
		if state.unreachable {
			return
		}
		variable := c.localVariable(index)
		v := builder.FindValue(variable)
		state.push(v)
	case wasm.OpcodeLocalSet:
		index := c.readI32u()
		if state.unreachable {
			return
		}
		variable := c.localVariable(index)
		v := state.pop()
		builder.DefineVariableInCurrentBB(variable, v)
	case wasm.OpcodeBlock:
		// Note: we do not need to create a BB for this as that would always have only one predecessor
		// which is the current BB, and therefore it's always ok to merge them in any way.

		bt := c.readBlockType()

		if state.unreachable {
			state.unreachableDepth++
			return
		}

		followingBlk := builder.AllocateBasicBlock()
		c.addBlockParamsFromWasmTypes(bt.Results, followingBlk)

		state.ctrlPush(controlFrame{
			kind:                         controlFrameKindBlock,
			originalStackLenWithoutParam: len(state.values) - len(bt.Params),
			followingBlock:               followingBlk,
			blockType:                    bt,
		})

	case wasm.OpcodeLoop:
		bt := c.readBlockType()

		if state.unreachable {
			state.unreachableDepth++
			return
		}

		loopHeader, afterLoopBlock := builder.AllocateBasicBlock(), builder.AllocateBasicBlock()
		c.addBlockParamsFromWasmTypes(bt.Params, loopHeader)
		c.addBlockParamsFromWasmTypes(bt.Results, afterLoopBlock)

		state.ctrlPush(controlFrame{
			originalStackLenWithoutParam: len(state.values) - len(bt.Params),
			kind:                         controlFrameKindLoop,
			blk:                          loopHeader,
			followingBlock:               afterLoopBlock,
			blockType:                    bt,
		})

		var args []ssa.Value
		if len(bt.Params) > 0 {
			args = cloneValuesList(state.values[len(state.values)-1-len(bt.Params):])
		}

		// Insert the jump to the header of loop.
		br := builder.AllocateInstruction()
		br.AsJump(args, loopHeader)
		builder.InsertInstruction(br)
		loopHeader.AddPred(builder.CurrentBlock(), br)

		builder.SetCurrentBlock(loopHeader)

	case wasm.OpcodeIf:
		bt := c.readBlockType()

		if state.unreachable {
			state.unreachableDepth++
			return
		}

		v := state.pop()
		thenBlk, elseBlk, followingBlk := builder.AllocateBasicBlock(), builder.AllocateBasicBlock(), builder.AllocateBasicBlock()

		currentBlk := builder.CurrentBlock()

		// We do not make the Wasm-level block parameters as SSA-level block params,
		// since they won't be phi and the definition is unique.

		// On the other hand, the following block after if-else-end will likely have
		// multiple definitions (one in Then and another in Else blocks).
		c.addBlockParamsFromWasmTypes(bt.Results, followingBlk)

		var args []ssa.Value
		if len(bt.Params) > 0 {
			args = cloneValuesList(state.values[len(state.values)-1-len(bt.Params):])
		}

		// Insert the conditional jump to the Else block.
		brz := builder.AllocateInstruction()
		brz.AsBrz(v, nil, elseBlk)
		builder.InsertInstruction(brz)
		elseBlk.AddPred(currentBlk, brz)

		// Then, insert the jump to the Then block.
		br := builder.AllocateInstruction()
		br.AsJump(nil, thenBlk)
		builder.InsertInstruction(br)
		thenBlk.AddPred(currentBlk, br)

		state.ctrlPush(controlFrame{
			kind:                         controlFrameKindIfWithoutElse,
			originalStackLenWithoutParam: len(state.values) - len(bt.Params),
			blk:                          elseBlk,
			followingBlock:               followingBlk,
			blockType:                    bt,
			clonedArgs:                   args,
		})

		builder.SetCurrentBlock(thenBlk)

		// Then and Else (if exists) have only one predecessor.
		builder.Seal(thenBlk)
		builder.Seal(elseBlk)
	case wasm.OpcodeElse:
		ifctrl := state.ctrlPeekAt(0)
		ifctrl.kind = controlFrameKindIfWithElse

		if unreachable := state.unreachable; unreachable && state.unreachableDepth > 0 {
			// If it is currently in unreachable and is a nested if,
			// we just remove the entire else block.
			return
		}

		if !state.unreachable {
			// If this Then block is currently reachable, we have to insert the branching to the following BB.
			followingBlk := ifctrl.followingBlock // == the BB after if-then-else.
			args := c.loweringState.nPeekDup(len(ifctrl.blockType.Results))
			c.insertJumpToBlock(args, builder.CurrentBlock(), followingBlk)
		} else {
			state.unreachable = false
		}

		// Reset the stack so that we can correctly handle the else block.
		state.values = state.values[:ifctrl.originalStackLenWithoutParam]
		elseBlk := ifctrl.blk
		for _, arg := range ifctrl.clonedArgs {
			state.push(arg)
		}

		builder.SetCurrentBlock(elseBlk)

	case wasm.OpcodeEnd:
		ctrl := state.ctrlPop()
		followingBlk := ctrl.followingBlock

		if !state.unreachable {
			// Top n-th args will be used as a result of the current control frame.
			args := c.loweringState.nPeekDup(len(ctrl.blockType.Results))

			// Insert the unconditional branch to the target.
			c.insertJumpToBlock(args, builder.CurrentBlock(), followingBlk)
		} else { // unreachable.
			if state.unreachableDepth > 0 {
				state.unreachableDepth--
				return
			} else {
				state.unreachable = false
			}
		}

		switch ctrl.kind {
		case controlFrameKindFunction:
			return // This is the very end of function.
		case controlFrameKindLoop:
			// Loop header block can be reached from any br/br_table contained in the loop,
			// so now that we've reached End of it, we can seal it.
			builder.Seal(ctrl.blk)
		case controlFrameKindIfWithoutElse:
			// If this is the end of Then block, we have to emit the empty Else block.
			elseBlk := ctrl.blk
			builder.SetCurrentBlock(elseBlk)
			c.insertJumpToBlock(nil, elseBlk, followingBlk)
		}

		builder.Seal(ctrl.followingBlock)

		// Ready to start translating the following block.
		c.switchTo(ctrl.originalStackLenWithoutParam, followingBlk)

	case wasm.OpcodeBr:
		labelIndex := c.readI32u()
		if state.unreachable {
			return
		}

		targetFrame := state.ctrlPeekAt(int(labelIndex))
		var targetBlk ssa.BasicBlock
		var argNum int
		if targetFrame.isLoop() {
			targetBlk, argNum = targetFrame.blk, len(targetFrame.blockType.Params)
		} else {
			targetBlk, argNum = targetFrame.followingBlock, len(targetFrame.blockType.Results)
		}
		args := c.loweringState.nPeekDup(argNum)
		c.insertJumpToBlock(args, builder.CurrentBlock(), targetBlk)

		state.unreachable = true

	case wasm.OpcodeBrIf:
		labelIndex := c.readI32u()
		if state.unreachable {
			return
		}

		currentBlk := builder.CurrentBlock()
		v := state.pop()

		targetFrame := state.ctrlPeekAt(int(labelIndex))
		var targetBlk ssa.BasicBlock
		var argNum int
		if targetFrame.isLoop() {
			targetBlk, argNum = targetFrame.blk, len(targetFrame.blockType.Params)
		} else {
			targetBlk, argNum = targetFrame.followingBlock, len(targetFrame.blockType.Results)
		}
		args := c.loweringState.nPeekDup(argNum)

		// Insert the conditional jump to the target block.
		brnz := builder.AllocateInstruction()
		brnz.AsBrz(v, args, targetBlk)
		builder.InsertInstruction(brnz)
		targetBlk.AddPred(currentBlk, brnz)

		// Insert the unconditional jump to the Else block which corresponds to after br_if.
		elseBlk := builder.AllocateBasicBlock()
		c.insertJumpToBlock(args, currentBlk, elseBlk)

		// Now start translating the instructions after br_if.
		builder.SetCurrentBlock(elseBlk)

	case wasm.OpcodeNop:
	case wasm.OpcodeReturn:
		results := c.loweringState.nPeekDup(c.results())
		instr := builder.AllocateInstruction()

		instr.AsReturn(results)
		builder.InsertInstruction(instr)
		state.unreachable = true

	case wasm.OpcodeUnreachable:
		trapBlk := c.getOrCreateTrapBlock(wazevoapi.TrapCodeUnreachable)
		c.insertJumpToBlock(nil, builder.CurrentBlock(), trapBlk)
		state.unreachable = true

	case wasm.OpcodeCall:
		fnIndex := c.readI32u()
		if state.unreachable {
			return
		}

		// Before transfer the control to the callee, we have to store the current module's moduleContextPtr
		// into execContext.callerModuleContextPtr in case when the callee is a Go function.
		//
		// TODO: maybe this can be optimized out if this is in-module function calls. Investigate later.
		c.storeCallerModuleContext()

		typIndex := c.m.FunctionSection[fnIndex]
		typ := &c.m.TypeSection[typIndex]

		// TODO: reuse slice?
		argN := len(typ.Params)
		args := make([]ssa.Value, argN+2)
		args[0] = c.execCtxPtrValue
		args[1] = c.moduleCtxPtrValue
		copy(args[2:], state.nPeek(argN))

		sig := c.signatures[typ]
		if fnIndex >= c.m.ImportFunctionCount {
			call := builder.AllocateInstruction()
			call.AsCall(ssa.FuncRef(fnIndex), sig, args)
			builder.InsertInstruction(call)

			first, rest := call.Returns()
			state.push(first)
			for _, v := range rest {
				state.push(v)
			}
		} else {
			panic("TODO: support calling imported functions")
		}
	case wasm.OpcodeDrop:
		_ = state.pop()
	default:
		panic("TODO: unsupported in wazevo yet: " + wasm.InstructionName(op))
	}
}

func (c *Compiler) storeCallerModuleContext() {
	builder := c.ssaBuilder
	execCtx := c.execCtxPtrValue
	store := builder.AllocateInstruction()
	store.AsStore(c.moduleCtxPtrValue, execCtx, c.offsets.ExecutionContextCallerModuleContextPtr.U32())
	builder.InsertInstruction(store)
}

func (c *Compiler) readI32u() uint32 {
	v, n, err := leb128.LoadUint32(c.wasmFunctionBody[c.loweringState.pc+1:])
	if err != nil {
		panic(err) // shouldn't be reached since compilation comes after validation.
	}
	c.loweringState.pc += int(n)
	return v
}

func (c *Compiler) readI32s() int32 {
	v, n, err := leb128.LoadInt32(c.wasmFunctionBody[c.loweringState.pc+1:])
	if err != nil {
		panic(err) // shouldn't be reached since compilation comes after validation.
	}
	c.loweringState.pc += int(n)
	return v
}

func (c *Compiler) readBlockType() *wasm.FunctionType {
	state := &c.loweringState

	c.br.Reset(c.wasmFunctionBody[state.pc+1:])
	bt, num, err := wasm.DecodeBlockType(c.m.TypeSection, c.br, api.CoreFeaturesV2)
	if err != nil {
		panic(err) // shouldn't be reached since compilation comes after validation.
	}
	state.pc += int(num)

	return bt
}

func (c *Compiler) insertJumpToBlock(args []ssa.Value, currentBlk, targetBlk ssa.BasicBlock) {
	builder := c.ssaBuilder
	jmp := builder.AllocateInstruction()
	jmp.AsJump(args, targetBlk)
	builder.InsertInstruction(jmp)
	targetBlk.AddPred(currentBlk, jmp)
}

func (c *Compiler) switchTo(originalStackLen int, targetBlk ssa.BasicBlock) {
	// Now we should adjust the stack and start translating the continuation block.
	c.loweringState.values = c.loweringState.values[:originalStackLen]

	c.ssaBuilder.SetCurrentBlock(targetBlk)

	// At this point, blocks params consist only of the Wasm-level parameters,
	// (since it's added only when we are trying to resolve variable *inside* this block).
	for i := 0; i < targetBlk.Params(); i++ {
		_, value := targetBlk.Param(i)
		c.loweringState.push(value)
	}
}

func cloneValuesList(in []ssa.Value) (ret []ssa.Value) {
	ret = make([]ssa.Value, len(in))
	for i := range ret {
		ret[i] = in[i]
	}
	return
}

func (c *Compiler) results() int {
	return len(c.wasmFunctionTyp.Results)
}
