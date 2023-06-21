package ssa

import (
	"fmt"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"testing"
)

func Test_passBlockFrequency(t *testing.T) {
	insertJump := func(b *builder, from *basicBlock, to *basicBlock) {
		jmp := b.AllocateInstruction()
		jmp.opcode = OpcodeJump
		jmp.blk = to
		from.currentInstr = jmp
	}

	for _, tc := range []struct {
		name  string
		edges edgesCase
		setup func(b *builder)
		exp   map[basicBlockID]int64
	}{
		{
			name: "linear",
			// 0 -> 1 -> 2 -> 3 -> 4
			edges: edgesCase{
				0: {1},
				1: {2},
				2: {3},
				3: {4},
			},
			setup: func(b *builder) {},
			exp:   map[basicBlockID]int64{0: 1, 1: 1, 2: 1, 3: 1, 4: 1},
		},
		{
			name: "diamond - blk1 as fallthrough",
			//  0
			// / \
			// 1   2
			// \ /
			//  3
			edges: edgesCase{
				0: {1, 2},
				1: {3},
				2: {3},
			},
			setup: func(b *builder) {
				b0, b1, b2, b3 :=
					b.basicBlocksPool.View(0), b.basicBlocksPool.View(1),
					b.basicBlocksPool.View(2), b.basicBlocksPool.View(3)
				insertJump(b, b0, b1) // b1 as a fallthrough edge.
				insertJump(b, b1, b3)
				insertJump(b, b2, b3)
			},
			exp: map[basicBlockID]int64{
				0: 1,
				1: 2, // fallthrough edge is be prioritized.
				2: 1,
				3: 3,
			},
		},
		{
			name: "diamond - blk2 as fallthrough",
			//  0
			// / \
			// 1   2
			// \ /
			//  3
			edges: edgesCase{
				0: {1, 2},
				1: {3},
				2: {3},
			},
			setup: func(b *builder) {
				b0, b1, b2, b3 :=
					b.basicBlocksPool.View(0), b.basicBlocksPool.View(1),
					b.basicBlocksPool.View(2), b.basicBlocksPool.View(3)
				insertJump(b, b0, b2) // b2 as a fallthrough edge.
				insertJump(b, b1, b3)
				insertJump(b, b2, b3)
			},
			exp: map[basicBlockID]int64{
				0: 1,
				1: 1,
				2: 2, // fallthrough edge is be prioritized.
				3: 3,
			},
		},
		{
			name: "loop",
			//                0
			//               / \
			//         ---> 1   4
			//         |    |   |
			//         3 -- 2   |
			//         |        |
			//          \       /
			//           \    /
			//             v v
			//              5
			edges: edgesCase{
				0: {1, 4},
				1: {2},
				2: {3},
				3: {1, 5},
				4: {5},
			},
			setup: func(b *builder) {
				b0, _, _, b3, b4, b5 :=
					b.basicBlocksPool.View(0), b.basicBlocksPool.View(1),
					b.basicBlocksPool.View(2), b.basicBlocksPool.View(3),
					b.basicBlocksPool.View(4), b.basicBlocksPool.View(5)
				insertJump(b, b0, b4) // b4 as a fallthrough edge.
				insertJump(b, b3, b5) // b5 as a fallthrough edge.
			},
		},
	} {

		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			b := constructGraphFromEdges(tc.edges)
			tc.setup(b)
			// Dominance calculation is necessary for block frequency calculation.
			passCalculateImmediateDominators(b)

			// Run the calculation.
			passBlockFrequency(b)

			fmt.Println(b.edgeWeights)
			fmt.Println(b.blockFrequencies)

			for blk := b.blockIteratorBegin(); blk != nil; blk = b.blockIteratorNext() {
				actual := b.blockFrequencies[blk.id]
				require.Equal(t, tc.exp[blk.id], actual)
			}
		})
	}
}
