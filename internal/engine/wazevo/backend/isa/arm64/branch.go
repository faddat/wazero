package arm64

import "strconv"

type (
	// branchTarget is a target for un/conditional branching.
	// Its first bit is used to distinguish between a label and a relative offset.
	branchTarget     uint64
	branchTargetKind uint8
)

const (
	branchTargetKindLabel = iota
	branchTargetKindOffset
)

// kind returns the kind of branch target.
func (b branchTarget) kind() branchTargetKind {
	return branchTargetKind(b & 0b1)
}

func (b branchTarget) label() label {
	if b.kind() != branchTargetKindLabel {
		panic("branch target is not a label")
	}
	return label(b >> 1)
}

func (l label) asBranchTarget() branchTarget {
	return branchTarget(l<<1) | branchTarget(branchTargetKindLabel)
}

func (b branchTarget) offset() int64 {
	if b.kind() != branchTargetKindOffset {
		panic("branch target is not an offset")
	}
	return int64(b >> 1)
}

func (b branchTarget) String() string {
	switch b.kind() {
	case branchTargetKindLabel:
		return b.label().String()
	case branchTargetKindOffset:
		return strconv.FormatInt(b.offset(), 10)
	default:
		panic("invalid branch target kind")
	}
}

func (b branchTarget) asUint64() uint64 {
	return uint64(b)
}
