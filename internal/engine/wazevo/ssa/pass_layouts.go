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
	// type edge [2]basicBlockID

	//edgeWeights := make(map[edge]int)
	//for blk := b.blockIteratorBegin(); blk != nil; blk = b.blockIteratorNext() {
	//}
}

// passLayoutBlocks determines the order of basic blocks by using the block frequency info calculated by passBlockFrequency.
//
// TODO: The current algorithm is just a simple greedy algorithm. While it is a good starting point,
// but there are many ways to improve this. E.g. Pettis-Hansen algorithm could be used as in LLVM.
func passLayoutBlocks(b *builder) {
}
