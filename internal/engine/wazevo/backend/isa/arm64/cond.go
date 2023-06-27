package arm64

import (
	"strconv"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/backend"
)

type (
	cond     uint64
	condKind byte
)

const (
	// condKindRegisterZero represents a condition which checks if the register is zero.
	// This indicates that the instruction must be encoded as CBZ:
	// https://developer.arm.com/documentation/ddi0596/2020-12/Base-Instructions/CBZ--Compare-and-Branch-on-Zero-
	condKindRegisterZero condKind = iota
	// This indicates that the instruction must be encoded as CBNZ:
	// https://developer.arm.com/documentation/ddi0596/2020-12/Base-Instructions/CBNZ--Compare-and-Branch-on-Nonzero-
	condKindRegisterNotZero
	// This indicates that the instruction must be encoded as B.cond:
	// https://developer.arm.com/documentation/ddi0596/2020-12/Base-Instructions/B-cond--Branch-conditionally-
	condKindCondFlagSet
)

// kind returns the kind of condition which is stored in the first two bits.
func (c cond) kind() condKind {
	return condKind(c & 0b11)
}

func (c cond) asUint64() uint64 {
	return uint64(c)
}

// register returns the register for register conditions.
// This panics if the condition is not a register condition (condKindRegisterZero or condKindRegisterNotZero).
func (c cond) register() backend.RealReg {
	if c.kind() != condKindRegisterZero && c.kind() != condKindRegisterNotZero {
		panic("condition is not a register")
	}
	return backend.RealReg(c >> 2)
}

func registerAsRegZeroCond(r backend.RealReg) cond {
	return cond(r<<2) | cond(condKindRegisterZero)
}

func registerAsRegNonZeroCond(r backend.RealReg) cond {
	return cond(r<<2) | cond(condKindRegisterNotZero)
}

func (c cond) flag() condFlag {
	if c.kind() != condKindCondFlagSet {
		panic("condition is not a flag")
	}
	return condFlag(c >> 2)
}

func (c condFlag) asCond() cond {
	return cond(c<<2) | cond(condKindCondFlagSet)
}

// condFlag represents a condition flag for conditional branches.
type condFlag uint8

const (
	eq condFlag = iota // eq represents "equal"
	ne                 // ne represents "not equal"
	hs                 // hs represents "higher or same"
	lo                 // lo represents "lower"
	mi                 // mi represents "minus or negative result"
	pl                 // pl represents "plus or positive result"
	vs                 // vs represents "overflow set"
	vc                 // vc represents "overflow clear"
	hi                 // hi represents "higher"
	ls                 // ls represents "lower or same"
	ge                 // ge represents "greater or equal"
	lt                 // lt represents "less than"
	gt                 // gt represents "greater than"
	le                 // le represents "less than or equal"
	al                 // al represents "always"
	nv                 // nv represents "never"
)

// invert returns the inverted condition.
func (c condFlag) invert() condFlag {
	switch c {
	case eq:
		return ne
	case ne:
		return eq
	case hs:
		return lo
	case lo:
		return hs
	case mi:
		return pl
	case pl:
		return mi
	case vs:
		return vc
	case vc:
		return vs
	case hi:
		return ls
	case ls:
		return hi
	case ge:
		return lt
	case lt:
		return ge
	case gt:
		return le
	case le:
		return gt
	case al:
		return nv
	case nv:
		return al
	default:
		panic(c)
	}
}

// String implements fmt.Stringer.
func (c condFlag) String() string {
	switch c {
	case eq:
		return "eq"
	case ne:
		return "ne"
	case hs:
		return "hs"
	case lo:
		return "lo"
	case mi:
		return "mi"
	case pl:
		return "pl"
	case vs:
		return "vs"
	case vc:
		return "vc"
	case hi:
		return "hi"
	case ls:
		return "ls"
	case ge:
		return "ge"
	case lt:
		return "lt"
	case gt:
		return "gt"
	case le:
		return "le"
	case al:
		return "al"
	case nv:
		return "nv"
	default:
		panic(strconv.Itoa(int(c)))
	}
}
