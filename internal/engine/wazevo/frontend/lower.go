package frontend

import (
	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
	"github.com/tetratelabs/wazero/internal/wasm"
)

type (
	loweringState struct {
		values        []ssa.Value
		controlFrames []controlFrame
		unreachable   bool
		pc            int
	}
	controlFrame struct {
		// loopBlock is not nil if this is the loop.
		loopBlock,
		// followingBlock is the basic block we enter if we reach "end" of block.
		followingBlock ssa.BasicBlock
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
	tail := len(l.values) - 1
	ret = l.controlFrames[tail]
	l.values = l.values[:tail]
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
	switch op {
	case wasm.OpcodeReturn:
		c.insertReturn()
	case wasm.OpcodeEnd:
		ctrl := c.loweringState.ctrlPop()
		if ctrl.isReturn() {
			c.insertReturn()
		}
	}
}

func (c *Compiler) insertReturn() {
	results := c.loweringState.nPeek(c.results())
	instr := c.ssaBuilder.AllocateInstruction()

	// Results is the view over c.loweringState.values, so we need to copy it.
	// TODO: reuse the slice.
	vs := make([]ssa.Value, len(results))
	for i := range vs {
		vs[i] = results[i]
	}
	instr.AsReturn(vs)
	c.ssaBuilder.InsertInstruction(instr)
}

func (c *Compiler) results() int {
	return len(c.wasmFunctionTyp.Results)
}
