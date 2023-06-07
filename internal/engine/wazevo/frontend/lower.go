package frontend

import (
	"fmt"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
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
		// blk is the loop body if this is loop, and is the else-block if this is a if frame.
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
	controlFrameKindIf
	controlFrameKindBlock
)

func (ctrl *controlFrame) isReturn() bool {
	return ctrl.kind == controlFrameKindFunction
}

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

func (l *loweringState) peek() ssa.Value {
	i := len(l.values) - 1
	return l.values[i]
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

func (l *loweringState) ctrlPeekAt(n int) (ret controlFrame) {
	tail := len(l.controlFrames) - 1
	return l.controlFrames[tail-n]
}

func (c *Compiler) lowerBody(_entryBlock ssa.BasicBlock) {
	// Pushes the empty control frame which corresponds to the function return.
	c.loweringState.ctrlPush(controlFrame{kind: controlFrameKindFunction})

	for c.loweringState.pc < len(c.wasmFunctionBody) {
		op := c.wasmFunctionBody[c.loweringState.pc]
		c.lowerOpcode(op)
		fmt.Println("---------" + wasm.InstructionName(op) + " --------")
		fmt.Println(c.ssaBuilder.String())
		c.loweringState.pc++
	}
}

func (c *Compiler) lowerOpcode(op wasm.Opcode) {
	builder := c.ssaBuilder
	state := &c.loweringState
	switch op {
	case wasm.OpcodeNop:
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

		loopBodyBlk, afterLoopBlock := builder.AllocateBasicBlock(), builder.AllocateBasicBlock()
		c.addBlockParamsFromWasmTypes(bt.Params, loopBodyBlk)
		c.addBlockParamsFromWasmTypes(bt.Results, afterLoopBlock)

		state.ctrlPush(controlFrame{
			originalStackLenWithoutParam: len(state.values) - len(bt.Params),
			kind:                         controlFrameKindLoop,
			blk:                          loopBodyBlk,
			followingBlock:               afterLoopBlock,
			blockType:                    bt,
		})

	case wasm.OpcodeIf:
		bt := c.readBlockType()

		if state.unreachable {
			state.unreachableDepth++
			return
		}

		v := state.pop()
		thenBlk, elseBlk, followingBlk :=
			builder.AllocateBasicBlock(), builder.AllocateBasicBlock(), builder.AllocateBasicBlock()

		// We do not make the Wasm-level block parameters as SSA-level block params,
		// since they won't be phi and the definition is unique.

		// On the other hand, the following block after if-else-end will likely have
		// multiple definitions (one in Then and another in Else blocks).
		c.addBlockParamsFromWasmTypes(bt.Results, followingBlk)

		var args []ssa.Value
		if len(bt.Params) > 0 {
			args = cloneValuesList(state.values[len(state.values)-1-len(bt.Params):])
		}

		// Insert the conditional jump to the else block.
		brz := builder.AllocateInstruction()
		brz.AsBrz(v, args, elseBlk)
		builder.InsertInstruction(brz)
		elseBlk.AddPred(builder.CurrentBlock())

		state.ctrlPush(controlFrame{
			kind:                         controlFrameKindIf,
			originalStackLenWithoutParam: len(state.values) - len(bt.Params),
			blk:                          elseBlk,
			followingBlock:               followingBlk,
			blockType:                    bt,
			clonedArgs:                   args,
		})

		c.ssaBuilder.SetCurrentBlock(thenBlk)

	case wasm.OpcodeElse:
		if unreachable := state.unreachable; unreachable && state.unreachableDepth > 0 {
			// If it is currently in unreachable and is a nested if,
			// we just remove the entire else block.
			return
		}

		ifctrl := state.ctrlPeekAt(0)
		if !state.unreachable {
			// If this Then block is currently reachable, we have to insert the branching to the following BB.
			followingBlk := ifctrl.followingBlock // == the BB after if-then-else.
			args := c.loweringState.nPeekDup(len(ifctrl.blockType.Results))
			c.jumpToBlock(args, builder.CurrentBlock(), followingBlk)
		} else {
			state.unreachable = false
		}

		// Reset the stack so that we can correctly handle the else block.
		state.values = state.values[:ifctrl.originalStackLenWithoutParam]
		elseBlk := ifctrl.blk
		for _, arg := range ifctrl.clonedArgs {
			state.push(arg)
		}

		c.ssaBuilder.SetCurrentBlock(elseBlk)
	case wasm.OpcodeReturn:
		c.insertReturn()
		state.unreachable = true

	case wasm.OpcodeEnd:
		if unreachable := state.unreachable; unreachable && state.unreachableDepth > 0 {
			state.unreachableDepth--
			return
		} else if unreachable {
			ctrl := state.ctrlPop()
			if ctrl.isReturn() {
				// This is the very end of function body.
				return
			}
			// We do not need branching here because this is unreachable.
			c.switchTo(ctrl.originalStackLenWithoutParam, ctrl.followingBlock)
			state.unreachable = false
			return
		}

		ctrl := state.ctrlPop()
		if ctrl.isReturn() {
			c.insertReturn()
			return
		}

		// Top n-th args will be used as a result of the current control frame.
		args := c.loweringState.nPeekDup(len(ctrl.blockType.Results))

		currentBlk := builder.CurrentBlock()
		followingBlk := ctrl.followingBlock

		// Insert the unconditional branch to the target.
		c.jumpToBlock(args, currentBlk, followingBlk)

		// Ready to start translating the following block.
		c.switchTo(ctrl.originalStackLenWithoutParam, followingBlk)

	case wasm.OpcodeBr:
		v := c.readU32()
		if state.unreachable {
			return
		}

		targetFrame := state.ctrlPeekAt(int(v))
		if targetFrame.isReturn() {
			c.insertReturn()
			state.unreachable = true
			return
		}

		var targetBlk ssa.BasicBlock
		var argNum int
		if targetFrame.isLoop() {
			targetBlk, argNum = targetFrame.blk, len(targetFrame.blockType.Params)
		} else {
			targetBlk, argNum = targetFrame.followingBlock, len(targetFrame.blockType.Results)
		}
		args := c.loweringState.nPeekDup(argNum)
		c.jumpToBlock(args, builder.CurrentBlock(), targetBlk)

		state.unreachable = true
	default:
		panic("TODO: unsupported in wazevo yet" + wasm.InstructionName(op))
	}
}

func (c *Compiler) readU32() uint32 {
	v, n, err := leb128.LoadUint32(c.wasmFunctionBody[c.loweringState.pc+1:])
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

func (c *Compiler) jumpToBlock(args []ssa.Value, currentBlk, targetBlk ssa.BasicBlock) {
	builder := c.ssaBuilder
	jmp := builder.AllocateInstruction()
	jmp.AsJump(args, targetBlk)
	for i := 0; i < targetBlk.Params(); i++ {
		variable, _ := targetBlk.Param(i)
		builder.DefineVariable(variable, args[i], currentBlk)
	}

	builder.InsertInstruction(jmp)
	targetBlk.AddPred(currentBlk)
}

func (c *Compiler) switchTo(originalStackLen int, targetBlk ssa.BasicBlock) {
	// Now we should adjust the stack and start translating the continuation block.
	c.loweringState.values = c.loweringState.values[:originalStackLen]

	c.ssaBuilder.SetCurrentBlock(targetBlk)
	for i := 0; i < targetBlk.Params(); i++ {
		_, value := targetBlk.Param(i)
		c.loweringState.push(value)
	}
}

func (c *Compiler) insertReturn() {
	results := c.loweringState.nPeekDup(c.results())
	instr := c.ssaBuilder.AllocateInstruction()

	// Results is the view over c.loweringState.values, so we need to copy it.
	// TODO: reuse the slice.
	vs := cloneValuesList(results)
	instr.AsReturn(vs)
	c.ssaBuilder.InsertInstruction(instr)
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
