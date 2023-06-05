package ssa

import "fmt"

// Value represents an SSA value.
type Value interface {
	String() string
}

// blockParam implements Value and represents a parameter to a basicBlock.
type blockParam struct {
	typ Type
	// n is the index of this blockParam in the bb.
	n int
	// bb is the basicBlock where this param exists.
	bb *basicBlock
}

// String implements Value.
func (p *blockParam) String() (ret string) {
	return fmt.Sprintf("%d(%s)", p.n, p.typ.String())
}

var (
	_ Value = (*blockParam)(nil)
	_ Value = (*Instruction)(nil)
)
