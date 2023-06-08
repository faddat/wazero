package ssa

type Type byte

const (
	TypeInvalid Type = 1 + iota

	// TypeI8 represents an integer type with 8 bits.
	// TODO: do we need this?
	TypeI8

	// TypeI16 represents an integer type with 16 bits.
	// TODO: do we need this?
	TypeI16

	// TypeI32 represents an integer type with 32 bits.
	TypeI32

	// TypeI64 represents an integer type with 64 bits.
	TypeI64

	// TypeF32 represents 32-bit floats in the IEEE 754.
	TypeF32

	// TypeF64 represents 64-bit floats in the IEEE 754.
	TypeF64

	// TODO: SIMD, ref types!
)

// String implements fmt.Stringer.
func (t Type) String() (ret string) {
	switch t {
	case TypeI8:
		return "i8"
	case TypeI16:
		return "i16"
	case TypeI32:
		return "i32"
	case TypeI64:
		return "i64"
	case TypeF32:
		return "f32"
	case TypeF64:
		return "f64"
	}
	return
}
