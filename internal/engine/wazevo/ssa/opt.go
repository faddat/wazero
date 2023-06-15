package ssa

// Optimize implements Builder.Optimize.
func (b *builder) Optimize() {
	passDeadBlockElimination(b)
	passRedundantPhiElimination(b)
	// TODO: block coalescing.
	// TODO: Copy-propagation.
	// TODO: Constant folding.
	// TODO: Common subexpression elimination.
	// TODO: Arithmetic simplifications.
	// TODO: and more!
	// This is the last as it gathers the value usage count and instructionGroupID info for backends to use.
	passDeadCodeElimination(b)
}

// passDeadBlockElimination searches the unreachable blocks, and sets the basicBlock.invalid flag true if so.
func passDeadBlockElimination(b *builder) {
	entryBlk := b.basicBlocksPool.View(0)
	b.blkStack = append(b.blkStack, entryBlk)
	for len(b.blkStack) > 0 {
		reachableBlk := b.blkStack[len(b.blkStack)-1]
		b.blkStack = b.blkStack[:len(b.blkStack)-1]
		b.blkVisited[reachableBlk] = struct{}{}

		for _, successor := range reachableBlk.success {
			if _, ok := b.blkVisited[successor]; ok {
				continue
			}
			b.blkStack = append(b.blkStack, successor)
		}
	}

	for blk := b.blockIteratorBegin(); blk != nil; blk = b.blockIteratorNext() {
		if _, ok := b.blkVisited[blk]; !ok {
			blk.invalid = true
		}
	}
}

// passRedundantPhiElimination eliminates the redundant PHIs (in our terminology, parameters of a block).
func passRedundantPhiElimination(b *builder) {
	blk := b.blockIteratorBegin()
	// Below, we intentionally use the named iteration variable name, as this comes with inevitable nested for loops!
	for blk = b.blockIteratorNext(); /* skip entry block! */ blk != nil; blk = b.blockIteratorNext() {
		paramNum := len(blk.params)

		// We will store the unnecessary param's index into b.ints.
		for paramIndex := 0; paramIndex < paramNum; paramIndex++ {
			phiValue := blk.params[paramIndex].value
			redundant := true

			nonSelfReferencingValue := valueInvalid
			for predIndex := range blk.preds {
				pred := blk.preds[predIndex].branch.vs[paramIndex]
				if pred == phiValue {
					// This is self-referencing: PHI from the same PHI.
					continue
				}

				if !nonSelfReferencingValue.Valid() {
					nonSelfReferencingValue = pred
					continue
				}

				if nonSelfReferencingValue != pred {
					redundant = false
					break
				}
			}

			if !nonSelfReferencingValue.Valid() {
				// This shouldn't happen, and must be a bug in builder.go.
				panic("BUG: params added but only self-referencing")
			}

			if redundant {
				b.redundantParameterIndexToValue[paramIndex] = nonSelfReferencingValue
				b.redundantParameterIndexes = append(b.redundantParameterIndexes, paramIndex)
			}
		}

		if len(b.redundantParameterIndexToValue) == 0 {
			continue
		}

		// Remove the redundant PHIs from the argument list of branching instructions.
		for predIndex := range blk.preds {
			var cur int
			predBlk := blk.preds[predIndex]
			branchInst := predBlk.branch
			for argIndex, value := range branchInst.vs {
				if _, ok := b.redundantParameterIndexToValue[argIndex]; !ok {
					branchInst.vs[cur] = value
					cur++
				}
			}
			branchInst.vs = branchInst.vs[:cur]
		}

		// Still need to have the definition of the value of the PHI (previously as the parameter).
		for _, redundantParamIndex := range b.redundantParameterIndexes {
			phiValue := blk.params[redundantParamIndex].value
			newValue := b.redundantParameterIndexToValue[redundantParamIndex]
			// Create an alias in this block temporarily to
			b.alias(phiValue, newValue)
		}

		// Finally, Remove the param from the blk.
		var cur int
		for paramIndex := 0; paramIndex < paramNum; paramIndex++ {
			param := blk.params[paramIndex]
			if _, ok := b.redundantParameterIndexToValue[paramIndex]; !ok {
				blk.params[cur] = param
				cur++
			}
		}
		blk.params = blk.params[:cur]

		// Clears the map for the next iteration.
		for _, paramIndex := range b.redundantParameterIndexes {
			delete(b.redundantParameterIndexToValue, paramIndex)
		}
		b.redundantParameterIndexes = b.redundantParameterIndexes[:0]
	}
}

// passDeadCodeElimination traverses all the instructions, and calculates the reference count of each Value,
// and eliminates all the unnecessary instructions whose ref count is zero. The results are stored at builder.valueRefCounts.
//
// This also assigns a InstructionGroupID to each Instruction during the process.
//
// This is the last SSA-level optimization pass and after this, the SSA function is ready to be used by backends.
//
// TODO: the algorithm here might not be efficient. Get back to this later.
func passDeadCodeElimination(b *builder) {
	nvid := int(b.nextValueID)
	if nvid >= len(b.valueRefCounts) {
		b.valueRefCounts = append(b.valueRefCounts, make([]int, b.nextValueID)...)
	}
	if nvid >= len(b.valueIDToInstruction) {
		b.valueIDToInstruction = append(b.valueIDToInstruction, make([]*Instruction, b.nextValueID)...)
	}

	// First, we gather all the instructions with side effects.
	liveInstructions := b.instStack[:0]
	// During the process, we will assign InstructionGroupID to each instruction, which is not
	// relevant to dead code elimination, but we need in the backend.
	var gid InstructionGroupID
	for blk := b.blockIteratorBegin(); blk != nil; blk = b.blockIteratorNext() {
		for cur := blk.rootInstr; cur != nil; cur = cur.next {
			cur.gid = gid
			if cur.HasSideEffects() {
				liveInstructions = append(liveInstructions, cur)
				// Side effects create different instruction groups.
				gid++
			}

			r1, rs := cur.Returns()
			if r1.Valid() {
				b.valueIDToInstruction[r1.ID()] = cur
			}
			for _, r := range rs {
				b.valueIDToInstruction[r.ID()] = cur
			}
		}
	}

	// TODO: take alias into account.

	// Find all the instructions referenced by live instructions transitively.
	for len(liveInstructions) > 0 {
		tail := len(liveInstructions) - 1
		live := liveInstructions[tail]
		liveInstructions = liveInstructions[:tail]
		if live.live {
			// If it's already marked alive, this is referenced multiple times,
			// so we can skip it.
			continue
		}
		live.live = true

		// Before we walk, we need to resolve the alias first.
		b.resolveArgumentAlias(live)

		v1, v2, vs := live.args()
		if v1.Valid() {
			producingInst := b.valueIDToInstruction[v1.ID()]
			if producingInst != nil {
				liveInstructions = append(liveInstructions, producingInst)
			}
		}

		if v2.Valid() {
			producingInst := b.valueIDToInstruction[v2.ID()]
			if producingInst != nil {
				liveInstructions = append(liveInstructions, producingInst)
			}
		}

		for _, v := range vs {
			producingInst := b.valueIDToInstruction[v.ID()]
			if producingInst != nil {
				liveInstructions = append(liveInstructions, producingInst)
			}
		}
	}

	// Now that all the live instructions are flagged as live=true, we eliminate all dead instructions.
	for blk := b.blockIteratorBegin(); blk != nil; blk = b.blockIteratorNext() {
		for cur := blk.rootInstr; cur != nil; cur = cur.next {
			if !cur.live {
				// Remove the instruction from the list.
				if prev := cur.prev; prev != nil {
					prev.next = cur.next
				} else {
					blk.rootInstr = cur.next
				}
				if next := cur.next; next != nil {
					next.prev = cur.prev
				}
				continue
			}

			// If the value alive, we can be sure that arguments are used definitely.
			// Hence, we can increment the value reference counts.
			v1, v2, vs := cur.args()
			if v1.Valid() {
				b.valueRefCounts[v1.ID()]++
			}
			if v2.Valid() {
				b.valueRefCounts[v2.ID()]++
			}
			for _, v := range vs {
				b.valueRefCounts[v.ID()]++
			}
		}
	}

	b.instStack = liveInstructions // we reuse the stack for the next iteration.
}
