package ssa

import (
	"fmt"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"testing"
)

func TestBuilder_Optimize(t *testing.T) {

	for _, tc := range []struct {
		name string
		// pass is the optimization pass to run.
		pass optimizationPass
		// setup creates the SSA function in the given *builder.
		// TODO: when we have the text SSA IR parser, we can eliminate this `setup`,
		// 	we could directly decode the *builder from the `before` string. I am still
		//  constantly changing the format, so let's keep setup for now.
		setup func(*builder)
		// before is the expected SSA function after `setup` is executed.
		before,
		// after is the expected output after optimization pass.
		after string
	}{
		{
			name: "dead code",
			pass: passDeadCodeElimination,
			setup: func(b *builder) {
				entry := b.AllocateBasicBlock()
				_, value := entry.AddParam(b, TypeI32)

				middle1, middle2 := b.AllocateBasicBlock(), b.AllocateBasicBlock()
				end := b.AllocateBasicBlock()

				b.SetCurrentBlock(entry)
				{
					brz := b.AllocateInstruction()
					brz.AsBrz(value, nil, middle1)
					b.InsertInstruction(brz)

					jmp := b.AllocateInstruction()
					jmp.AsJump(nil, middle2)
					b.InsertInstruction(jmp)
				}

				b.SetCurrentBlock(middle1)
				{
					jmp := b.AllocateInstruction()
					jmp.AsJump(nil, end)
					b.InsertInstruction(jmp)
				}

				b.SetCurrentBlock(middle2)
				{
					jmp := b.AllocateInstruction()
					jmp.AsJump(nil, end)
					b.InsertInstruction(jmp)
				}

				{
					unreachable := b.AllocateBasicBlock()
					b.SetCurrentBlock(unreachable)
					jmp := b.AllocateInstruction()
					jmp.AsJump(nil, end)
					b.InsertInstruction(jmp)
				}

				b.SetCurrentBlock(end)
				{
					jmp := b.AllocateInstruction()
					jmp.AsJump(nil, middle1)
					b.InsertInstruction(jmp)
				}
			},
			before: `
blk0: (v0:i32)
	Brz v0, blk1
	Jump blk2

blk1: () <-- (blk0,blk3)
	Jump blk3

blk2: () <-- (blk0)
	Jump blk3

blk3: () <-- (blk1,blk2,blk4)
	Jump blk1

blk4: ()
	Jump blk3
`,
			after: `
blk0: (v0:i32)
	Brz v0, blk1
	Jump blk2

blk1: () <-- (blk0,blk3)
	Jump blk3

blk2: () <-- (blk0)
	Jump blk3

blk3: () <-- (blk1,blk2)
	Jump blk1
`,
		},
		{
			name: "redundant phis",
			pass: passRedundantPhiElimination,
			setup: func(b *builder) {

				entry, loopHeader, end := b.AllocateBasicBlock(), b.AllocateBasicBlock(), b.AllocateBasicBlock()

				loopHeader.AddParam(b, TypeI32)
				var var1 = b.DeclareVariable(TypeI32)

				b.SetCurrentBlock(entry)
				{
					constInst := b.AllocateInstruction()
					constInst.AsIconst32(0xff)
					b.InsertInstruction(constInst)
					iConst, _ := constInst.Returns()
					b.DefineVariable(var1, iConst, entry)

					jmp := b.AllocateInstruction()
					jmp.AsJump([]Value{iConst}, loopHeader)
					b.InsertInstruction(jmp)
				}
				b.Seal(entry)

				b.SetCurrentBlock(loopHeader)
				{
					// At this point, loop is not sealed, so PHI will be added to this header. However, the only
					// input to the PHI is iConst above, so there must be an alias to iConst from the PHI value.
					value := b.FindValue(var1)

					tmpInst := b.AllocateInstruction()
					tmpInst.AsIconst32(0xff)
					b.InsertInstruction(tmpInst)
					tmp, _ := tmpInst.Returns()

					brz := b.AllocateInstruction()
					brz.AsBrz(value, []Value{tmp}, loopHeader) // Loop to itself.
					b.InsertInstruction(brz)

					jmp := b.AllocateInstruction()
					jmp.AsJump(nil, end)
					b.InsertInstruction(jmp)
				}
				b.Seal(loopHeader)

				b.SetCurrentBlock(end)
				{
					ret := b.AllocateInstruction()
					ret.AsReturn(nil)
					b.InsertInstruction(ret)
				}
			},
			before: `
blk0: ()
	v1:i32 = Iconst_32 0xff
	Jump blk1, v1, v1

blk1: (v0:i32,v2:i32) <-- (blk0,blk1)
	v3:i32 = Iconst_32 0xff
	Brz v2, blk1, v3, v2
	Jump blk2

blk2: () <-- (blk1)
	Return
`,
			after: `
blk0: ()
	v1:i32 = Iconst_32 0xff
	Jump blk1, v1

blk1: (v0:i32) <-- (blk0,blk1)
	v2 = v1
	v3:i32 = Iconst_32 0xff
	Brz v2, blk1, v3
	Jump blk2

blk2: () <-- (blk1)
	Return
`,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder().(*builder)
			tc.setup(b)
			fmt.Println(b.Format())
			require.Equal(t, tc.before, b.Format())
			tc.pass(b)
			fmt.Println("--------")
			fmt.Println(b.Format())
			require.Equal(t, tc.after, b.Format())
		})
	}
}
