package backend

import "math"

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

// setRealReg sets the RealReg of this VReg and returns the updated VReg.
func (v VReg) setRealReg(r RealReg) VReg {
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
type RealReg uint16

const (
	vRegIDInvalid VRegID = math.MaxUint32
	vRegInvalid   VReg   = VReg(vRegIDInvalid)
)
