package ssa

// Optimize implements Builder.Optimize.
func (b *builder) Optimize() {
	for _, pass := range defaultOptimizationPasses {
		pass(b)
	}
}

type optimizationPass func(*builder)

var defaultOptimizationPasses = []optimizationPass{
	deadCodeElimination,
	// TODO: block coalescing.
	// TODO: constant phi elimination.
	// TODO: redundant phi elimination.
	// TODO: Copy-propagation.
	// TODO: Constant folding.
	// TODO: Common subexpression elimination.
	// TODO: Arithmetic simplifications.
}

// deadCodeElimination searches the unreachable blocks, and sets the basicBlock.invalid flag true if so.
func deadCodeElimination(b *builder) {
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
