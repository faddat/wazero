package ssa

// Optimize implements Builder.Optimize.
func (b *builder) Optimize() {
	for _, pass := range defaultOptimizationPasses {
		pass(b)
	}
}

type optimizationPass func(*builder)

var defaultOptimizationPasses = []optimizationPass{
	// TODO: block coalescing.
	// TODO: dead code elimination.
	// TODO: constant phi elimination.
	// TODO: redundant phi elimination.
	// TODO: Copy-propagation.
	// TODO: Constant folding.
	// TODO: Common subexpression elimination.
	// TODO: Arithmetic simplifications.
}
