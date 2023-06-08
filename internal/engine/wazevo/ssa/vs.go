package ssa

import (
	"fmt"
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

// Value represents an SSA value. The relationship with Variable is 1: N (including 0),
// that means there might be multiple Variable(s) for a Value.
type Value uint32

const valueInvalid = 0

// String implements fmt.Stringer.
func (v Value) String() string {
	return fmt.Sprintf("v%d", v)
}

// Valid returns true if this value is valid.
// TODO: needs to be exported?
func (v Value) Valid() bool {
	return v != valueInvalid
}
