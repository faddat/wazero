package ssa

import (
	"testing"

	"github.com/tetratelabs/wazero/internal/testing/require"
)

func Test_maybeInvertBranch(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(b *builder) (now, next *basicBlock, verify func(t *testing.T))
		exp   bool
	}{
		{
			name: "ends with br_table",
			setup: func(b *builder) (now, next *basicBlock, verify func(t *testing.T)) {
				now, next = b.allocateBasicBlock(), b.allocateBasicBlock()
				inst := b.AllocateInstruction()
				// TODO: we haven't implemented AsBrTable on Instruction.
				inst.opcode = OpcodeBrTable
				now.currentInstr = inst
				verify = func(t *testing.T) { require.Equal(t, OpcodeBrTable, inst.opcode) }
				return
			},
		},
		{
			name: "no conditional branch without previous instruction",
			setup: func(b *builder) (now, next *basicBlock, verify func(t *testing.T)) {
				now, next = b.allocateBasicBlock(), b.allocateBasicBlock()
				tail := b.AllocateInstruction()
				tail.AsJump(nil, next)
				now.currentInstr = tail
				verify = func(t *testing.T) { require.Equal(t, OpcodeJump, tail.opcode) }
				return
			},
		},
		{
			name: "no conditional branch with previous instruction",
			setup: func(b *builder) (now, next *basicBlock, verify func(t *testing.T)) {
				now, next = b.allocateBasicBlock(), b.allocateBasicBlock()
				tail := b.AllocateInstruction()
				tail.AsJump(nil, next)
				now.currentInstr = tail
				prev := b.AllocateInstruction()
				prev.AsIconst64(1)
				prev.next = tail
				now.rootInstr = prev
				tail.prev = prev
				verify = func(t *testing.T) {
					require.Equal(t, OpcodeJump, tail.opcode)
					require.Equal(t, prev, tail.prev)
				}
				return
			},
		},
		{
			name: "tail target is already loop",
			setup: func(b *builder) (now, next *basicBlock, verify func(t *testing.T)) {
				now, next, loopHeader := b.allocateBasicBlock(), b.allocateBasicBlock(), b.allocateBasicBlock()
				loopHeader.loopHeader = true

				tail := b.AllocateInstruction()
				tail.AsJump(nil, loopHeader) // jump to loop, which doesn't need inversion.
				now.currentInstr = tail
				conditionalBr := b.AllocateInstruction()
				conditionalBr.AsBrz(0, nil, nil)
				conditionalBr.next = tail
				now.rootInstr = conditionalBr
				tail.prev = conditionalBr
				verify = func(t *testing.T) {
					require.Equal(t, OpcodeJump, tail.opcode)
					require.Equal(t, OpcodeBrz, conditionalBr.opcode) // intact.
					require.Equal(t, conditionalBr, tail.prev)
				}
				return
			},
		},
		{
			name: "tail target is already the next block",
			setup: func(b *builder) (now, next *basicBlock, verify func(t *testing.T)) {
				now, next = b.allocateBasicBlock(), b.allocateBasicBlock()
				tail := b.AllocateInstruction()
				tail.AsJump(nil, next) // jump to next block, which doesn't need inversion.
				now.currentInstr = tail
				conditionalBr := b.AllocateInstruction()
				conditionalBr.AsBrz(0, nil, nil)
				conditionalBr.next = tail
				now.rootInstr = conditionalBr
				tail.prev = conditionalBr
				verify = func(t *testing.T) {
					require.Equal(t, OpcodeJump, tail.opcode)
					require.Equal(t, OpcodeBrz, conditionalBr.opcode) // intact.
					require.Equal(t, conditionalBr, tail.prev)
				}
				return
			},
		},
		{
			name: "tail target is already the next block",
			setup: func(b *builder) (now, next *basicBlock, verify func(t *testing.T)) {
				now, next = b.allocateBasicBlock(), b.allocateBasicBlock()
				tail := b.AllocateInstruction()
				tail.AsJump(nil, next) // jump to next block, which doesn't need inversion.
				now.currentInstr = tail
				conditionalBr := b.AllocateInstruction()
				conditionalBr.AsBrz(0, nil, nil)
				conditionalBr.next = tail
				now.rootInstr = conditionalBr
				tail.prev = conditionalBr
				verify = func(t *testing.T) {
					require.Equal(t, OpcodeJump, tail.opcode)
					require.Equal(t, OpcodeBrz, conditionalBr.opcode) // intact.
					require.Equal(t, conditionalBr, tail.prev)
				}
				return
			},
		},
		{
			name: "conditional target is loop",
			setup: func(b *builder) (now, next *basicBlock, verify func(t *testing.T)) {
				now, next, loopHeader, nowNext := b.allocateBasicBlock(), b.allocateBasicBlock(), b.allocateBasicBlock(), b.allocateBasicBlock()
				loopHeader.loopHeader = true

				tail := b.AllocateInstruction()
				tail.AsJump(nil, nowNext)
				now.currentInstr = tail
				conditionalBr := b.AllocateInstruction()
				conditionalBr.AsBrz(0, nil, loopHeader) // jump to loop, which needs inversion.
				conditionalBr.next = tail
				now.rootInstr = conditionalBr
				tail.prev = conditionalBr
				verify = func(t *testing.T) {
					require.Equal(t, OpcodeJump, tail.opcode)
					require.Equal(t, OpcodeBrnz, conditionalBr.opcode) // inversion.
					require.Equal(t, loopHeader, tail.blk)             // swapped.
					require.Equal(t, nowNext, conditionalBr.blk)       // swapped.
					require.Equal(t, conditionalBr, tail.prev)
				}
				return
			},
			exp: true,
		},
		{
			name: "conditional target is the next block",
			setup: func(b *builder) (now, next *basicBlock, verify func(t *testing.T)) {
				now, next = b.allocateBasicBlock(), b.allocateBasicBlock()
				nowTarget := b.allocateBasicBlock()

				tail := b.AllocateInstruction()
				tail.AsJump(nil, nowTarget)
				now.currentInstr = tail
				conditionalBr := b.AllocateInstruction()
				conditionalBr.AsBrz(0, nil, next) // jump to loop, which needs inversion.
				conditionalBr.next = tail
				now.rootInstr = conditionalBr
				tail.prev = conditionalBr
				verify = func(t *testing.T) {
					require.Equal(t, OpcodeJump, tail.opcode)
					require.Equal(t, OpcodeBrnz, conditionalBr.opcode) // inversion.
					require.Equal(t, next, tail.blk)                   // swapped.
					require.Equal(t, nowTarget, conditionalBr.blk)     // swapped.
					require.Equal(t, conditionalBr, tail.prev)
				}
				return
			},
			exp: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder().(*builder)
			now, next, verify := tc.setup(b)
			actual := maybeInvertBranch(now, next)
			verify(t)
			require.Equal(t, tc.exp, actual)
		})
	}
}

func TestBuilder_splitCriticalEdge(t *testing.T) {
	b := NewBuilder().(*builder)
	predBlk, dummyBlk := b.allocateBasicBlock(), b.allocateBasicBlock()
	predBlk.reversePostOrder = 100
	b.SetCurrentBlock(predBlk)
	inst := b.AllocateInstruction()
	inst.AsIconst32(1)
	b.InsertInstruction(inst)
	v, _ := inst.Returns()
	originalBrz := b.AllocateInstruction() // This is the split edge.
	originalBrz.AsBrz(v, nil, dummyBlk)
	b.InsertInstruction(originalBrz)
	dummyJump := b.AllocateInstruction()
	dummyJump.AsJump(nil, dummyBlk)
	b.InsertInstruction(dummyJump)

	predInfo := &basicBlockPredecessorInfo{blk: predBlk, branch: originalBrz}
	trampoline := b.splitCriticalEdge(predBlk, predInfo)
	require.NotNil(t, trampoline)
	require.Equal(t, 100, trampoline.reversePostOrder)

	require.Equal(t, trampoline, predInfo.blk)
	require.Equal(t, originalBrz, predInfo.branch)
	require.Equal(t, trampoline.rootInstr, predInfo.branch)
	require.Equal(t, trampoline.currentInstr, predInfo.branch)

	replacedBrz := predBlk.rootInstr.next
	require.Equal(t, OpcodeBrz, replacedBrz.opcode)
	require.Equal(t, trampoline, replacedBrz.blk)
}

func Test_swapInstruction(t *testing.T) {
	t.Run("swap root", func(t *testing.T) {
		b := NewBuilder().(*builder)
		blk := b.allocateBasicBlock()

		dummy := b.AllocateInstruction()

		old := b.AllocateInstruction()
		old.next, dummy.prev = dummy, old
		newi := b.AllocateInstruction()
		blk.rootInstr = old
		swapInstruction(blk, old, newi)

		require.Equal(t, newi, blk.rootInstr)
		require.Equal(t, dummy, newi.next)
		require.Equal(t, dummy.prev, newi)
		require.Nil(t, old.next)
		require.Nil(t, old.prev)
	})
	t.Run("swap middle", func(t *testing.T) {
		b := NewBuilder().(*builder)
		blk := b.allocateBasicBlock()
		b.SetCurrentBlock(blk)
		i1, i2, i3 := b.AllocateInstruction(), b.AllocateInstruction(), b.AllocateInstruction()
		i1.AsIconst32(1)
		i2.AsIconst32(2)
		i3.AsIconst32(3)
		b.InsertInstruction(i1)
		b.InsertInstruction(i2)
		b.InsertInstruction(i3)

		newi := b.AllocateInstruction()
		newi.AsIconst32(100)
		swapInstruction(blk, i2, newi)

		require.Equal(t, i1, blk.rootInstr)
		require.Equal(t, newi, i1.next)
		require.Equal(t, i3, newi.next)
		require.Equal(t, i1, newi.prev)
		require.Equal(t, newi, i3.prev)
		require.Nil(t, i2.next)
		require.Nil(t, i2.prev)
	})
	t.Run("swap tail", func(t *testing.T) {
		b := NewBuilder().(*builder)
		blk := b.allocateBasicBlock()
		b.SetCurrentBlock(blk)
		i1, i2 := b.AllocateInstruction(), b.AllocateInstruction()
		i1.AsIconst32(1)
		i2.AsIconst32(2)
		b.InsertInstruction(i1)
		b.InsertInstruction(i2)

		newi := b.AllocateInstruction()
		newi.AsIconst32(100)
		swapInstruction(blk, i2, newi)

		require.Equal(t, i1, blk.rootInstr)
		require.Equal(t, newi, i1.next)
		require.Equal(t, i1, newi.prev)
		require.Nil(t, newi.next)
		require.Nil(t, i2.next)
		require.Nil(t, i2.prev)
	})
}
