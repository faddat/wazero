package backend

import (
	"math"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

// VReg represents a register which is assigned to an SSA value. This is used to represent a register in the backend.
// A VReg may or may not be a physical register, and the info of physical register can be obtained by RealReg.
// Note that a VReg can be assigned to multiple SSA values. Notably that means
// in the backend, we loosen the assumption of SSA.
type VReg uint64

// VRegID is the lower 32bit of VReg, which is the pure identifier of VReg without RealReg info.
type VRegID uint32

// RealReg returns the RealReg of this VReg.
func (v VReg) RealReg() RealReg {
	return RealReg(v >> 32)
}

// SetRealReg sets the RealReg of this VReg and returns the updated VReg.
func (v VReg) SetRealReg(r RealReg) VReg {
	return VReg(r)<<32 | v&0xffffffff
}

// ID returns the VRegID of this VReg.
func (v VReg) ID() VRegID {
	return VRegID(v & 0xffffffff)
}

// Valid returns true if this VReg is Valid.
func (v VReg) Valid() bool {
	return v.ID() != vRegIDInvalid
}

// RealReg represents a physical register.
type RealReg byte

const RealRegInvalid = RealReg(0)

const (
	// vRegIDReservedBegin is the beginning of unreserved VRegID.
	// The reserved IDs are used for assigning fixed physical registers.
	// See VRegFromRealRegister.
	vRegIDUnreservedBegin        = 256
	vRegIDInvalid         VRegID = math.MaxUint32
	vRegInvalid                  = VReg(vRegIDInvalid)
)

// VRegFromRealRegister returns a VReg which is assigned to the given physical register.
func VRegFromRealRegister(real RealReg) VReg {
	// Use the literal value of RealReg as the VRegID, which will never collide with the VRegID of other normal VRegs
	// thanks to the vRegIDUnreservedBegin.
	ret := VReg(VRegID(real))
	ret = ret.SetRealReg(real)
	return ret
}

// RegType represents the type of a register.
type RegType byte

const (
	RegTypeInvalid = iota
	RegTypeInt
	RegTypeFloat
)

// RegTypeOf returns the RegType of the given ssa.Type.
func RegTypeOf(p ssa.Type) RegType {
	switch p {
	case ssa.TypeI32, ssa.TypeI64:
		return RegTypeInt
	case ssa.TypeF32, ssa.TypeF64:
		return RegTypeFloat
	default:
		panic(p)
	}
}
