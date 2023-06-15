package backend

import "math"

// vReg represents a register which is assigned to an SSA value. This is used to represent a register in the backend.
// A vReg may or may not be a physical register, and the info of physical register can be obtained by realReg.
// Note that a vReg can be assigned to multiple SSA values. Notably that means
// in the backend, we loosen the assumption of SSA.
type vReg uint64

// vRegID is the lower 32bit of vReg, which is the pure identifier of vReg without realReg info.
type vRegID uint32

// realReg returns the realReg of this vReg.
func (v *vReg) realReg() realReg {
	return realReg(*v >> 32)
}

// setRealReg sets the realReg of this vReg.
func (v *vReg) setRealReg(r realReg) {
	*v = vReg(r)<<32 | *v&0xffffffff
}

// id returns the vRegID of this vReg.
func (v *vReg) id() vRegID {
	return vRegID(*v & 0xffffffff)
}

// valid returns true if this vReg is valid.
func (v *vReg) valid() bool {
	return v.id() != vRegIDInvalid
}

// realReg represents a physical register.
type realReg uint32

const (
	vRegIDInvalid vRegID = math.MaxUint32
	vRegInvalid   vReg   = vReg(vRegIDInvalid)
)
