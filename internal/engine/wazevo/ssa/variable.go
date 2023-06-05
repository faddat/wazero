package ssa

import "strconv"

// Variable is a unique identifier for a source program's variable and will correspond to
// multiple ssa Value(s).
//
// For example, `Local 1` is a Variable in WebAssembly, and Value(s) will be created for it
// whenever it executes `local.set 1`.
type Variable uint32

// String implements fmt.Stringer.
func (v Variable) String() string {
	return strconv.Itoa(int(v))
}
