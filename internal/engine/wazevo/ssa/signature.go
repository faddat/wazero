package ssa

import (
	"fmt"
	"strings"
)

// Signature is a function prototype.
type Signature struct {
	ID              SignatureID
	Params, Results []Type

	// used is true if this is used by the currently-compiled function.
	// Debugging only.
	used bool
}

// String implements fmt.Stringer.
func (s *Signature) String() string {
	str := strings.Builder{}
	str.WriteString(s.ID.String())
	str.WriteString(": ")
	for _, typ := range s.Params {
		str.WriteString(typ.String())
	}
	str.WriteByte('_')
	for _, typ := range s.Results {
		str.WriteString(typ.String())
	}
	return str.String()
}

// SignatureID is an unique identifier used to lookup.
type SignatureID int

// String implements fmt.Stringer.
func (s SignatureID) String() string {
	return fmt.Sprintf("sig%d", s)
}