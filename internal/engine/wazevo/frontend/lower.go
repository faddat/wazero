package frontend

import (
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
		// loopBlock is not nil if this is the loop.
		loopBlock,
		// followingBlock is the basic block we enter if we reach "end" of block.
		followingBlock ssa.BasicBlock
		blockType *wasm.FunctionType
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

func (l *loweringState) nPeek(n int) []ssa.Value {
	tail := len(l.values)
	return l.values[tail-n : tail]
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

func (c *Compiler) lowerOpcode(op wasm.Opcode) {
	builder := c.ssaBuilder
	state := &c.loweringState
	switch op {
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
			return
		}

		ctrl := state.ctrlPop()
		if ctrl.isReturn() {
			c.insertReturn()
			return
		}

		// Top n-th args will be used as a result of the current control frame.
		args := c.loweringState.nPeek(len(ctrl.blockType.Results))

		currentBlk := builder.CurrentBlock()
		targetBlk := ctrl.followingBlock
		// Record that the target has the current one as a predecessor.
		targetBlk.AddPred(currentBlk)

		// Insert the unconditional branch to the target.
		jmp := builder.AllocateInstruction()
		jmp.AsJump(cloneValuesList(args), targetBlk)
		for i := 0; i < targetBlk.Params(); i++ {
			variable, _ := targetBlk.Param(i)
			builder.DefineVariable(variable, args[i], currentBlk)
		}

		c.switchTo(ctrl.originalStackLenWithoutParam, targetBlk)
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
	results := c.loweringState.nPeek(c.results())
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
