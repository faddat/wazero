package ssa

import (
	"fmt"
	"math"
)

// Variable is a unique identifier for a source program's variable and will correspond to
// multiple ssa Value(s).
//
// For example, `Local 1` is a Variable in WebAssembly, and Value(s) will be created for it
// whenever it executes `local.set 1`.
type Variable uint32

// String implements fmt.Stringer.
func (v Variable) String() string {
	return fmt.Sprintf("var%d", v)
}

// Value represents an SSA value with a type information. The relationship with Variable is 1: N (including 0),
// that means there might be multiple Variable(s) for a Value.
//
// Higher 32-bit is used to store Type for this value.
type Value uint64

// valueID is the lower 32bit of Value, which is the pure identifier of Value without type info.
type valueID uint32

const valueIDInvalid valueID = math.MaxUint32
const valueInvalid Value = Value(valueIDInvalid)

// Format creates a debug string for this Value using the data stored in Builder.
func (v *Value) format(b Builder) string {
	if annotation, ok := b.(*builder).valueAnnotations[v.id()]; ok {
		return annotation
	}
	return fmt.Sprintf("v%d", v.id())
}

func (v *Value) formatWithType(b Builder) string {
	if annotation, ok := b.(*builder).valueAnnotations[v.id()]; ok {
		return annotation + ":" + v._Type().String()
	} else {
		return fmt.Sprintf("v%d:%s", v.id(), v._Type())
	}
}

// valid returns true if this value is valid.
func (v *Value) valid() bool {
	return v.id() != valueIDInvalid
}

// _Type returns the Type of this value.
func (v *Value) _Type() Type {
	return Type(*v >> 32)
}

// id returns the valueID of this value.
func (v *Value) id() valueID {
	return valueID(*v & 0xffffffff)
}

// setType sets a type of this Value.
func (v *Value) setType(typ Type) {
	*v |= Value(typ) << 32
}

// valueAlias holds the information to alias the source Value to the destination Value.
// Aliases are needed during the optimizations, where we remove/modify the BasicBlock and Instruction(s).
type valueAlias struct {
	src, dst Value
}

func (va valueAlias) format(b Builder) string {
	return fmt.Sprintf("%s = %s", va.dst.format(b), va.src.format(b))
}
