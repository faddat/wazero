package frontend

import (
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
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
		// originalStackLen holds the number of values on the Wasm stack
		// when start executing this control frame minus params for the block.
		originalStackLenWithoutParam int
		// loopBodyBlock is not nil if this is the loop.
		loopBodyBlock,
		// followingBlock is the basic block we enter if we reach "end" of block.
		followingBlock ssa.BasicBlock
		blockType *wasm.FunctionType
		// clonedArgs hold the arguments to Else block.
		clonedArgs []ssa.Value
	}
)

func (ctrl *controlFrame) isReturn() bool {
	return ctrl.followingBlock == nil
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
	return
}

func (l *loweringState) peek() ssa.Value {
	i := len(l.values) - 1
	return l.values[i]
}

func (l *loweringState) nPeekDup(n int) []ssa.Value {
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
	return
}

func (l *loweringState) ctrlPeekAt(n int) (ret controlFrame) {
	tail := len(l.controlFrames) - 1
	return l.controlFrames[tail-n]
}

func (c *Compiler) lowerBody(_entryBlock ssa.BasicBlock) {
	// Pushes the empty control frame which corresponds to the function return.
	c.loweringState.ctrlPush(controlFrame{})

	for c.loweringState.pc < len(c.wasmFunctionBody) {
		op := c.wasmFunctionBody[c.loweringState.pc]
		c.lowerOpcode(op)
		c.loweringState.pc++
	}
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
			loopBodyBlock:                loopBodyBlk,
			followingBlock:               afterLoopBlock,
			blockType:                    bt,
		})

	case wasm.OpcodeIf:
		bt := c.readBlockType()

		if state.unreachable {
			state.unreachableDepth++
			return
		}

		thenBlk, followingBlk := builder.AllocateBasicBlock(), builder.AllocateBasicBlock()

		// We do not make the Wasm-level block parameters as SSA-level block params,
		// since they won't be phi and the definition is unique.

		// On the other hand, the following block after if-else-end will likely have
		// multiple definitions (one in Then and another in Else blocks).
		c.addBlockParamsFromWasmTypes(bt.Results, followingBlk)

		args := cloneValuesList(state.values[len(state.values)-1-len(bt.Params):])
		state.ctrlPush(controlFrame{
			originalStackLenWithoutParam: len(state.values) - len(bt.Params),
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
		elseBlk := builder.AllocateBasicBlock()
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
		// Record that the target has the current one as a predecessor.
		followingBlk.AddPred(currentBlk)

		// Insert the unconditional branch to the target.
		c.jumpToBlock(args, currentBlk, followingBlk)

		// Ready to start translating the following block.
		c.switchTo(ctrl.originalStackLenWithoutParam, followingBlk)
	default:
		panic("TODO: unsupported in wazevo yet" + wasm.InstructionName(op))
	}
}

func (c *Compiler) jumpToBlock(args []ssa.Value, currentBlk, targetBlk ssa.BasicBlock) {
	builder := c.ssaBuilder
	jmp := builder.AllocateInstruction()
	jmp.AsJump(cloneValuesList(args), targetBlk)
	for i := 0; i < targetBlk.Params(); i++ {
		variable, _ := targetBlk.Param(i)
		builder.DefineVariable(variable, args[i], currentBlk)
	}
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
