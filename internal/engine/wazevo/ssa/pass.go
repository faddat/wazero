package ssa

// RunPasses implements Builder.RunPasses.
//
// The order here matters; some pass depends on the previous ones.
//
// Note that passes suffixed with "Opt" are the optimization passes, meaning that they edit the instructions and blocks
// while the other passes are not, like passBlockFrequency does not edit them, but only calculates the additional information.
func (b *builder) RunPasses() {
	passDeadBlockEliminationOpt(b)
	passRedundantPhiEliminationOpt(b)
	// The result of passCalculateImmediateDominators will be used by various passes below.
	passCalculateImmediateDominators(b)

	// TODO: implement more optimization passes like:
	// 	block coalescing.
	// 	Copy-propagation.
	// 	Constant folding.
	// 	Common subexpression elimination.
	// 	Arithmetic simplifications.
	// 	and more!

	// passDeadCodeEliminationOpt could be more accurate if we do this after other optimizations.
	passDeadCodeEliminationOpt(b)
	passBlockFrequency(b)
	// passLayoutBlocks depends on passLayoutBlocks.
	passLayoutBlocks(b)
}

// passDeadBlockEliminationOpt searches the unreachable blocks, and sets the basicBlock.invalid flag true if so.
func passDeadBlockEliminationOpt(b *builder) {
	entryBlk := b.entryBlk()
	b.clearBlkVisited()
	b.blkStack = append(b.blkStack, entryBlk)
	for len(b.blkStack) > 0 {
		reachableBlk := b.blkStack[len(b.blkStack)-1]
		b.blkStack = b.blkStack[:len(b.blkStack)-1]
		b.blkVisited[reachableBlk] = 0 // the value won't be used in this pass.

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

// passRedundantPhiEliminationOpt eliminates the redundant PHIs (in our terminology, parameters of a block).
func passRedundantPhiEliminationOpt(b *builder) {
	redundantParameterIndexes := b.ints[:0] // reuse the slice from previous iterations.

	_ = b.blockIteratorBegin() // skip entry block!
	// Below, we intentionally use the named iteration variable name, as this comes with inevitable nested for loops!
	for blk := b.blockIteratorNext(); blk != nil; blk = b.blockIteratorNext() {
		paramNum := len(blk.params)

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
				redundantParameterIndexes = append(redundantParameterIndexes, paramIndex)
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
		for _, redundantParamIndex := range redundantParameterIndexes {
			phiValue := blk.params[redundantParamIndex].value
			onlyValue := b.redundantParameterIndexToValue[redundantParamIndex]
			// Create an alias in this block from the only phi argument to the phi value.
			b.alias(phiValue, onlyValue)
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
		for _, paramIndex := range redundantParameterIndexes {
			delete(b.redundantParameterIndexToValue, paramIndex)
		}
		redundantParameterIndexes = redundantParameterIndexes[:0]
	}

	// Reuse the slice for the future passes.
	b.ints = redundantParameterIndexes
}

// passDeadCodeEliminationOpt traverses all the instructions, and calculates the reference count of each Value, and
// eliminates all the unnecessary instructions whose ref count is zero.
// The results are stored at builder.valueRefCounts. This also assigns a InstructionGroupID to each Instruction
// during the process. This is the last SSA-level optimization pass and after this,
// the SSA function is ready to be used by backends.
//
// TODO: the algorithm here might not be efficient. Get back to this later.
func passDeadCodeEliminationOpt(b *builder) {
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

// clearBlkVisited clears the b.blkVisited map so that we can reuse it for multiple places.
func (b *builder) clearBlkVisited() {
	for i := 0; i < b.basicBlocksPool.Allocated(); i++ {
		blk := b.basicBlocksPool.View(i)
		delete(b.blkVisited, blk)
	}
}