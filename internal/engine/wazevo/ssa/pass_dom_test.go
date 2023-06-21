package ssa

import (
	"github.com/tetratelabs/wazero/internal/testing/require"
	"sort"
	"testing"
)

func TestBuilder_passCalculateDominatorTree(t *testing.T) {
	const numBlocks = 10

	for _, tc := range []struct {
		name    string
		edges   map[basicBlockID][]basicBlockID
		expDoms map[basicBlockID]basicBlockID
	}{
		{
			name: "linear",
			// 0 -> 1 -> 2 -> 3 -> 4
			edges: map[basicBlockID][]basicBlockID{
				0: {1},
				1: {2},
				2: {3},
				3: {4},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 1,
				3: 2,
				4: 3,
			},
		},
		{
			name: "diamond",
			//  0
			// / \
			// 1   2
			// \ /
			//  3
			edges: map[basicBlockID][]basicBlockID{
				0: {1, 2},
				1: {3},
				2: {3},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 0,
				3: 0,
			},
		},
		{
			name: "merge",
			// 0 -> 1 -> 3
			// |         ^
			// v         |
			// 2 ---------
			edges: map[basicBlockID][]basicBlockID{
				0: {1, 2},
				1: {3},
				2: {3},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 0,
				3: 0,
			},
		},
		{
			name: "branch",
			//  0
			// / \
			// 1   2
			edges: map[basicBlockID][]basicBlockID{
				0: {1, 2},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 0,
			},
		},
		{
			name: "loop",
			// 0 -> 1 -> 2
			// ^         |
			// |         v
			// 3 <-------
			edges: map[basicBlockID][]basicBlockID{
				0: {1},
				1: {2},
				2: {3},
				3: {0},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 1,
				3: 2,
			},
		},
		{
			name: "larger diamond",
			//     0
			//   / | \
			//  1  2  3
			//   \ | /
			//     4
			edges: map[basicBlockID][]basicBlockID{
				0: {1, 2, 3},
				1: {4},
				2: {4},
				3: {4},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 0,
				3: 0,
				4: 0,
			},
		},
		{
			name: "two independent branches",
			//  0
			// / \
			// 1   2
			// |   |
			// 3   4
			edges: map[basicBlockID][]basicBlockID{
				0: {1, 2},
				1: {3},
				2: {4},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 0,
				3: 1,
				4: 2,
			},
		},
		{
			name: "loop with branch",
			// 0 -> 1 -> 2
			//     |    |
			//     v    v
			//     3 <- 4
			edges: map[basicBlockID][]basicBlockID{
				0: {1},
				1: {2, 3},
				2: {4},
				4: {3},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 1,
				3: 1,
				4: 2,
			},
		},
		{
			name: "branches with merge",
			//  0
			// / \
			// 1   2
			// \   /
			//  3-4
			edges: map[basicBlockID][]basicBlockID{
				0: {1, 2},
				1: {3},
				2: {4},
				3: {4},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 0,
				3: 1,
				4: 0,
			},
		},
		{
			name: "complex",
			//   0
			//  / \
			// 1   2
			// |\ /|
			// | X |
			// |/ \|
			// 3   4
			edges: map[basicBlockID][]basicBlockID{
				0: {1, 2},
				1: {3, 4},
				2: {3, 4},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 0,
				3: 0,
				4: 0,
			},
		},
		{
			name: "nested loops",
			//     0
			//    / \
			//   v   v
			//   1 -> 2
			//   ^    |
			//   |    v
			//   4 <- 3
			edges: map[basicBlockID][]basicBlockID{
				0: {1, 2},
				1: {2},
				2: {3, 1},
				3: {4},
				4: {1},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 0,
				3: 2,
				4: 3,
			},
		},
		{
			name: "two intersecting loops",
			//   0
			//   v
			//   1 --> 2 --> 3
			//   ^     |     |
			//   |     v     v
			//   4 <-- 5 <-- 6
			edges: map[basicBlockID][]basicBlockID{
				0: {1},
				1: {2, 4},
				2: {3, 5},
				3: {6},
				4: {1},
				5: {4},
				6: {5},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 1,
				3: 2,
				4: 1,
				5: 2,
				6: 3,
			},
		},
		{
			name: "non-loop back edges",
			//     0
			//     v
			//     1 --> 2 --> 3 --> 4
			//     ^           |     |
			//     |           v     v
			//     8 <-------- 6 <-- 5
			edges: map[basicBlockID][]basicBlockID{
				0: {1},
				1: {2, 8},
				2: {3},
				3: {4, 6},
				4: {5},
				5: {6},
				6: {8},
				8: {1},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 1,
				3: 2,
				4: 3,
				5: 4,
				6: 3,
				8: 1,
			},
		},
		{
			name: "multiple independent paths",
			//   0
			//   v
			//   1 --> 2 --> 3 --> 4 --> 5
			//   |           ^     ^
			//   v           |     |
			//   6 --> 7 --> 8 --> 9
			edges: map[basicBlockID][]basicBlockID{
				0: {1},
				1: {2, 6},
				2: {3},
				3: {4},
				4: {5},
				6: {7},
				7: {8},
				8: {3, 9},
				9: {4},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 1,
				3: 1,
				4: 1,
				5: 4,
				6: 1,
				7: 6,
				8: 7,
				9: 8,
			},
		},
		{
			name: "nested loops with branches",
			//   0 --> 1 --> 2 --> 3
			//        ^     |     |
			//        |     v     v
			//        6 <-- 4 <-- 5
			//        ^
			//        |
			//        7
			edges: map[basicBlockID][]basicBlockID{
				0: {1},
				1: {2, 6},
				2: {3, 4},
				3: {5},
				4: {6},
				5: {4},
				6: {1, 7},
				7: {6},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 1,
				3: 2,
				4: 2,
				5: 3,
				6: 1,
				7: 6,
			},
		},
		{
			name: "double back edges",
			//     0
			//     v
			//     1 --> 2 --> 3 --> 4 --> 5
			//     ^                 |
			//     |                 v
			//     7 <--------------- 6
			edges: map[basicBlockID][]basicBlockID{
				0: {1},
				1: {2, 7},
				2: {3},
				3: {4},
				4: {5, 6},
				5: {4},
				6: {7},
				7: {1},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 1,
				3: 2,
				4: 3,
				5: 4,
				6: 4,
				7: 1,
			},
		},
		{
			name: "double nested loops with branches",
			//     0 --> 1 --> 2 --> 3 --> 4 --> 5 --> 6
			//          ^     |           |     ^
			//          |     v           v     |
			//          9 <-- 8 <--------- 7 <--|
			edges: map[basicBlockID][]basicBlockID{
				0: {1},
				1: {2, 9},
				2: {3, 8},
				3: {4},
				4: {5, 7},
				5: {6},
				6: {7},
				7: {8},
				8: {9},
				9: {1},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 1,
				3: 2,
				4: 3,
				5: 4,
				6: 5,
				7: 4,
				8: 2,
				9: 1,
			},
		},
		{
			name: "split paths with a loop",
			//       0
			//       v
			//       1
			//      / \
			//     v   v
			//     2<--3
			//     ^   |
			//     |   v
			//     6<--4
			//     |
			//     v
			//     5
			edges: map[basicBlockID][]basicBlockID{
				0: {1},
				1: {2, 3},
				3: {2, 4},
				4: {6},
				6: {2, 5},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 1,
				3: 1,
				4: 3,
				5: 6,
				6: 4,
			},
		},
		{
			name: "multiple exits with a loop",
			//     0
			//     v
			//     1
			//    / \
			//   v   v
			//   2<--3
			//   |
			//   v
			//   5<->4
			//   |
			//   v
			//   6
			edges: map[basicBlockID][]basicBlockID{
				0: {1},
				1: {2, 3},
				2: {5},
				3: {2},
				4: {5},
				5: {4, 6},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 1,
				3: 1,
				4: 5,
				5: 2,
				6: 5,
			},
		},
		{
			name: "parallel loops with merge",
			//       0
			//       v
			//       1
			//      / \
			//     v   v
			//     3<--2
			//     |
			//     v
			//     4<->5
			//     |   ^
			//     v   v
			//     7<->6
			edges: map[basicBlockID][]basicBlockID{
				0: {1},
				1: {2, 3},
				2: {3},
				3: {4},
				4: {5, 7},
				5: {4, 6},
				6: {7},
				7: {6},
			},
			expDoms: map[basicBlockID]basicBlockID{
				1: 0,
				2: 1,
				3: 1,
				4: 3,
				5: 4,
				6: 4,
				7: 4,
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder().(*builder)

			// Allocate blocks.
			blocks := make(map[basicBlockID]*basicBlock, numBlocks)
			for i := 0; i < numBlocks; i++ {
				blk := b.AllocateBasicBlock()
				blocks[basicBlockID(i)] = blk.(*basicBlock)
			}

			// Collect edges.
			var pairs [][2]*basicBlock
			for fromID, toIDs := range tc.edges {
				for _, toID := range toIDs {
					from, to := blocks[fromID], blocks[toID]
					pairs = append(pairs, [2]*basicBlock{from, to})
				}
			}

			// To have a consistent behavior in test, we sort the pairs.
			sort.Slice(pairs, func(i, j int) bool {
				xf, yf := pairs[i][0], pairs[j][0]
				xt, yt := pairs[i][1], pairs[j][1]
				if xf.id < yf.id {
					return true
				}
				return xt.id < yt.id
			})

			// Add edges.
			for _, p := range pairs {
				from, to := p[0], p[1]
				to.addPred(from, &Instruction{})
			}

			passCalculateDominatorTree(b)

			for blockID, expDomID := range tc.expDoms {
				expBlock := blocks[expDomID]
				require.Equal(t, expBlock, b.dominators[blockID],
					"block %d expecting %d, but got %s", blockID, expDomID, b.dominators[blockID])
			}
		})
	}
}
