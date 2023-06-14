package ssa

// Optimize implements Builder.Optimize.
func (b *builder) Optimize() {
	for _, pass := range defaultOptimizationPasses {
		pass(b)
	}
}

// optimizationPass represents a pass which operates on and manipulates the SSA function
// constructed in the given builder.
type optimizationPass func(*builder)

// defaultOptimizationPasses consists of the optimization passes run by default.
var defaultOptimizationPasses = []optimizationPass{
	passDeadBlockElimination,
	passRedundantPhiElimination,
	// TODO: block coalescing.
	// TODO: Copy-propagation.
	// TODO: Constant folding.
	// TODO: Common subexpression elimination.
	// TODO: Arithmetic simplifications.
	// TODO: and more!
	passDeadCodeElimination,
}

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

	for i := 0; i < b.basicBlocksPool.allocated; i++ {
		blk := b.basicBlocksPool.view(i)
		if _, ok := b.blkVisited[blk]; !ok {
			blk.invalid = true
		}
	}
}

// passRedundantPhiElimination eliminates the redundant PHIs (in our terminology, parameters of a block).
func passRedundantPhiElimination(b *builder) {
	// Intentionally use the named iteration variable name, as this comes with inevitable nested for loops!
	for blockIndex := 1; /* skip entry block! */ blockIndex < b.basicBlocksPool.allocated; blockIndex++ {
		blk := b.basicBlocksPool.view(blockIndex)
		if blk.invalid {
			// Already removed block.
			continue
		}

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
//
// This also calculates the instructionGroupID of each SSA Instruction: divided by side effects and blocks.
func passDeadCodeElimination(b *builder) {
	if iid := int(b.nextValueID); iid >= len(b.valueRefCounts) {
		b.valueRefCounts = append(b.valueRefCounts, make([]int, b.nextValueID)...)
	}

	var gid instructionGroupID = 0
	for blockIndex := 0; blockIndex < b.basicBlocksPool.allocated; blockIndex++ {
		blk := b.basicBlocksPool.view(blockIndex)
		if blk.invalid {
			// Already removed block.
			continue
		}

		// In order to calculate the exact refCount, we pre-oder traverse the instruction tree.
		// TODO

		root := blk.currentInstr

		// instructionGroupID won't be shared by instructions in different blocks.
		gid++
	}
}
