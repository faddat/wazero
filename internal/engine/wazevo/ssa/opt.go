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
	// This must be the second to the last as it gathers the value usage count info for backends to use.
	passDeadCodeElimination(b)
	// Finally we can assign instruction group ID.
	passInstructionGroupIDAssignment(b)
}

// optimizationPass represents a pass which operates on and manipulates the SSA function
// constructed in the given builder.
type optimizationPass func(*builder)

// passDeadBlockElimination searches the unreachable blocks, and sets the basicBlock.invalid flag true if so.
func passDeadBlockElimination(b *builder) {
	entryBlk := b.basicBlocksPool.view(0)
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

				if !nonSelfReferencingValue.valid() {
					nonSelfReferencingValue = pred
					continue
				}

				if nonSelfReferencingValue != pred {
					redundant = false
					break
				}
			}

			if !nonSelfReferencingValue.valid() {
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
			blk.alias(newValue, phiValue)
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
func passDeadCodeElimination(b *builder) {
	if iid := int(b.nextValueID); iid >= len(b.valueRefCounts) {
		b.valueRefCounts = append(b.valueRefCounts, make([]int, b.nextValueID)...)
	}

	for blk := b.blockIteratorBegin(); blk != nil; blk = b.blockIteratorNext() {
		// TODO!!
	}
}

// passInstructionGroupIDAssignment assigns a InstructionGroupID to each Instruction.
// This is the last SSA-level optimization pass and after this, the SSA function is ready to be used by backends.
func passInstructionGroupIDAssignment(b *builder) {
	var gid InstructionGroupID
	for blk := b.blockIteratorBegin(); blk != nil; blk = b.blockIteratorNext() {
		// Walk through the instructions in this block.
		cur := blk.rootInstr
		for ; cur != nil; cur = cur.next {
			if instructionSideEffects[cur.opcode] {
				// Side effects create different instruction groups.
				gid++
			}
			cur.gid = gid
		}

		// Instructions in different blocks should have different group IDs.
		gid++
	}
}
