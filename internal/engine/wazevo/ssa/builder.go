package ssa

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/wazevoapi"
)

// Builder is used to builds SSA consisting of Basic Blocks per function.
type Builder interface {
	// Reset must be called to reuse this builder for the next function.
	Reset()

	// AllocateBasicBlock creates a basic block in SSA function.
	AllocateBasicBlock() BasicBlock

	// CurrentBlock returns the currently handled BasicBlock which is set by the latest call to SetCurrentBlock.
	CurrentBlock() BasicBlock

	// SetCurrentBlock sets the instruction insertion target to the BasicBlock `b`.
	SetCurrentBlock(b BasicBlock)

	// DeclareVariable declares a Variable of the given Type.
	DeclareVariable(Type) Variable

	// DefineVariable defines a variable in the `block` with value.
	// The defining instruction will be inserted into the `block`.
	DefineVariable(variable Variable, value Value, block BasicBlock)

	// DefineVariableInCurrentBB is the same as DefineVariable except the definition is
	// inserted into the current BasicBlock. Alias to DefineVariable(x, y, CurrentBlock()).
	DefineVariableInCurrentBB(variable Variable, value Value)

	// AllocateInstruction returns a new Instruction.
	AllocateInstruction() *Instruction

	// InsertInstruction executes BasicBlock.InsertInstruction for the currently handled basic block.
	InsertInstruction(raw *Instruction)

	// allocateValue allocates an unused Value.
	allocateValue(typ Type) Value

	// FindValue searches the latest definition of the given Variable and returns the result.
	FindValue(variable Variable) Value

	// Seal declares that we've known all the predecessors to this block and were added via AddPred.
	// After calling this, AddPred will be forbidden.
	Seal(blk BasicBlock)

	// AnnotateValue is for debugging purpose.
	AnnotateValue(value Value, annotation string)

	// DeclareSignature appends the *Signature to be referenced by various instructions (e.g. OpcodeCall).
	DeclareSignature(signature *Signature)

	// UsedSignatures returns the slice of Signatures which are used/referenced by the currently-compiled function.
	UsedSignatures() []*Signature

	// RunPasses runs various optimization passes on the constructed SSA function.
	RunPasses()

	// Format returns the debugging string of the SSA function.
	Format() string

	// BlockIteratorBegin initializes the state to iterate over all the valid BasicBlock(s) compiled.
	// Combined with BlockIteratorNext, we can use this like:
	//
	// 	for blk := builder.BlockIteratorBegin(); blk != nil; blk = builder.BlockIteratorNext() {
	// 		// ...
	//	}
	//
	// The returned blocks are ordered in the order of AllocateBasicBlock being called.
	BlockIteratorBegin() BasicBlock

	// BlockIteratorNext advances the state for iteration initialized by BlockIteratorBegin.
	// Returns nil if there's no unseen BasicBlock.
	BlockIteratorNext() BasicBlock

	// BlockIteratorReversePostOrderBegin is almost the same as BlockIteratorBegin except it returns the BasicBlock in the reverse post-order.
	// This is available after passCalculateImmediateDominators is run.
	BlockIteratorReversePostOrderBegin() BasicBlock

	// BlockIteratorReversePostOrderNext is almost the same as BlockIteratorPostOrderNext except it returns the BasicBlock in the reverse post-order.
	// This is available after passCalculateImmediateDominators is run.
	BlockIteratorReversePostOrderNext() BasicBlock

	// ValueRefCountMap returns the map of ValueID to its reference count.
	// The returned slice must not be modified.
	ValueRefCountMap() []int
}

// NewBuilder returns a new Builder implementation.
func NewBuilder() Builder {
	return &builder{
		instructionsPool:               wazevoapi.NewPool[Instruction](),
		basicBlocksPool:                wazevoapi.NewPool[basicBlock](),
		valueAnnotations:               make(map[ValueID]string),
		signatures:                     make(map[SignatureID]*Signature),
		blkVisited:                     make(map[*basicBlock]int),
		valueIDAliases:                 make(map[ValueID]Value),
		redundantParameterIndexToValue: make(map[int]Value),
	}
}

// builder implements Builder interface.
type builder struct {
	basicBlocksPool  wazevoapi.Pool[basicBlock]
	instructionsPool wazevoapi.Pool[Instruction]
	signatures       map[SignatureID]*Signature

	// reversePostOrderedBasicBlocks are the BasicBlock(s) ordered in the reverse post-order after passCalculateImmediateDominators.
	reversePostOrderedBasicBlocks []*basicBlock
	currentBB                     *basicBlock

	// variables track the types for Variable with the index regarded Variable.
	variables []Type
	// nextValueID is used by builder.AllocateValue.
	nextValueID ValueID
	// nextVariable is used by builder.AllocateVariable.
	nextVariable Variable

	valueIDAliases   map[ValueID]Value
	valueAnnotations map[ValueID]string

	// valueRefCounts is used to lower the SSA in backend, and will be calculated
	// by the last SSA-level optimization pass.
	valueRefCounts []int

	// dominators stores the immediate dominator of each BasicBlock.
	// The index is blockID of the BasicBlock.
	dominators []*basicBlock

	// The followings are used for optimization passes.
	instStack                      []*Instruction
	blkVisited                     map[*basicBlock]int
	valueIDToInstruction           []*Instruction
	blkStack                       []*basicBlock
	blkStack2                      []*basicBlock
	ints                           []int
	redundantParameterIndexToValue map[int]Value

	// blockIterCur is used to implement blockIteratorBegin and blockIteratorNext.
	blockIterCur int
}

// Reset implements Builder.Reset.
func (b *builder) Reset() {
	b.instructionsPool.Reset()
	for _, sig := range b.signatures {
		sig.used = false
	}

	b.ints = b.ints[:0]
	b.blkStack = b.blkStack[:0]
	b.blkStack2 = b.blkStack2[:0]
	b.dominators = b.dominators[:0]

	for i := 0; i < b.basicBlocksPool.Allocated(); i++ {
		blk := b.basicBlocksPool.View(i)
		blk.reset()
		delete(b.blkVisited, blk)
	}
	b.basicBlocksPool.Reset()

	for i := Variable(0); i < b.nextVariable; i++ {
		b.variables[i] = typeInvalid
	}

	for v := ValueID(0); v < b.nextValueID; v++ {
		delete(b.valueAnnotations, v)
		delete(b.valueIDAliases, v)
		b.valueRefCounts[v] = 0
		b.valueIDToInstruction[v] = nil
	}
	b.nextValueID = 0
	b.reversePostOrderedBasicBlocks = b.reversePostOrderedBasicBlocks[:0]
}

// AnnotateValue implements Builder.AnnotateValue.
func (b *builder) AnnotateValue(value Value, a string) {
	b.valueAnnotations[value.ID()] = a
}

// AllocateInstruction implements Builder.AllocateInstruction.
func (b *builder) AllocateInstruction() *Instruction {
	instr := b.instructionsPool.Allocate()
	instr.reset()
	return instr
}

// DeclareSignature implements Builder.AnnotateValue.
func (b *builder) DeclareSignature(s *Signature) {
	b.signatures[s.ID] = s
	s.used = false
}

// UsedSignatures implements Builder.UsedSignatures.
func (b *builder) UsedSignatures() (ret []*Signature) {
	for _, sig := range b.signatures {
		if sig.used {
			ret = append(ret, sig)
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].ID < ret[j].ID
	})

	return
}

// AllocateBasicBlock implements Builder.AllocateBasicBlock.
func (b *builder) AllocateBasicBlock() BasicBlock {
	id := basicBlockID(b.basicBlocksPool.Allocated())
	blk := b.basicBlocksPool.Allocate()
	blk.id = id
	blk.lastDefinitions = make(map[Variable]Value)
	blk.unknownValues = make(map[Variable]Value)
	return blk
}

// InsertInstruction implements Builder.InsertInstruction.
func (b *builder) InsertInstruction(instr *Instruction) {
	b.currentBB.InsertInstruction(instr)

	resultTypesFn := instructionReturnTypes[instr.opcode]
	if resultTypesFn == nil {
		panic("TODO: " + instr.Format(b))
	}

	t1, ts := resultTypesFn(b, instr)
	if t1.invalid() {
		return
	}

	r1 := b.allocateValue(t1)
	instr.rValue = r1

	tsl := len(ts)
	if tsl == 0 {
		return
	}

	// TODO: reuse slices, though this seems not to be common.
	instr.rValues = make([]Value, tsl)
	for i := 0; i < tsl; i++ {
		instr.rValues[i] = b.allocateValue(ts[i])
	}
}

// DefineVariable implements Builder.DefineVariable.
func (b *builder) DefineVariable(variable Variable, value Value, block BasicBlock) {
	if b.variables[variable].invalid() {
		panic("BUG: trying to define variable " + variable.String() + " but is not declared yet")
	}

	bb := block.(*basicBlock)
	bb.lastDefinitions[variable] = value
}

// DefineVariableInCurrentBB implements Builder.DefineVariableInCurrentBB.
func (b *builder) DefineVariableInCurrentBB(variable Variable, value Value) {
	b.DefineVariable(variable, value, b.currentBB)
}

// SetCurrentBlock implements Builder.SetCurrentBlock.
func (b *builder) SetCurrentBlock(bb BasicBlock) {
	b.currentBB = bb.(*basicBlock)
}

// CurrentBlock implements Builder.CurrentBlock.
func (b *builder) CurrentBlock() BasicBlock {
	return b.currentBB
}

// DeclareVariable implements Builder.DeclareVariable.
func (b *builder) DeclareVariable(typ Type) Variable {
	v := b.allocateVariable()
	iv := int(v)
	if l := len(b.variables); l <= iv {
		b.variables = append(b.variables, make([]Type, 2*(l+1))...)
	}
	b.variables[v] = typ
	return v
}

// allocateVariable allocates a new variable.
func (b *builder) allocateVariable() (ret Variable) {
	ret = b.nextVariable
	b.nextVariable++
	return
}

// allocateValue implements Builder.AllocateValue.
func (b *builder) allocateValue(typ Type) (v Value) {
	v = Value(b.nextValueID)
	v.setType(typ)
	b.nextValueID++
	return
}

// FindValue implements Builder.FindValue.
func (b *builder) FindValue(variable Variable) Value {
	typ := b.definedVariableType(variable)
	return b.findValue(typ, variable, b.currentBB)
}

// findValue recursively tries to find the latest definition of a `variable`. The algorithm is described in
// the section 2 of the paper https://link.springer.com/content/pdf/10.1007/978-3-642-37051-9_6.pdf.
//
// TODO: reimplement this in iterative, not recursive, to avoid stack overflow.
func (b *builder) findValue(typ Type, variable Variable, blk *basicBlock) Value {
	if val, ok := blk.lastDefinitions[variable]; ok {
		// The value is already defined in this block!
		return val
	} else if !blk.sealed { // Incomplete CFG as in the paper.
		// If this is not sealed, that means it might have additional unknown predecessor later on.
		// So we temporarily define the placeholder value here (not add as a parameter yet!),
		// and record it as unknown.
		// The unknown values are resolved when we call seal this block via BasicBlock.Seal().
		value := b.allocateValue(typ)
		blk.lastDefinitions[variable] = value
		blk.unknownValues[variable] = value
		return value
	}

	if pred := blk.singlePred; pred != nil {
		// If this block is sealed and have only one predecessor,
		// we can use the value in that block without ambiguity on definition.
		return b.findValue(typ, variable, pred)
	}

	// If this block has multiple predecessors, we have to gather the definitions,
	// and treat them as an argument to this block. So the first thing we do now is
	// define a new parameter to this block which may or may not be redundant, but
	// later we eliminate trivial params in an optimization pass.
	paramValue := blk.AddParam(b, typ)
	b.DefineVariable(variable, paramValue, blk)
	// After the new param is added, we have to manipulate the original branching instructions
	// in predecessors so that they would pass the definition of `variable` as the argument to
	// the newly added PHI.
	for i := range blk.preds {
		pred := &blk.preds[i]
		// Find the definition in the predecessor recursively.
		value := b.findValue(typ, variable, pred.blk)
		pred.branch.addArgumentBranchInst(value)
	}
	return paramValue
}

// Seal implements Builder.Seal.
func (b *builder) Seal(raw BasicBlock) {
	blk := raw.(*basicBlock)
	if len(blk.preds) == 1 {
		blk.singlePred = blk.preds[0].blk
	}
	blk.sealed = true

	for variable, phiValue := range blk.unknownValues {
		typ := b.definedVariableType(variable)
		blk.addParamOn(typ, phiValue)
		for i := range blk.preds {
			pred := &blk.preds[i]
			predValue := b.findValue(typ, variable, pred.blk)
			pred.branch.addArgumentBranchInst(predValue)
		}
	}
}

// definedVariableType returns the type of the given variable. If the variable is not defined yet, it panics.
func (b *builder) definedVariableType(variable Variable) Type {
	typ := b.variables[variable]
	if typ.invalid() {
		panic(fmt.Sprintf("%s is not defined yet", variable))
	}
	return typ
}

// Format implements Builder.Format.
func (b *builder) Format() string {
	str := strings.Builder{}
	usedSigs := b.UsedSignatures()
	if len(usedSigs) > 0 {
		str.WriteByte('\n')
		str.WriteString("signatures:\n")
		for _, sig := range usedSigs {
			str.WriteByte('\t')
			str.WriteString(sig.String())
			str.WriteByte('\n')
		}
	}

	for blk := b.BlockIteratorBegin(); blk != nil; blk = b.BlockIteratorNext() {
		bb := blk.(*basicBlock)
		str.WriteByte('\n')
		str.WriteString(bb.FormatHeader(b))
		str.WriteByte('\n')

		for cur := bb.Root(); cur != nil; cur = cur.Next() {
			str.WriteByte('\t')
			str.WriteString(cur.Format(b))
			str.WriteByte('\n')
		}
	}
	return str.String()
}

// BlockIteratorNext implements Builder.BlockIteratorNext.
func (b *builder) BlockIteratorNext() BasicBlock {
	if blk := b.blockIteratorNext(); blk == nil {
		return nil // BasicBlock((*basicBlock)(nil)) != BasicBlock(nil)
	} else {
		return blk
	}
}

// BlockIteratorNext implements Builder.BlockIteratorNext.
func (b *builder) blockIteratorNext() *basicBlock {
	index := b.blockIterCur
	for {
		if index == b.basicBlocksPool.Allocated() {
			return nil
		}
		ret := b.basicBlocksPool.View(index)
		index++
		if !ret.invalid {
			b.blockIterCur = index
			return ret
		}
	}
}

// BlockIteratorBegin implements Builder.BlockIteratorBegin.
func (b *builder) BlockIteratorBegin() BasicBlock {
	return b.blockIteratorBegin()
}

// BlockIteratorBegin implements Builder.BlockIteratorBegin.
func (b *builder) blockIteratorBegin() *basicBlock {
	b.blockIterCur = 0
	return b.blockIteratorNext()
}

// BlockIteratorReversePostOrderBegin implements Builder.BlockIteratorReversePostOrderBegin.
func (b *builder) BlockIteratorReversePostOrderBegin() BasicBlock {
	return b.blockIteratorReversePostOrderBegin()
}

// BlockIteratorBegin implements Builder.BlockIteratorBegin.
func (b *builder) blockIteratorReversePostOrderBegin() *basicBlock {
	b.blockIterCur = 0
	return b.blockIteratorNext()
}

// BlockIteratorReversePostOrderNext implements Builder.BlockIteratorReversePostOrderNext.
func (b *builder) BlockIteratorReversePostOrderNext() BasicBlock {
	if blk := b.blockIteratorNext(); blk == nil {
		return nil // BasicBlock((*basicBlock)(nil)) != BasicBlock(nil)
	} else {
		return blk
	}
}

// BlockIteratorNext implements Builder.BlockIteratorNext.
func (b *builder) blockIteratorReversePostOrderNext() *basicBlock {
	if b.blockIterCur >= len(b.reversePostOrderedBasicBlocks) {
		return nil
	} else {
		return b.reversePostOrderedBasicBlocks[b.blockIterCur]
	}
}

// ValueRefCountMap implements Builder.ValueRefCountMap.
func (b *builder) ValueRefCountMap() []int {
	return b.valueRefCounts
}

// alias records the alias of the given values. The alias(es) will be
// eliminated in the optimization pass via resolveArgumentAlias.
func (b *builder) alias(dst, src Value) {
	b.valueIDAliases[dst.ID()] = src
}

// resolveArgumentAlias resolves the alias of the arguments of the given instruction.
func (b *builder) resolveArgumentAlias(instr *Instruction) {
	if src, ok := b.valueIDAliases[instr.v.ID()]; ok {
		instr.v = src
	}
	if src, ok := b.valueIDAliases[instr.v2.ID()]; ok {
		instr.v2 = src
	}
	for i, v := range instr.vs {
		if src, ok := b.valueIDAliases[v.ID()]; ok {
			instr.vs[i] = src
		}
	}
}

// entryBlk returns the entry block of the function.
func (b *builder) entryBlk() *basicBlock {
	return b.basicBlocksPool.View(0)
}

// isDominatedBy returns true if the given block `n` is dominated by the given block `d`.
// Before calling this, the builder must pass by passCalculateImmediateDominators.
func (b *builder) isDominatedBy(n *basicBlock, d *basicBlock) bool {
	if len(b.dominators) == 0 {
		panic("BUG: passCalculateImmediateDominators must be called before calling isDominatedBy")
	}
	ent := b.entryBlk()
	doms := b.dominators
	for n != d && n != ent { // 1 is assumed to be the root of the dominator tree
		n = doms[n.id]
	}
	return n == d
}
