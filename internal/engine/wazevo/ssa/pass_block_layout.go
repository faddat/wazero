package ssa

// passBlockFrequency calculates the block frequency of each block.
// This is similar to what BlockFrequencyInfo pass does in LLVM:
// https://llvm.org/doxygen/classllvm_1_1BlockFrequencyInfoImpl.html#details
//
// The calculated info will be necessary for backend to determine the order of basic block layout
// which is similar to MachineBlockPlacement pass in LLVM: https://llvm.org/doxygen/MachineBlockPlacement_8cpp_source.html
//
// TODO: currently the algorithm is very simple and naive. We need to improve this later.
// e.g. we could add more heuristics, or use the profile data if available.
// e.g. Ball-Larus algorithm: https://www.cs.cornell.edu/courses/cs6120/2019fa/blog/efficient-path-prof/
func passBlockFrequency(b *builder) {
	// First, we calculate the edge weight with heuristics.
	for blk := b.blockIteratorBegin(); blk != nil; blk = b.blockIteratorNext() {
		ss := blk.success
		switch len(ss) {
		case 0:
		case 1:
			// The sole successor should be higher weights.
			b.assignEdgeWeight(blk, ss[0], 10)
		case 2:
			thenBlk, elseBlk := ss[0], ss[1]
			thenIsLoop := thenBlk.loopHeader && b.isDominatedBy(blk, thenBlk)
			elseIsLoop := elseBlk.loopHeader && b.isDominatedBy(blk, elseBlk)

			// Assign higher weight to loop back-edges.
			if thenIsLoop {
				// When both are loop back-edges, we assign higher weight to thenBlk
				// because it is more likely to be a hot path (I guess....).
				b.assignEdgeWeight(blk, thenBlk, 10)
				b.assignEdgeWeight(blk, elseBlk, 1)
				break // break switch!
			} else if elseIsLoop {
				b.assignEdgeWeight(blk, thenBlk, 1)
				b.assignEdgeWeight(blk, elseBlk, 10)
				break // break switch!
			}

			lastJump := blk.currentInstr
			if lastJump.opcode != OpcodeJump {
				panic("BUG") // sanity check. TODO: delete this later.
			}

			// Assign higher weight to the fallthrough edge which is the target of the last branching instruction.
			if blk.currentInstr.blk.(*basicBlock) == thenBlk {
				b.assignEdgeWeight(blk, thenBlk, 10)
				b.assignEdgeWeight(blk, elseBlk, 1)
			} else {
				b.assignEdgeWeight(blk, thenBlk, 1)
				b.assignEdgeWeight(blk, elseBlk, 10)
			}
		default:
			panic("TODO: blocks with more than 2 successors are not supported yet i.e. OpCodeBrTable instruction")
		}
	}

	// Now that we have the edge weights, we can calculate the block frequency.

	// Initialize the frequencies to 1 for entry block and 0 for others.
	const initialBlockFrequencyForEntry = 1
	// Reuse the blockFrequencies slice from the previous iteration.
	b.blockFrequencies = b.blockFrequencies[:0]
	for i := 0; i < b.basicBlocksPool.Allocated(); i++ {
		b.blockFrequencies = append(b.blockFrequencies, 0)
	}
	b.blockFrequencies[0] = initialBlockFrequencyForEntry

	// Propagate frequencies until it converges from the entry block.
	for changed := true; changed; changed = false {
		for blk := b.blockIteratorBegin(); blk != nil; blk = b.blockIteratorNext() {
			var newFreq int
			for i := range blk.preds {
				pred := blk.preds[i].blk
				newFreq += b.blockFrequencies[pred.id] * b.edgeWeight(pred, blk)
			}

			if b.blockFrequencies[blk.id] != newFreq {
				b.blockFrequencies[blk.id] = newFreq
				changed = true
			}
		}
	}
}

// passLayoutBlocks determines the order of basic blocks by using the block frequency info calculated by passBlockFrequency.
//
// TODO: The current algorithm is just a simple greedy algorithm. While it is a good starting point,
// but there are many ways to improve this. E.g. Pettis-Hansen algorithm could be used as in LLVM.
func passLayoutBlocks(b *builder) {
}
